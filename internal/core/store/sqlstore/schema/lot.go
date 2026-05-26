package schema

// TableLots 表名常量。
const TableLots = "lots"

// Lot 仓位 lot（03 §10A.6）；用于 lot 级收益归因与 FIFO 卖出匹配。
type Lot struct {
	// ID 主键，格式 lot_{YYYYMMDD}_{seq}。
	ID string `db:"id"`
	// Code 标的代码，与 portfolio position.code 一致。
	Code string `db:"code"`
	// Name 标的名称。
	Name string `db:"name"`
	// JournalID 开启本 lot 的 journal id（buy/add/import）。
	JournalID string `db:"journal_id"`
	// ActionType 开启动作：buy | add | import。
	ActionType string `db:"action_type"`
	// PositionType 仓位风格：core（主仓）| swing（波段）。
	PositionType string `db:"position_type"`
	// OpenAt lot 开启日期 ISO8601 或 YYYY-MM-DD。
	OpenAt string `db:"open_at"`
	// CloseAt lot 完全关闭时间；status=closed 时填写。
	CloseAt string `db:"close_at"`
	// InitialPct 该 lot 产生时占总资产比例（逻辑 DECIMAL，读走 decimal_scan）。
	InitialPct string `db:"initial_pct"`
	// CurrentPct 当前剩余比例，随卖出递减（逻辑 DECIMAL）。
	CurrentPct string `db:"current_pct"`
	// CostBasis 该 lot 成本价（逻辑 DECIMAL）。
	CostBasis string `db:"cost_basis"`
	// Shares 可选股数（逻辑 DECIMAL）。
	Shares string `db:"shares"`
	// Status open | partial | closed。
	Status string `db:"status"`
	// LinkedBuyJournalID add 类型 lot 指向原建仓 journal id。
	LinkedBuyJournalID string `db:"linked_buy_journal_id"`
	// DividendsReceived 已收现金分红累计（元）；MVP-1 默认 0，见 docs/06 §D8。
	DividendsReceived string `db:"dividends_received"`
	// AdjustedCostBasis 前复权调整后成本；NULL 表示沿用 CostBasis。
	AdjustedCostBasis string `db:"adjusted_cost_basis"`
	// CorporateActionsJSON 送转/拆股/配股事件 JSON 数组；MVP-1 可 null。
	CorporateActionsJSON string `db:"corporate_actions_json"`
	// CreatedAt 记录创建时间 ISO8601。
	CreatedAt string `db:"created_at"`
}

// LotStatus 合法 status 值。
const (
	LotStatusOpen     = "open"
	LotStatusPartial  = "partial"
	LotStatusClosed   = "closed"
)
