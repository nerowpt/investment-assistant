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
