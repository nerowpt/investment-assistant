package schema

// TableChecklistSubmissions 表名常量。
const TableChecklistSubmissions = "checklist_submissions"

// ChecklistSubmission Checklist 提交记录（03 §10A.2）。
type ChecklistSubmission struct {
	// ID 主键，格式 cs_{YYYYMMDD}_{seq}。
	ID string `db:"id"`
	// ChecklistType 类型：watch | buy | add | inspect | sell | review | import（swap 预留 MVP-2）。
	ChecklistType string `db:"checklist_type"`
	// Code 标的代码；主题级观察可空。
	Code string `db:"code"`
	// Name 标的或主题名称。
	Name string `db:"name"`
	// PayloadJSON 用户提交的完整字段 JSON（权威）；含 emotion_retrospect: null 预留位（D3）。
	PayloadJSON string `db:"payload_json"`
	// PayloadSchemaVersion payload 结构版本，当前 1。
	PayloadSchemaVersion int `db:"payload_schema_version"`
	// Status draft | submitted | approved | rejected。
	Status string `db:"status"`
	// SubmittedBy 提交者，MVP-1 固定 user。
	SubmittedBy string `db:"submitted_by"`
	// TierAcknowledgement C/D 级主要依据时 0/1 确认。
	TierAcknowledgement *int `db:"tier_acknowledgement"`
	// EmotionSelfCheck 情绪自检说明文案。
	EmotionSelfCheck string `db:"emotion_self_check"`
	// RiskGuardrailResultJSON M7 护栏检查结果 JSON。
	RiskGuardrailResultJSON string `db:"risk_guardrail_result_json"`
	// ExceptionJSON hard_block 例外说明 JSON（D5）；soft 时为 ack。
	ExceptionJSON string `db:"exception_json"`
	// LinkedLibraryIDsJSON 关联 L1 id 列表 JSON。
	LinkedLibraryIDsJSON string `db:"linked_library_ids_json"`
	// GeneratedJournalID approve 后回填 journal id。
	GeneratedJournalID string `db:"generated_journal_id"`
	// GeneratedInspectionID inspect approve 后回填 insp_* id。
	GeneratedInspectionID string `db:"generated_inspection_id"`
	// GeneratedReviewID review approve 后回填 rev_* id。
	GeneratedReviewID string `db:"generated_review_id"`
	// CreatedAt 创建时间。
	CreatedAt string `db:"created_at"`
	// SubmittedAt 提交时间，draft 时空。
	SubmittedAt string `db:"submitted_at"`
	// ApprovedAt 批准时间。
	ApprovedAt string `db:"approved_at"`
}
