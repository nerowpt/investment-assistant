package schema

// TableJournals 表名常量。
const TableJournals = "journals"

// Journal 决策日志行（03 §10A.4）。
// 每笔 Checklist approve 产生的 buy/add/sell/import/rule_change 写入一条，写入后不可变。
type Journal struct {
	// ID 主键，格式 j_{YYYYMMDD}_{seq}（T9 日序，如 j_20260519_001）。
	ID string `db:"id"`
	// ActionType 动作类型：buy | add | sell | import | rule_change。
	ActionType string `db:"action_type"`
	// Code 标的代码；import 批次级 journal 可空。
	Code string `db:"code"`
	// Name 标的名称，便于列表展示。
	Name string `db:"name"`
	// ChecklistSubmissionID 来源 Checklist id（cs_*）；rule_change 可来自 review 流程。
	ChecklistSubmissionID string `db:"checklist_submission_id"`
	// DataSnapshotID 关联冻结快照 id；buy/add/sell 通常必填，import 可简版或空。
	DataSnapshotID string `db:"data_snapshot_id"`
	// PayloadJSON approve 时刻从 checklist 完整复制的 JSON（权威业务字段）。
	PayloadJSON string `db:"payload_json"`
	// Summary 一行摘要，CLI list 用。
	Summary string `db:"summary"`
	// LotID buy/add/import 创建 lot 时指向新 lot id；sell 通常为空。
	LotID string `db:"lot_id"`
	// CreatedAt 创建时间 ISO8601，不可更新。
	CreatedAt string `db:"created_at"`
}

// JournalActionType 合法 action_type 常量（MVP-1）。
const (
	JournalActionBuy        = "buy"
	JournalActionAdd        = "add"
	JournalActionSell       = "sell"
	JournalActionImport     = "import"
	JournalActionRuleChange = "rule_change"
)
