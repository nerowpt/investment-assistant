package schema

// TableLibraryCandidates 表名常量。
const TableLibraryCandidates = "library_candidates"

// LibraryCandidate L1 待归纳候选（03 §九）。
type LibraryCandidate struct {
	ID                      string `db:"id"`
	Status                  string `db:"status"`                    // pending | dismissed | promoted | expired
	SourceEntry             string `db:"source_entry"`              // manual | passive_discovery | crawl | import_batch
	Title                   string `db:"title"`
	Source                  string `db:"source"`
	Tier                    string `db:"tier"`                      // S | A | B | C | D
	Timestamp               string `db:"timestamp"`
	Author                  string `db:"author"`
	ContentType             string `db:"content_type"`
	MediaType               string `db:"media_type"`                // MVP-1 启用 text|pdf|html|structured
	RelatedStocksJSON       string `db:"related_stocks_json"`
	TagsJSON                string `db:"tags_json"`
	DedupKey                string `db:"dedup_key"`
	StagingPath             string `db:"staging_path"`
	CanonicalURL            string `db:"canonical_url"`
	ExtractJSON             string `db:"extract_json"`
	SummaryDraft            string `db:"summary_draft"`
	SimilarityJSON          string `db:"similarity_json"`
	MatchTier               string `db:"match_tier"`                // none | exact | near | theme
	Resolution              string `db:"resolution"`
	ResolutionTargetItemID  string `db:"resolution_target_item_id"`
	ExpiresAt               string `db:"expires_at"`
	PromotedLibraryItemID   string `db:"promoted_library_item_id"`
	DismissedReason         string `db:"dismissed_reason"`
	CreatedAt               string `db:"created_at"`
	UpdatedAt               string `db:"updated_at"`
}

// TableLibraryItems 表名常量。
const TableLibraryItems = "library_items"

// LibraryItem L1 正式素材（03 §九）。
type LibraryItem struct {
	ID                       string `db:"id"`
	Status                   string `db:"status"`
	Title                    string `db:"title"`
	Source                   string `db:"source"`
	Tier                     string `db:"tier"`
	Timestamp                string `db:"timestamp"`
	CollectedAt              string `db:"collected_at"`
	Author                   string `db:"author"`
	ContentType              string `db:"content_type"`
	MediaType                string `db:"media_type"`
	RelatedStocksJSON        string `db:"related_stocks_json"`
	TagsJSON                 string `db:"tags_json"`
	DedupKey                 string `db:"dedup_key"`
	CanonicalURL             string `db:"canonical_url"`
	ClusterID                string `db:"cluster_id"`
	PrimaryAssetID           string `db:"primary_asset_id"`
	SummaryByUser            string `db:"summary_by_user"`
	UserNotes                string `db:"user_notes"`
	PromotedFromCandidateID  string `db:"promoted_from_candidate_id"`
	MergedIntoID             string `db:"merged_into_id"`
	DuplicateOfID            string `db:"duplicate_of_id"`
	SchemaVersion            int    `db:"schema_version"`
	ReferenceCount           int    `db:"reference_count"`
	LastReferencedAt         string `db:"last_referenced_at"`
	ArchivedAt               string `db:"archived_at"`
	CreatedAt                string `db:"created_at"`
	UpdatedAt                string `db:"updated_at"`
}

// MVP1MediaTypes MVP-1 允许的 media_type（docs/06 §D9）。
var MVP1MediaTypes = []string{"text", "pdf", "html", "structured"}

// MVP1ContentTypes MVP-1 允许的 content_type（03 §9.3）。
var MVP1ContentTypes = []string{"announcement", "report", "article", "note", "transcript", "data"}

// TableLibraryItemAssets 表名常量。
const TableLibraryItemAssets = "library_item_assets"

// LibraryItemAsset L1 物理资产行（03 §9.4）。
type LibraryItemAsset struct {
	ID                      string `db:"id"`
	LibraryItemID           string `db:"library_item_id"`
	AssetRole               string `db:"asset_role"` // primary | supplement
	Source                  string `db:"source"`
	Tier                    string `db:"tier"`
	Timestamp               string `db:"timestamp"`
	FilePath                string `db:"file_path"`
	FileSHA256              string `db:"file_sha256"`
	CanonicalURL            string `db:"canonical_url"`
	PromotedFromCandidateID string `db:"promoted_from_candidate_id"`
	SupplementNote          string `db:"supplement_note"`
	CreatedAt               string `db:"created_at"`
}
