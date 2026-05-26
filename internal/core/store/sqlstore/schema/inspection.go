package schema

// TableInspectionRecords 表名常量。
const TableInspectionRecords = "inspection_records"

// InspectionRecord 巡检 Checklist approve 产物（03 §10A.9）。
type InspectionRecord struct {
	// ID 主键，格式 insp_{YYYYMMDD}_{seq}。
	ID string `db:"id"`
	// ChecklistSubmissionID 来源 checklist id。
	ChecklistSubmissionID string `db:"checklist_submission_id"`
	// Code 标的代码。
	Code string `db:"code"`
	// InspectionType 巡检类型（与 checklist 子类型对齐）。
	InspectionType string `db:"inspection_type"`
	// LinkedBuyJournalID 关联建仓 journal（可选）。
	LinkedBuyJournalID string `db:"linked_buy_journal_id"`
	// FactUpdateSummaryJSON 事实区更新 JSON（系统/AI 事实，非结论）。
	FactUpdateSummaryJSON string `db:"fact_update_summary_json"`
	// UserJudgmentJSON 用户判断：四维度+四象限+planned_action 等。
	UserJudgmentJSON string `db:"user_judgment_json"`
	// ReportPath 可选 Markdown 报告相对路径。
	ReportPath string `db:"report_path"`
	// CreatedAt 创建时间。
	CreatedAt string `db:"created_at"`
}

// TableReviewReports 表名常量。
const TableReviewReports = "review_reports"

// ReviewReport 复盘 Checklist approve 产物（03 §10A.9）。
type ReviewReport struct {
	// ID 主键，格式 rev_{period}_{seq}。
	ID string `db:"id"`
	// ChecklistSubmissionID 来源 checklist id。
	ChecklistSubmissionID string `db:"checklist_submission_id"`
	// ReviewType 复盘类型：monthly | quarterly | adhoc 等。
	ReviewType string `db:"review_type"`
	// PeriodStart 复盘区间开始 YYYY-MM-DD。
	PeriodStart string `db:"period_start"`
	// PeriodEnd 复盘区间结束。
	PeriodEnd string `db:"period_end"`
	// StatsJSON 系统统计区 JSON。
	StatsJSON string `db:"stats_json"`
	// UserJudgmentJSON 用户判断：confirmed_patterns 等。
	UserJudgmentJSON string `db:"user_judgment_json"`
	// ReportPath reports/reviews/ 下 Markdown 路径。
	ReportPath string `db:"report_path"`
	// CreatedAt 创建时间。
	CreatedAt string `db:"created_at"`
}
