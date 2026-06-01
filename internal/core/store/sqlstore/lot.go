package sqlstore

import (
	"database/sql"
	"fmt"

	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore/schema"
)

// ListOpenLotsByCode 列出标的 open/partial lot（FIFO 序：open_at, id）。
func ListOpenLotsByCode(db *sql.DB, code string) ([]schema.Lot, error) {
	rows, err := db.Query(`
		SELECT id, code, COALESCE(name,''), journal_id, action_type, position_type,
			open_at, COALESCE(close_at,''),
			CAST(initial_pct AS TEXT), CAST(current_pct AS TEXT), CAST(cost_basis AS TEXT),
			COALESCE(CAST(shares AS TEXT),''), status, COALESCE(linked_buy_journal_id,''),
			COALESCE(CAST(dividends_received AS TEXT),''), COALESCE(CAST(adjusted_cost_basis AS TEXT),''),
			COALESCE(corporate_actions_json,''), created_at
		FROM lots
		WHERE code = ? AND status IN ('open', 'partial')
		ORDER BY open_at ASC, id ASC`, code)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []schema.Lot
	for rows.Next() {
		var l schema.Lot
		if err := rows.Scan(
			&l.ID, &l.Code, &l.Name, &l.JournalID, &l.ActionType, &l.PositionType,
			&l.OpenAt, &l.CloseAt, &l.InitialPct, &l.CurrentPct, &l.CostBasis,
			&l.Shares, &l.Status, &l.LinkedBuyJournalID,
			&l.DividendsReceived, &l.AdjustedCostBasis, &l.CorporateActionsJSON, &l.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// UpdateLotAfterSell 更新 lot 剩余股数、仓位 % 与 status（sell approve）。
func UpdateLotAfterSell(db execer, lotID, newCurrentPct, newShares, status, closeAt string) error {
	_, err := db.Exec(`
		UPDATE lots SET current_pct = ?, shares = ?, status = ?, close_at = ? WHERE id = ?`,
		newCurrentPct, nullStr(newShares), status, nullStr(closeAt), lotID,
	)
	return err
}

// InsertLotAllocation 写入 lot_allocations 行。
func InsertLotAllocation(db execer, row *schema.LotAllocation) error {
	_, err := db.Exec(`
		INSERT INTO lot_allocations (
			id, sell_journal_id, lot_id, allocated_pct, cost_basis_at_sale,
			proceeds_pct, realized_return_pct, match_method, user_confirmed, created_at,
			allocated_shares, proceeds_amount, realized_pnl_amount
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		row.ID, row.SellJournalID, row.LotID, row.AllocatedPct, row.CostBasisAtSale,
		nullStr(row.ProceedsPct), nullStr(row.RealizedReturnPct),
		row.MatchMethod, row.UserConfirmed, row.CreatedAt,
		nullStr(row.AllocatedShares), nullStr(row.ProceedsAmount), nullStr(row.RealizedPnLAmount),
	)
	return err
}

// UpdateChecklistPayload 更新 draft/submitted checklist 的 payload_json。
func UpdateChecklistPayload(db *sql.DB, id, payloadJSON string) error {
	res, err := db.Exec(`
		UPDATE checklist_submissions SET payload_json = ?
		WHERE id = ? AND status IN ('draft', 'submitted')`, payloadJSON, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("checklist 不存在或不可更新 payload: %s", id)
	}
	return nil
}
