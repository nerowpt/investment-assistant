package library

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/investment-assistant/investment-assistant/internal/core/account"
	"github.com/investment-assistant/investment-assistant/internal/core/ids"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore/schema"
	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
)

const candidateTTLDays = 180

// Service L1 归纳业务入口。
type Service struct {
	ac   *account.Context
	db   *sql.DB
	tags *yamlstore.ControlledTags
}

// NewService 构造 library 服务（须已 migrate）。
func NewService(ac *account.Context, db *sql.DB) (*Service, error) {
	tags, err := yamlstore.LoadControlledTags(ac.ControlledTagsPath())
	if err != nil {
		return nil, fmt.Errorf("读取 controlled_tags: %w", err)
	}
	return &Service{ac: ac, db: db, tags: tags}, nil
}

// IngestInput 主动录入参数。
type IngestInput struct {
	URL         string
	FilePath    string
	Text        string
	Title       string
	Source      string
	Tier        string
	Stocks      []string
	ContentType string
	MediaType   string
	AutoDismiss bool // exact 时自动 dismiss
}

// IngestResult 录入结果。
type IngestResult struct {
	CandidateID string
	Status      string
	MatchTier   string
	DedupKey    string
	AutoAction  string
}

// DraftInput crawl / CoreIngest 回调写入 candidate。
type DraftInput struct {
	SourceEntry  string
	Title        string
	ContentType  string
	MediaType    string
	Stocks       []string
	CanonicalURL string
	SummaryDraft string
	ExtractJSON  string
	Tier         string
	Source       string
	AutoDismiss  bool
}

// StageFromDraft 由 CoreIngest 或 crawl 写入 candidate（不经过 CLI ingest 三选一）。
func (s *Service) StageFromDraft(in DraftInput) (*IngestResult, error) {
	text := ""
	if in.ExtractJSON != "" {
		text = in.ExtractJSON
	} else if in.SummaryDraft != "" {
		text = in.SummaryDraft
	}
	url := in.CanonicalURL
	if url == "" && text == "" {
		return nil, fmt.Errorf("draft 须含 canonical_url 或 extract/summary")
	}
	core := IngestInput{
		Title:       in.Title,
		Source:      in.Source,
		Tier:        in.Tier,
		Stocks:      in.Stocks,
		ContentType: in.ContentType,
		MediaType:   in.MediaType,
		AutoDismiss: in.AutoDismiss,
	}
	if url != "" {
		core.URL = url
	} else {
		core.Text = text
	}
	return s.ingestCore(core, in.SourceEntry)
}

// Ingest 主动录入 → library_candidates（03 §10C.3）。
func (s *Service) Ingest(in IngestInput) (*IngestResult, error) {
	return s.ingestCore(in, "manual")
}

// QuickAddInput 向导内快速录入 L1 素材（ingest + promote 一步完成）。
type QuickAddInput struct {
	Title string // 素材标题
	Text  string // 正文/摘要
	Stock string // 关联标的 code
	Tier  string // 默认 B
}

// QuickAdd 快速录入并晋升为 active library_item（H8 前端「新增素材」）。
func (s *Service) QuickAdd(in QuickAddInput) (string, error) {
	tier := in.Tier
	if tier == "" {
		tier = "B"
	}
	res, err := s.Ingest(IngestInput{
		Text:   in.Text,
		Title:  in.Title,
		Tier:   tier,
		Stocks: []string{in.Stock},
	})
	if err != nil {
		return "", err
	}
	return s.Promote(PromoteInput{
		CandidateID: res.CandidateID,
		ContentType: "note",
		MediaType:   "text",
		Tier:        tier,
		Stocks:      []string{in.Stock},
		Summary:     in.Text,
	})
}

