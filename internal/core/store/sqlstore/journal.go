package sqlstore

import (
	"database/sql"

	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore/schema"
)

type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// InsertJournal 写入 journals 行。
func InsertJournal(db execer, row *schema.Journal) error {
	_, err := db.Exec(`
		INSERT INTO journals (
			id, action_type, code, name, checklist_submission_id,
			data_snapshot_id, payload_json, summary, lot_id, created_at
		) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		row.ID, row.ActionType, nullStr(row.Code), nullStr(row.Name),
		nullStr(row.ChecklistSubmissionID), nullStr(row.DataSnapshotID),
		row.PayloadJSON, nullStr(row.Summary), nullStr(row.LotID), row.CreatedAt,
	)
	return err
}

// InsertLot 写入 lots 行。
func InsertLot(db execer, row *schema.Lot) error {
	_, err := db.Exec(`
		INSERT INTO lots (
			id, code, name, journal_id, action_type, position_type,
			open_at, close_at, initial_pct, current_pct, cost_basis, shares,
			status, linked_buy_journal_id, dividends_received, adjusted_cost_basis,
			corporate_actions_json, created_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		row.ID, row.Code, nullStr(row.Name), row.JournalID, row.ActionType, row.PositionType,
		row.OpenAt, nullStr(row.CloseAt), row.InitialPct, row.CurrentPct, row.CostBasis,
		nullStr(row.Shares), row.Status, nullStr(row.LinkedBuyJournalID),
		nullStr(row.DividendsReceived), nullStr(row.AdjustedCostBasis),
		nullStr(row.CorporateActionsJSON), row.CreatedAt,
	)
	return err
}

// InsertDataSnapshot 写入 data_snapshots 行。
func InsertDataSnapshot(db execer, row *schema.DataSnapshot) error {
	_, err := db.Exec(`
		INSERT INTO data_snapshots (id, journal_id, snapshot_json, schema_version, created_at)
		VALUES (?,?,?,?,?)`,
		row.ID, row.JournalID, row.SnapshotJSON, row.SchemaVersion, row.CreatedAt,
	)
	return err
}

// InsertSyncRepair 写入 sync_repairs 行（YAML 写失败时，T5 不回滚 SQL）。
func InsertSyncRepair(db execer, row *schema.SyncRepair) error {
	_, err := db.Exec(`
		INSERT INTO sync_repairs (
			id, checklist_submission_id, journal_id, yaml_files_json,
			error_message, status, created_at, resolved_at
		) VALUES (?,?,?,?,?,?,?,?)`,
		row.ID, nullStr(row.ChecklistSubmissionID), nullStr(row.JournalID),
		row.YAMLFilesJSON, row.ErrorMessage, row.Status, row.CreatedAt, nullStr(row.ResolvedAt),
	)
	return err
}

// InsertInspectionRecord 写入 inspection_records。
func InsertInspectionRecord(db execer, row *schema.InspectionRecord) error {
	_, err := db.Exec(`
		INSERT INTO inspection_records (
			id, checklist_submission_id, code, inspection_type, linked_buy_journal_id,
			fact_update_summary_json, user_judgment_json, report_path, created_at
		) VALUES (?,?,?,?,?,?,?,?,?)`,
		row.ID, row.ChecklistSubmissionID, row.Code, row.InspectionType,
		nullStr(row.LinkedBuyJournalID), nullStr(row.FactUpdateSummaryJSON),
		row.UserJudgmentJSON, nullStr(row.ReportPath), row.CreatedAt,
	)
	return err
}

// InsertReviewReport 写入 review_reports。
func InsertReviewReport(db execer, row *schema.ReviewReport) error {
	_, err := db.Exec(`
		INSERT INTO review_reports (
			id, checklist_submission_id, review_type, period_start, period_end,
			stats_json, user_judgment_json, report_path, created_at
		) VALUES (?,?,?,?,?,?,?,?,?)`,
		row.ID, row.ChecklistSubmissionID, row.ReviewType, row.PeriodStart, row.PeriodEnd,
		nullStr(row.StatsJSON), row.UserJudgmentJSON, nullStr(row.ReportPath), row.CreatedAt,
	)
	return err
}

// UpdateRiskExceptionsJournalID 回填 risk_exceptions.journal_id。
func UpdateRiskExceptionsJournalID(db execer, checklistID, journalID string) error {
	_, err := db.Exec(`
		UPDATE risk_exceptions SET journal_id = ? WHERE checklist_submission_id = ?`,
		journalID, checklistID,
	)
	return err
}

// JournalExists 检查 journal id 是否存在。
func JournalExists(db *sql.DB, id string) (bool, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(1) FROM journals WHERE id = ?`, id).Scan(&n)
	return n > 0, err
}

// LotExists 检查 lot id 是否存在。
func LotExists(db *sql.DB, id string) (bool, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(1) FROM lots WHERE id = ?`, id).Scan(&n)
	return n > 0, err
}

