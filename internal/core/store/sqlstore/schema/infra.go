package schema

// TableIDSequences 表名常量。
const TableIDSequences = "id_sequences"

// IDSequence 日序 ID 生成器（T9）；prefix 如 cs / j / lot。
type IDSequence struct {
	// Prefix ID 前缀。
	Prefix string `db:"prefix"`
	// SeqDate 日期 YYYYMMDD。
	SeqDate string `db:"seq_date"`
	// NextSeq 下一序号，从 1 起。
	NextSeq int `db:"next_seq"`
}

// TableSyncRepairs 表名常量。
const TableSyncRepairs = "sync_repairs"

// SyncRepair SQL 已提交但 YAML 未写完的修复队列（T5）。
type SyncRepair struct {
	// ID 主键。
	ID string `db:"id"`
	// ChecklistSubmissionID 关联 checklist（可选）。
	ChecklistSubmissionID string `db:"checklist_submission_id"`
	// JournalID 关联 journal（可选）。
	JournalID string `db:"journal_id"`
	// YAMLFilesJSON 待写文件清单 JSON。
	YAMLFilesJSON string `db:"yaml_files_json"`
	// ErrorMessage 失败原因。
	ErrorMessage string `db:"error_message"`
	// Status pending | resolved | aborted。
	Status string `db:"status"`
	// CreatedAt 创建时间。
	CreatedAt string `db:"created_at"`
	// ResolvedAt 修复完成时间。
	ResolvedAt string `db:"resolved_at"`
}

// TableSchemaMeta 表名常量。
const TableSchemaMeta = "schema_meta"

// SchemaMeta 迁移版本标记。
type SchemaMeta struct {
	// Key 键，如 schema_version。
	Key string `db:"key"`
	// Value 值，如 1。
	Value string `db:"value"`
}

// TableMonitorEvents 表名常量。
const TableMonitorEvents = "monitor_events"

// MonitorEvent MVP-1 简版监控事件。
type MonitorEvent struct {
	// ID 主键。
	ID string `db:"id"`
	// Code 关联标的（可选）。
	Code string `db:"code"`
	// EventType 事件类型，如 price_threshold | announcement。
	EventType string `db:"event_type"`
	// PayloadJSON 事件详情 JSON。
	PayloadJSON string `db:"payload_json"`
	// Acknowledged 是否已确认，0/1。
	Acknowledged int `db:"acknowledged"`
	// CreatedAt 创建时间。
	CreatedAt string `db:"created_at"`
}