func (s *Service) ingestCore(in IngestInput, sourceEntry string) (*IngestResult, error) {
	if err := validateIngestCore(in); err != nil {
		return nil, err
	}
	now := nowISO()
	stocksJSON, _ := json.Marshal(in.Stocks)
	tier := in.Tier
	if tier == "" {
		tier = "B"
	}
	title := strings.TrimSpace(in.Title)
	source := strings.TrimSpace(in.Source)
	if source == "" {
		source = "manual"
	}

	var (
		canonicalURL string
		stagingRel   string
		fileSHA      string
		extractJSON  string
		mediaType    = in.MediaType
	)

	year := time.Now().Format("2006")
	lcID, err := ids.Next(s.db, "lc")
	if err != nil {
		return nil, err
	}

	if in.URL != "" {
		canonicalURL = strings.TrimSpace(in.URL)
		if title == "" {
			title = canonicalURL
		}
		if mediaType == "" {
			mediaType = "html"
		}
	} else if in.FilePath != "" {
		abs, sha, ext, err := stageFile(s.ac.LibraryDir, year, lcID, in.FilePath)
		if err != nil {
			return nil, err
		}
		fileSHA = sha
		stagingRel = relLibraryPath(year, lcID, ext)
		_ = abs
		if title == "" {
			title = filepath.Base(in.FilePath)
		}
		if mediaType == "" {
			mediaType = mediaFromExt(ext)
		}
	} else {
		body := map[string]string{"body": in.Text}
		b, _ := json.Marshal(body)
		extractJSON = string(b)
		if title == "" {
			title = truncate(in.Text, 40)
		}
		if mediaType == "" {
			mediaType = "text"
		}
	}

	primaryStock := ""
	if len(in.Stocks) > 0 {
		primaryStock = in.Stocks[0]
	}
	dedupIn := DedupInput{
		FileSHA256:   fileSHA,
		CanonicalURL: canonicalURL,
		Source:       source,
		Title:        title,
		Timestamp:    now,
		PrimaryStock: primaryStock,
	}
	dedupKey := ComputeDedupKey(dedupIn)

	if pendingID, err := sqlstore.FindPendingCandidateByDedupKey(s.db, dedupKey); err != nil {
		return nil, err
	} else if pendingID != "" {
		return nil, fmt.Errorf("已存在 pending 候选 %s（dedup_key 相同）", pendingID)
	}

	exactItemID, err := sqlstore.FindItemIDByDedupKey(s.db, dedupKey)
	if err != nil {
		return nil, err
	}
	if exactItemID != "" && in.AutoDismiss {
		return &IngestResult{
			CandidateID: pendingIDFromDedup(s.db, dedupKey),
			Status:      "dismissed",
			MatchTier:   "exact",
			DedupKey:    dedupKey,
			AutoAction:  "auto_dismiss_exact",
		}, nil
	}

	if existID, existStatus, err := sqlstore.FindCandidateByDedupKey(s.db, dedupKey); err != nil {
		return nil, err
	} else if existID != "" && in.AutoDismiss {
		return &IngestResult{
			CandidateID: existID,
			Status:      "dismissed",
			MatchTier:   "exact",
			DedupKey:    dedupKey,
			AutoAction:  "auto_dismiss_exact",
		}, nil
	} else if existID != "" {
		return nil, fmt.Errorf("dedup_key 已存在于 candidate %s（status=%s）", existID, existStatus)
	}

	sim := s.analyze(dedupIn, in.Stocks, exactItemID)
	expires := time.Now().AddDate(0, 0, candidateTTLDays).Format(time.RFC3339)

	status := "pending"
	resolution := "pending"
	autoAction := ""
	dismissReason := ""

	if sim.MatchTier == "exact" || exactItemID != "" {
		if in.AutoDismiss {
			status = "dismissed"
			resolution = "dismiss"
			autoAction = "auto_dismiss_exact"
			dismissReason = "exact_duplicate"
		}
	}

	candidate := &schema.LibraryCandidate{
		ID:                lcID,
		Status:            status,
		SourceEntry:       sourceEntry,
		Title:            title,
		Source:           source,
		Tier:             tier,
		Timestamp:        now,
		ContentType:      in.ContentType,
		MediaType:        mediaType,
		RelatedStocksJSON: string(stocksJSON),
		TagsJSON:         "[]",
		DedupKey:         dedupKey,
		StagingPath:      stagingRel,
		CanonicalURL:     canonicalURL,
		ExtractJSON:      extractJSON,
		SimilarityJSON:   sim.JSON(),
		MatchTier:        sim.MatchTier,
		Resolution:       resolution,
		ExpiresAt:        expires,
		DismissedReason:  dismissReason,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := sqlstore.InsertLibraryCandidate(s.db, candidate); err != nil {
		return nil, fmt.Errorf("写入 candidate: %w", err)
	}

	return &IngestResult{
		CandidateID: lcID,
		Status:      status,
		MatchTier:   sim.MatchTier,
		DedupKey:    dedupKey,
		AutoAction:  autoAction,
	}, nil
}

// PromoteInput promote_new 参数。
type PromoteInput struct {
	CandidateID string
	ContentType string
	MediaType   string
	Tier        string
	Tags        []string
	Stocks      []string
	Summary     string
}

// Promote 候选 → 新 library_item（03 §9.9）。
func (s *Service) Promote(in PromoteInput) (string, error) {
	c, err := sqlstore.GetLibraryCandidate(s.db, in.CandidateID)
	if err != nil {
		return "", err
	}
	if c.Status != "pending" {
		return "", fmt.Errorf("candidate status 须为 pending，当前: %s", c.Status)
	}
	contentType := in.ContentType
	if contentType == "" {
		contentType = c.ContentType
	}
	mediaType := in.MediaType
	if mediaType == "" {
		mediaType = c.MediaType
	}
	tier := in.Tier
	if tier == "" {
		tier = c.Tier
	}
	if contentType == "" || mediaType == "" || tier == "" {
		return "", fmt.Errorf("content_type / media_type / tier 必填")
	}
	if !slices.Contains(schema.MVP1ContentTypes, contentType) {
		return "", fmt.Errorf("content_type 非法: %s", contentType)
	}
	if !slices.Contains(schema.MVP1MediaTypes, mediaType) {
		return "", fmt.Errorf("media_type 非法: %s", mediaType)
	}

	tags := in.Tags
	if len(tags) == 0 {
		tags = sqlstore.ParseJSONStringArray(c.TagsJSON)
	}
	if err := s.tags.ValidateTagIDs(tags); err != nil {
		return "", err
	}
	tagsJSON, _ := json.Marshal(tags)

	stocks := in.Stocks
	if len(stocks) == 0 {
		stocks = sqlstore.ParseJSONStringArray(c.RelatedStocksJSON)
	}
	stocksJSON, _ := json.Marshal(stocks)

	now := nowISO()
	libID, err := ids.Next(s.db, "lib")
	if err != nil {
		return "", err
	}
	assetID, err := ids.Next(s.db, "la")
	if err != nil {
		return "", err
	}

	finalPath, err := s.finalizeAssetPath(c, libID, mediaType)
	if err != nil {
		return "", err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	item := &schema.LibraryItem{
		ID:                      libID,
		Status:                  "active",
		Title:                   c.Title,
		Source:                  c.Source,
		Tier:                    tier,
		Timestamp:               c.Timestamp,
		CollectedAt:             now,
		Author:                  c.Author,
		ContentType:             contentType,
		MediaType:               mediaType,
		RelatedStocksJSON:       string(stocksJSON),
		TagsJSON:                string(tagsJSON),
		DedupKey:                c.DedupKey,
		CanonicalURL:            c.CanonicalURL,
		PrimaryAssetID:          assetID,
		SummaryByUser:           pickSummary(in.Summary, c.SummaryDraft),
		PromotedFromCandidateID: c.ID,
		SchemaVersion:           1,
		ReferenceCount:          0,
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	if err := insertLibraryItemTx(tx, item); err != nil {
		return "", err
	}

	asset := &schema.LibraryItemAsset{
		ID:                      assetID,
		LibraryItemID:           libID,
		AssetRole:               "primary",
		Source:                  c.Source,
		Tier:                    tier,
		Timestamp:               c.Timestamp,
		FilePath:                finalPath,
		FileSHA256:              fileSHAFromDedup(c.DedupKey),
		CanonicalURL:            c.CanonicalURL,
		PromotedFromCandidateID: c.ID,
		CreatedAt:               now,
	}
	if err := insertLibraryAssetTx(tx, asset); err != nil {
		return "", err
	}

	if err := updateCandidateStatusTx(tx, c.ID, "promoted", "promote_new", "", libID, "", now); err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}
	return libID, nil
}

// SupplementInput supplement 参数。
type SupplementInput struct {
	CandidateID string
	IntoItemID  string
	Note        string
	TagsAdd     []string
	TagsRemove  []string
}

// Supplement 候选追加到已有 item。
func (s *Service) Supplement(in SupplementInput) error {
	c, err := sqlstore.GetLibraryCandidate(s.db, in.CandidateID)
	if err != nil {
		return err
	}
	if c.Status != "pending" {
		return fmt.Errorf("candidate status 须为 pending，当前: %s", c.Status)
	}
	target, err := sqlstore.GetLibraryItem(s.db, in.IntoItemID)
	if err != nil {
		return err
	}
	if target.Status != "active" {
		return fmt.Errorf("目标 item 须为 active，当前: %s", target.Status)
	}

	baseTags := sqlstore.ParseJSONStringArray(target.TagsJSON)
	candTags := sqlstore.ParseJSONStringArray(c.TagsJSON)
	merged := yamlstore.MergeTags(baseTags, yamlstore.MergeTags(candTags, in.TagsAdd))
	if len(in.TagsRemove) > 0 {
		remove := map[string]struct{}{}
		for _, id := range in.TagsRemove {
			remove[id] = struct{}{}
		}
		var filtered []string
		for _, id := range merged {
			if _, ok := remove[id]; !ok {
				filtered = append(filtered, id)
			}
		}
		merged = filtered
	}
	if err := s.tags.ValidateTagIDs(merged); err != nil {
		return err
	}
	tagsJSON, _ := json.Marshal(merged)

	now := nowISO()
	assetID, err := ids.Next(s.db, "la")
	if err != nil {
		return err
	}
	finalPath, err := s.finalizeAssetPath(c, target.ID, target.MediaType)
	if err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	asset := &schema.LibraryItemAsset{
		ID:                      assetID,
		LibraryItemID:           target.ID,
		AssetRole:               "supplement",
		Source:                  c.Source,
		Tier:                    c.Tier,
		Timestamp:               c.Timestamp,
		FilePath:                finalPath,
		FileSHA256:              fileSHAFromDedup(c.DedupKey),
		CanonicalURL:            c.CanonicalURL,
		PromotedFromCandidateID: c.ID,
		SupplementNote:          in.Note,
		CreatedAt:               now,
	}
	if err := insertLibraryAssetTx(tx, asset); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE library_items SET tags_json = ?, updated_at = ? WHERE id = ?`,
		string(tagsJSON), now, target.ID); err != nil {
		return err
	}
	if err := updateCandidateStatusTx(tx, c.ID, "promoted", "supplement", target.ID, target.ID, "", now); err != nil {
		return err
	}
	return tx.Commit()
}

// Dismiss 丢弃候选。
func (s *Service) Dismiss(candidateID, reason string) error {
	c, err := sqlstore.GetLibraryCandidate(s.db, candidateID)
	if err != nil {
		return err
	}
	if c.Status != "pending" {
		return fmt.Errorf("candidate status 须为 pending，当前: %s", c.Status)
	}
	now := nowISO()
	return sqlstore.UpdateCandidateStatus(s.db, candidateID, "dismissed", "dismiss", "", "", reason, now)
}

// ExpireCandidates TTL 过期 job。
func (s *Service) ExpireCandidates(dryRun bool) (int, error) {
	return sqlstore.ExpireCandidates(s.db, nowISO(), dryRun)
}

// ArchiveItem 归档 library_item。
func (s *Service) ArchiveItem(itemID string) error {
	item, err := sqlstore.GetLibraryItem(s.db, itemID)
	if err != nil {
		return err
	}
	if item.Status != "active" {
		return fmt.Errorf("仅 active item 可 archive，当前: %s", item.Status)
	}
	return sqlstore.ArchiveLibraryItem(s.db, itemID, nowISO())
}

func (s *Service) analyze(in DedupInput, stocks []string, exactItemID string) SimilarityResult {
	items, err := sqlstore.ListActiveItemsForSimilarity(s.db)
	if err != nil {
		return SimilarityResult{MatchTier: "none", AnalyzedAt: nowISO()}
	}
	var existing []ItemForSimilarity
	for _, it := range items {
		existing = append(existing, ItemForSimilarity{
			ID:            it.ID,
			Title:         it.Title,
			RelatedStocks: sqlstore.ParseJSONStringArray(it.RelatedStocksJSON),
			Tags:          sqlstore.ParseJSONStringArray(it.TagsJSON),
			DedupKey:      it.DedupKey,
		})
	}
	return AnalyzeSimilarity(in, stocks, existing, exactItemID)
}

func validateIngestSource(in IngestInput) error {
	n := 0
	if in.URL != "" {
		n++
	}
	if in.FilePath != "" {
		n++
	}
	if in.Text != "" {
		n++
	}
	if n != 1 {
		return fmt.Errorf("须且仅能指定 --url / --file / --text 之一")
	}
	return nil
}

func validateIngestCore(in IngestInput) error {
	if in.FilePath != "" {
		if in.URL != "" || in.Text != "" {
			return fmt.Errorf("file 不能与 url/text 同时使用")
		}
		return nil
	}
	if in.URL != "" && in.Text != "" {
		return fmt.Errorf("url 与 text 不能同时使用")
	}
	if in.URL == "" && in.Text == "" {
		return fmt.Errorf("须指定 url 或 text 或 file")
	}
	return nil
}

func stageFile(libraryRoot, year, lcID, src string) (absPath, sha256hex, ext string, err error) {
	raw, err := os.ReadFile(src)
	if err != nil {
		return "", "", "", fmt.Errorf("读取文件: %w", err)
	}
	sum := sha256.Sum256(raw)
	sha256hex = hex.EncodeToString(sum[:])
	ext = filepath.Ext(src)
	if ext == "" {
		ext = ".bin"
	}
	inboxDir := filepath.Join(libraryRoot, "inbox", year)
	if err := os.MkdirAll(inboxDir, 0o755); err != nil {
		return "", "", "", err
	}
	name := lcID + ext
	absPath = filepath.Join(inboxDir, name)
	if err := writeFileAtomic(absPath, raw); err != nil {
		return "", "", "", err
	}
	return absPath, sha256hex, ext, nil
}

func relLibraryPath(year, id, ext string) string {
	return filepath.ToSlash(filepath.Join("inbox", year, id+ext))
}

func (s *Service) finalizeAssetPath(c *schema.LibraryCandidate, libID, mediaType string) (string, error) {
	if c.StagingPath == "" {
		return "", nil
	}
	src := filepath.Join(s.ac.LibraryDir, filepath.FromSlash(c.StagingPath))
	if _, err := os.Stat(src); err != nil {
		return c.StagingPath, nil
	}
	year := time.Now().Format("2006")
	subdir := mediaSubdir(mediaType)
	ext := filepath.Ext(src)
	if ext == "" {
		ext = filepath.Ext(c.StagingPath)
	}
	destRel := filepath.ToSlash(filepath.Join(subdir, year, libID+ext))
	destAbs := filepath.Join(s.ac.LibraryDir, filepath.FromSlash(destRel))
	if err := os.MkdirAll(filepath.Dir(destAbs), 0o755); err != nil {
		return "", err
	}
	if err := os.Rename(src, destAbs); err != nil {
		return "", fmt.Errorf("移动素材文件: %w", err)
	}
	return destRel, nil
}

func mediaSubdir(mediaType string) string {
	switch mediaType {
	case "pdf":
		return "docs"
	case "html":
		return "snapshots"
	case "structured":
		return "uploads"
	default:
		return "media"
	}
}

func mediaFromExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".pdf":
		return "pdf"
	case ".html", ".htm":
		return "html"
	case ".csv", ".json":
		return "structured"
	default:
		return "text"
	}
}

func fileSHAFromDedup(dedupKey string) string {
	if strings.HasPrefix(dedupKey, "sha256:") {
		return strings.TrimPrefix(dedupKey, "sha256:")
	}
	return ""
}

func pickSummary(user, draft string) string {
	if strings.TrimSpace(user) != "" {
		return user
	}
	return draft
}

func truncate(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}

func nowISO() string {
	return time.Now().Format(time.RFC3339)
}

func writeFileAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func insertLibraryItemTx(tx *sql.Tx, item *schema.LibraryItem) error {
	_, err := tx.Exec(`
		INSERT INTO library_items (
			id, status, title, source, tier, timestamp, collected_at, author,
			content_type, media_type, related_stocks_json, tags_json, dedup_key,
			canonical_url, cluster_id, primary_asset_id, summary_by_user, user_notes,
			promoted_from_candidate_id, merged_into_id, duplicate_of_id, schema_version,
			reference_count, last_referenced_at, archived_at, created_at, updated_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		item.ID, item.Status, item.Title, item.Source, item.Tier, item.Timestamp, item.CollectedAt,
		sqlstoreNull(item.Author), item.ContentType, item.MediaType, item.RelatedStocksJSON, item.TagsJSON, item.DedupKey,
		sqlstoreNull(item.CanonicalURL), sqlstoreNull(item.ClusterID), sqlstoreNull(item.PrimaryAssetID),
		sqlstoreNull(item.SummaryByUser), sqlstoreNull(item.UserNotes), sqlstoreNull(item.PromotedFromCandidateID),
		sqlstoreNull(item.MergedIntoID), sqlstoreNull(item.DuplicateOfID), item.SchemaVersion,
		item.ReferenceCount, sqlstoreNull(item.LastReferencedAt), sqlstoreNull(item.ArchivedAt),
		item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func insertLibraryAssetTx(tx *sql.Tx, a *schema.LibraryItemAsset) error {
	_, err := tx.Exec(`
		INSERT INTO library_item_assets (
			id, library_item_id, asset_role, source, tier, timestamp,
			file_path, file_sha256, canonical_url, promoted_from_candidate_id,
			supplement_note, created_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		a.ID, a.LibraryItemID, a.AssetRole, a.Source, a.Tier, a.Timestamp,
		sqlstoreNull(a.FilePath), sqlstoreNull(a.FileSHA256), sqlstoreNull(a.CanonicalURL),
		sqlstoreNull(a.PromotedFromCandidateID), sqlstoreNull(a.SupplementNote), a.CreatedAt,
	)
	return err
}

func updateCandidateStatusTx(tx *sql.Tx, id, status, resolution, target, promoted, reason, updatedAt string) error {
	_, err := tx.Exec(`
		UPDATE library_candidates SET
			status = ?, resolution = ?, resolution_target_item_id = ?,
			promoted_library_item_id = ?, dismissed_reason = ?, updated_at = ?
		WHERE id = ?`,
		status, sqlstoreNull(resolution), sqlstoreNull(target),
		sqlstoreNull(promoted), sqlstoreNull(reason), updatedAt, id,
	)
	return err
}

func sqlstoreNull(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// pendingIDFromDedup 返回已有 candidate id（若存在）。
func pendingIDFromDedup(db *sql.DB, dedupKey string) string {
	id, _, _ := sqlstore.FindCandidateByDedupKey(db, dedupKey)
	return id
}

// ReloadTags 重新加载 controlled_tags（tags CLI 修改后）。
func (s *Service) ReloadTags() error {
	t, err := yamlstore.LoadControlledTags(s.ac.ControlledTagsPath())
	if err != nil {
		return err
	}
	s.tags = t
	return nil
}

// ControlledTags 返回当前词表（CLI 用）。
func (s *Service) ControlledTags() *yamlstore.ControlledTags {
	return s.tags
}

// Unused import guard for io
var _ = io.Discard
