package schema

// TableDataSnapshots 表名常量。
const TableDataSnapshots = "data_snapshots"

// DataSnapshot 决策时市场/估值快照（03 §10A.5）；写入后只读，L1 后续更新不回填。
type DataSnapshot struct {
	// ID 主键，通常与 journal 关联生成。
	ID string `db:"id"`
	// JournalID 所属 journal id。
	JournalID string `db:"journal_id"`
	// SnapshotJSON 02 §4.5 最小字段组 + 行情/估值事实（tier 标注）。
	SnapshotJSON string `db:"snapshot_json"`
	// SchemaVersion 快照 JSON 结构版本。
	SchemaVersion int `db:"schema_version"`
	// CreatedAt 冻结时间 ISO8601，不可更新。
	CreatedAt string `db:"created_at"`
}
