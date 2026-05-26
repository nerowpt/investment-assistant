package schema

// TableRiskExceptions 表名常量。
const TableRiskExceptions = "risk_exceptions"

// RiskException 风险护栏触发记录（03 §10A.8）；warning 也入库。
type RiskException struct {
	// ID 主键。
	ID string `db:"id"`
	// Severity warning | hard_block。
	Severity string `db:"severity"`
	// RuleSource 规则来源，如 risk_rules | personal_redlines。
	RuleSource string `db:"rule_source"`
	// RuleID 规则 id，如 r001。
	RuleID string `db:"rule_id"`
	// ChecklistSubmissionID 触发时的 checklist id。
	ChecklistSubmissionID string `db:"checklist_submission_id"`
	// JournalID approve 后关联 journal（可选）。
	JournalID string `db:"journal_id"`
	// ExceptionReason hard_block 例外原因（≥80 字，D5）。
	ExceptionReason string `db:"exception_reason"`
	// ExpectedCompensation 预期补偿/对冲措施（hard_block 必填）。
	ExpectedCompensation string `db:"expected_compensation"`
	// ReviewDate 复盘日期 hard_block 必填。
	ReviewDate string `db:"review_date"`
	// OutcomeNote 复盘结果备注。
	OutcomeNote string `db:"outcome_note"`
	// CreatedAt 创建时间。
	CreatedAt string `db:"created_at"`
}

const (
	SeverityWarning   = "warning"
	SeverityHardBlock = "hard_block"
)