// JournalRow journals 列表/详情查询行。
type JournalRow struct {
	ID                    string
	ActionType            string
	Code                  string
	Name                  string
	ChecklistSubmissionID string
	DataSnapshotID        string
	Summary               string
	LotID                 string
	CreatedAt             string
}

// CountSellJournalsByCode 统计各标的卖出 journal 条数（看板「已减仓」提示）。
func CountSellJournalsByCode(db *sql.DB) (map[string]int, error) {
	rows, err := db.Query(`SELECT COALESCE(code,''), COUNT(1) FROM journals WHERE action_type = 'sell' GROUP BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var code string
		var n int
		if err := rows.Scan(&code, &n); err != nil {
			return nil, err
		}
		if code != "" {
			out[code] = n
		}
	}
	return out, rows.Err()
}

// SearchJournals 按标的/动作检索 journal（MCP/CLI 只读）。
func SearchJournals(db *sql.DB, code, actionType string, limit int) ([]JournalRow, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	q := `SELECT id, action_type, COALESCE(code,''), COALESCE(name,''),
		COALESCE(checklist_submission_id,''), COALESCE(data_snapshot_id,''),
		COALESCE(summary,''), COALESCE(lot_id,''), created_at
		FROM journals WHERE 1=1`
	var args []any
	if code != "" {
		q += ` AND code = ?`
		args = append(args, code)
	}
	if actionType != "" {
		q += ` AND action_type = ?`
		args = append(args, actionType)
	}
	q += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []JournalRow
	for rows.Next() {
		var r JournalRow
		if err := rows.Scan(&r.ID, &r.ActionType, &r.Code, &r.Name, &r.ChecklistSubmissionID,
			&r.DataSnapshotID, &r.Summary, &r.LotID, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetJournal 按 id 读取 journal。
func GetJournal(db *sql.DB, id string) (*JournalRow, error) {
	row := db.QueryRow(`SELECT id, action_type, COALESCE(code,''), COALESCE(name,''),
		COALESCE(checklist_submission_id,''), COALESCE(data_snapshot_id,''),
		COALESCE(summary,''), COALESCE(lot_id,''), created_at
		FROM journals WHERE id = ?`, id)
	var r JournalRow
	err := row.Scan(&r.ID, &r.ActionType, &r.Code, &r.Name, &r.ChecklistSubmissionID,
		&r.DataSnapshotID, &r.Summary, &r.LotID, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// GetJournalPayload 读取 journal payload_json。
func GetJournalPayload(db *sql.DB, id string) (string, error) {
	var s string
	err := db.QueryRow(`SELECT payload_json FROM journals WHERE id = ?`, id).Scan(&s)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return s, err
}

// GetDataSnapshotSummary 读取 snapshot_json（MCP 返回摘要时可截断）。
func GetDataSnapshotSummary(db *sql.DB, snapID string) (string, error) {
	if snapID == "" {
		return "", nil
	}
	var raw string
	err := db.QueryRow(`SELECT snapshot_json FROM data_snapshots WHERE id = ?`, snapID).Scan(&raw)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return raw, err
}
