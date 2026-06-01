package schema

// TableLotAllocations 表名常量。
const TableLotAllocations = "lot_allocations"

// LotAllocation 卖出时 lot 匹配记录（03 §10A.7 Q4C FIFO）。
type LotAllocation struct {
	// ID 主键。
	ID string `db:"id"`
	// SellJournalID 卖出 journal id。
	SellJournalID string `db:"sell_journal_id"`
	// LotID 被匹配的 lot id。
	LotID string `db:"lot_id"`
	// AllocatedPct 本次从该 lot 扣减的总仓位百分点（逻辑 DECIMAL）。
	AllocatedPct string `db:"allocated_pct"`
	// CostBasisAtSale 卖出时 lot 成本价（逻辑 DECIMAL）。
	CostBasisAtSale string `db:"cost_basis_at_sale"`
	// ProceedsPct 卖出所得占组合比例（可选，逻辑 DECIMAL）。
	ProceedsPct string `db:"proceeds_pct"`
	// RealizedReturnPct 该 allocation 实现收益率 %（逻辑 DECIMAL）。
	RealizedReturnPct string `db:"realized_return_pct"`
	// AllocatedShares 本次从该 lot 卖出的股数（B 模型；逻辑 DECIMAL）。
	AllocatedShares string `db:"allocated_shares"`
	// ProceedsAmount 卖出所得金额（元）= execution_price × allocated_shares。
	ProceedsAmount string `db:"proceeds_amount"`
	// RealizedPnLAmount 实现盈亏（元）= (成交价 - 成本) × 股数。
	RealizedPnLAmount string `db:"realized_pnl_amount"`
	// MatchMethod recommended_fifo | user_adjusted。
	MatchMethod string `db:"match_method"`
	// UserConfirmed 用户是否确认匹配，0/1。
	UserConfirmed int `db:"user_confirmed"`
	// CreatedAt 创建时间 ISO8601。
	CreatedAt string `db:"created_at"`
}

const (
	MatchMethodRecommendedFIFO = "recommended_fifo"
	MatchMethodUserAdjusted      = "user_adjusted"
)
