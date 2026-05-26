package sqlstore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore/schema"
)

// LibraryCandidateRow 列表/详情用候选行。
type LibraryCandidateRow struct {
	ID                     string
	Status                 string
	Title                  string
	Source                 string
	Tier                   string
	MatchTier              string
	DedupKey               string
	StagingPath            string
	SimilarityJSON         string
	ExpiresAt              string
	PromotedLibraryItemID  string
	DismissedReason        string
	CreatedAt              string
}

// LibraryItemRow 列表/详情用素材行。
type LibraryItemRow struct {
	ID                      string
	Status                  string
	Title                   string
	Source                  string
	Tier                    string
	ContentType             string
	MediaType               string
	RelatedStocksJSON       string
	TagsJSON                string
	DedupKey                string
	SummaryByUser           string
	PromotedFromCandidateID string
	CreatedAt               string
}

// FindItemIDByDedupKey 按 dedup_key 查 active library_item。
func FindItemIDByDedupKey(db *sql.DB, dedupKey string) (string, error) {
	var id string
	err := db.QueryRow(
		`SELECT id FROM library_items WHERE dedup_key = ? AND status = 'active' LIMIT 1`,
		dedupKey,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return id, nil
}

// FindPendingCandidateByDedupKey 查 pending 候选。
func FindPendingCandidateByDedupKey(db *sql.DB, dedupKey string) (string, error) {
	var id string
	err := db.QueryRow(
		`SELECT id FROM library_candidates WHERE dedup_key = ? AND status = 'pending' LIMIT 1`,
		dedupKey,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return id, nil
}

// FindCandidateByDedupKey 查任意状态候选（dedup_key UNIQUE）。
func FindCandidateByDedupKey(db *sql.DB, dedupKey string) (id, status string, err error) {
	err = db.QueryRow(
		`SELECT id, status FROM library_candidates WHERE dedup_key = ? LIMIT 1`,
		dedupKey,
	).Scan(&id, &status)
	if err == sql.ErrNoRows {
		return "", "", nil
	}
	return id, status, err
}

// ListActiveItemsForSimilarity 加载 active items 供相似度分析。
func ListActiveItemsForSimilarity(db *sql.DB) ([]schema.LibraryItem, error) {
	rows, err := db.Query(`
		SELECT id, title, related_stocks_json, tags_json, dedup_key
		FROM library_items WHERE status = 'active'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []schema.LibraryItem
	for rows.Next() {
		var item schema.LibraryItem
		if err := rows.Scan(&item.ID, &item.Title, &item.RelatedStocksJSON, &item.TagsJSON, &item.DedupKey); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// InsertLibraryCandidate 写入候选。
func InsertLibraryCandidate(db *sql.DB, c *schema.LibraryCandidate) error {
	_, err := db.Exec(`
		INSERT INTO library_candidates (
			id, status, source_entry, title, source, tier, timestamp, author,
			content_type, media_type, related_stocks_json, tags_json, dedup_key,
			staging_path, canonical_url, extract_json, summary_draft, similarity_json,
			match_tier, resolution, resolution_target_item_id, expires_at,
			promoted_library_item_id, dismissed_reason, created_at, updated_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		c.ID, c.Status, c.SourceEntry, c.Title, c.Source, c.Tier, c.Timestamp, c.Author,
		nullIfEmpty(c.ContentType), nullIfEmpty(c.MediaType), c.RelatedStocksJSON, c.TagsJSON, c.DedupKey,
		nullIfEmpty(c.StagingPath), nullIfEmpty(c.CanonicalURL), nullIfEmpty(c.ExtractJSON), nullIfEmpty(c.SummaryDraft),
		nullIfEmpty(c.SimilarityJSON), nullIfEmpty(c.MatchTier), nullIfEmpty(c.Resolution), nullIfEmpty(c.ResolutionTargetItemID),
		c.ExpiresAt, nullIfEmpty(c.PromotedLibraryItemID), nullIfEmpty(c.DismissedReason), c.CreatedAt, c.UpdatedAt,
	)
	return err
}

// GetLibraryCandidate 读取单个候选。
func GetLibraryCandidate(db *sql.DB, id string) (*schema.LibraryCandidate, error) {
	row := db.QueryRow(`
		SELECT id, status, source_entry, title, source, tier, timestamp,
			COALESCE(author,''), COALESCE(content_type,''), COALESCE(media_type,''),
			COALESCE(related_stocks_json,'[]'), COALESCE(tags_json,'[]'), dedup_key,
			COALESCE(staging_path,''), COALESCE(canonical_url,''), COALESCE(extract_json,''),
			COALESCE(summary_draft,''), COALESCE(similarity_json,''), COALESCE(match_tier,''),
			COALESCE(resolution,''), COALESCE(resolution_target_item_id,''), expires_at,
			COALESCE(promoted_library_item_id,''), COALESCE(dismissed_reason,''),
			created_at, updated_at
		FROM library_candidates WHERE id = ?`, id)

	var c schema.LibraryCandidate
	err := row.Scan(
		&c.ID, &c.Status, &c.SourceEntry, &c.Title, &c.Source, &c.Tier, &c.Timestamp,
		&c.Author, &c.ContentType, &c.MediaType, &c.RelatedStocksJSON, &c.TagsJSON, &c.DedupKey,
		&c.StagingPath, &c.CanonicalURL, &c.ExtractJSON, &c.SummaryDraft, &c.SimilarityJSON, &c.MatchTier,
		&c.Resolution, &c.ResolutionTargetItemID, &c.ExpiresAt,
		&c.PromotedLibraryItemID, &c.DismissedReason, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("候选不存在: %s", id)
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ListLibraryCandidates 按 status 列表。
func ListLibraryCandidates(db *sql.DB, status, matchTier string) ([]LibraryCandidateRow, error) {
	q := `SELECT id, status, title, source, tier, COALESCE(match_tier,''), dedup_key,
		COALESCE(staging_path,''), COALESCE(similarity_json,''), expires_at,
		COALESCE(promoted_library_item_id,''), COALESCE(dismissed_reason,''), created_at
		FROM library_candidates WHERE 1=1`
	var args []any
	if status != "" {
		q += ` AND status = ?`
		args = append(args, status)
	}
	if matchTier != "" {
		q += ` AND match_tier = ?`
		args = append(args, matchTier)
	}
	q += ` ORDER BY created_at DESC`

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LibraryCandidateRow
	for rows.Next() {
		var r LibraryCandidateRow
		if err := rows.Scan(
			&r.ID, &r.Status, &r.Title, &r.Source, &r.Tier, &r.MatchTier, &r.DedupKey,
			&r.StagingPath, &r.SimilarityJSON, &r.ExpiresAt,
			&r.PromotedLibraryItemID, &r.DismissedReason, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// UpdateCandidateStatus 更新候选状态与 resolution 字段。
func UpdateCandidateStatus(db *sql.DB, id, status, resolution, targetItemID, promotedID, reason, updatedAt string) error {
	_, err := db.Exec(`
		UPDATE library_candidates SET
			status = ?, resolution = ?, resolution_target_item_id = ?,
			promoted_library_item_id = ?, dismissed_reason = ?, updated_at = ?
		WHERE id = ?`,
		status, nullIfEmpty(resolution), nullIfEmpty(targetItemID),
		nullIfEmpty(promotedID), nullIfEmpty(reason), updatedAt, id,
	)
	return err
}

// ExpireCandidates 将过期 pending 标记为 expired。
func ExpireCandidates(db *sql.DB, nowISO string, dryRun bool) (int, error) {
	if dryRun {
		var n int
		err := db.QueryRow(
			`SELECT COUNT(*) FROM library_candidates WHERE status = 'pending' AND expires_at < ?`, nowISO,
		).Scan(&n)
		return n, err
	}
	res, err := db.Exec(`
		UPDATE library_candidates SET status = 'expired', updated_at = ?
		WHERE status = 'pending' AND expires_at < ?`, nowISO, nowISO)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// InsertLibraryItem 写入 library_item。
func InsertLibraryItem(db *sql.DB, item *schema.LibraryItem) error {
	_, err := db.Exec(`
		INSERT INTO library_items (
			id, status, title, source, tier, timestamp, collected_at, author,
			content_type, media_type, related_stocks_json, tags_json, dedup_key,
			canonical_url, cluster_id, primary_asset_id, summary_by_user, user_notes,
			promoted_from_candidate_id, merged_into_id, duplicate_of_id, schema_version,
			reference_count, last_referenced_at, archived_at, created_at, updated_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		item.ID, item.Status, item.Title, item.Source, item.Tier, item.Timestamp, item.CollectedAt,
		nullIfEmpty(item.Author), item.ContentType, item.MediaType, item.RelatedStocksJSON, item.TagsJSON, item.DedupKey,
		nullIfEmpty(item.CanonicalURL), nullIfEmpty(item.ClusterID), nullIfEmpty(item.PrimaryAssetID),
		nullIfEmpty(item.SummaryByUser), nullIfEmpty(item.UserNotes), nullIfEmpty(item.PromotedFromCandidateID),
		nullIfEmpty(item.MergedIntoID), nullIfEmpty(item.DuplicateOfID), item.SchemaVersion,
		item.ReferenceCount, nullIfEmpty(item.LastReferencedAt), nullIfEmpty(item.ArchivedAt),
		item.CreatedAt, item.UpdatedAt,
	)
	return err
}

// InsertLibraryAsset 写入 library_item_assets。
func InsertLibraryAsset(db *sql.DB, a *schema.LibraryItemAsset) error {
	_, err := db.Exec(`
		INSERT INTO library_item_assets (
			id, library_item_id, asset_role, source, tier, timestamp,
			file_path, file_sha256, canonical_url, promoted_from_candidate_id,
			supplement_note, created_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		a.ID, a.LibraryItemID, a.AssetRole, a.Source, a.Tier, a.Timestamp,
		nullIfEmpty(a.FilePath), nullIfEmpty(a.FileSHA256), nullIfEmpty(a.CanonicalURL),
		nullIfEmpty(a.PromotedFromCandidateID), nullIfEmpty(a.SupplementNote), a.CreatedAt,
	)
	return err
}

// GetLibraryItem 读取单个 item。
func GetLibraryItem(db *sql.DB, id string) (*schema.LibraryItem, error) {
	row := db.QueryRow(`
		SELECT id, status, title, source, tier, timestamp, collected_at,
			COALESCE(author,''), content_type, media_type,
			COALESCE(related_stocks_json,'[]'), COALESCE(tags_json,'[]'), dedup_key,
			COALESCE(canonical_url,''), COALESCE(cluster_id,''), COALESCE(primary_asset_id,''),
			COALESCE(summary_by_user,''), COALESCE(user_notes,''), COALESCE(promoted_from_candidate_id,''),
			schema_version, reference_count, created_at, updated_at
		FROM library_items WHERE id = ?`, id)

	var item schema.LibraryItem
	err := row.Scan(
		&item.ID, &item.Status, &item.Title, &item.Source, &item.Tier, &item.Timestamp, &item.CollectedAt,
		&item.Author, &item.ContentType, &item.MediaType, &item.RelatedStocksJSON, &item.TagsJSON, &item.DedupKey,
		&item.CanonicalURL, &item.ClusterID, &item.PrimaryAssetID, &item.SummaryByUser, &item.UserNotes,
		&item.PromotedFromCandidateID, &item.SchemaVersion, &item.ReferenceCount, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("library_item 不存在: %s", id)
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// UpdateLibraryItemTags 写回 tags_json 与 updated_at。
func UpdateLibraryItemTags(db *sql.DB, id, tagsJSON, updatedAt string) error {
	_, err := db.Exec(`UPDATE library_items SET tags_json = ?, updated_at = ? WHERE id = ?`, tagsJSON, updatedAt, id)
	return err
}

// SetLibraryItemPrimaryAsset 回填 primary_asset_id。
func SetLibraryItemPrimaryAsset(db *sql.DB, itemID, assetID string) error {
	_, err := db.Exec(`UPDATE library_items SET primary_asset_id = ? WHERE id = ?`, assetID, itemID)
	return err
}

// ArchiveLibraryItem 归档 item。
func ArchiveLibraryItem(db *sql.DB, id, updatedAt string) error {
	_, err := db.Exec(`UPDATE library_items SET status = 'archived', updated_at = ? WHERE id = ?`, updatedAt, id)
	return err
}

// ListLibraryItems 列表查询。
func ListLibraryItems(db *sql.DB, status, stock, tag string) ([]LibraryItemRow, error) {
	q := `SELECT id, status, title, source, tier, content_type, media_type,
		COALESCE(related_stocks_json,'[]'), COALESCE(tags_json,'[]'), dedup_key,
		COALESCE(summary_by_user,''), COALESCE(promoted_from_candidate_id,''), created_at
		FROM library_items WHERE 1=1`
	var args []any
	if status != "" {
		q += ` AND status = ?`
		args = append(args, status)
	}
	if stock != "" {
		q += ` AND related_stocks_json LIKE ?`
		args = append(args, `%"`+stock+`"%`)
	}
	if tag != "" {
		q += ` AND tags_json LIKE ?`
		args = append(args, `%"`+tag+`"%`)
	}
	q += ` ORDER BY created_at DESC`

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LibraryItemRow
	for rows.Next() {
		var r LibraryItemRow
		if err := rows.Scan(
			&r.ID, &r.Status, &r.Title, &r.Source, &r.Tier, &r.ContentType, &r.MediaType,
			&r.RelatedStocksJSON, &r.TagsJSON, &r.DedupKey, &r.SummaryByUser,
			&r.PromotedFromCandidateID, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SearchLibraryItems 标题/notes 模糊搜索（MVP-1 LIKE）。
func SearchLibraryItems(db *sql.DB, query, stock string) ([]LibraryItemRow, error) {
	q := `SELECT id, status, title, source, tier, content_type, media_type,
		COALESCE(related_stocks_json,'[]'), COALESCE(tags_json,'[]'), dedup_key,
		COALESCE(summary_by_user,''), COALESCE(promoted_from_candidate_id,''), created_at
		FROM library_items WHERE status = 'active'
		AND (title LIKE ? OR summary_by_user LIKE ? OR user_notes LIKE ?)`
	pattern := "%" + query + "%"
	args := []any{pattern, pattern, pattern}
	if stock != "" {
		q += ` AND related_stocks_json LIKE ?`
		args = append(args, `%"`+stock+`"%`)
	}
	q += ` ORDER BY created_at DESC`

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LibraryItemRow
	for rows.Next() {
		var r LibraryItemRow
		if err := rows.Scan(
			&r.ID, &r.Status, &r.Title, &r.Source, &r.Tier, &r.ContentType, &r.MediaType,
			&r.RelatedStocksJSON, &r.TagsJSON, &r.DedupKey, &r.SummaryByUser,
			&r.PromotedFromCandidateID, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CountPrimaryAssets 统计 item 的 primary asset 数量。
func CountPrimaryAssets(db *sql.DB, itemID string) (int, error) {
	var n int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM library_item_assets WHERE library_item_id = ? AND asset_role = 'primary'`,
		itemID,
	).Scan(&n)
	return n, err
}

// ParseJSONStringArray 解析 JSON 字符串数组。
func ParseJSONStringArray(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
