package sqlstore

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore/schema"
)

// InsertChecklistSubmission 插入 checklist 行。
func InsertChecklistSubmission(db *sql.DB, row *schema.ChecklistSubmission) error {
	_, err := db.Exec(`
		INSERT INTO checklist_submissions (
			id, checklist_type, code, name, payload_json, payload_schema_version,
			status, submitted_by, tier_acknowledgement, emotion_self_check,
			risk_guardrail_result_json, exception_json, linked_library_ids_json,
			generated_journal_id, generated_inspection_id, generated_review_id,
			created_at, submitted_at, approved_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		row.ID, row.ChecklistType, row.Code, row.Name, row.PayloadJSON, row.PayloadSchemaVersion,
		row.Status, row.SubmittedBy, row.TierAcknowledgement, row.EmotionSelfCheck,
		row.RiskGuardrailResultJSON, row.ExceptionJSON, row.LinkedLibraryIDsJSON,
		nullStr(row.GeneratedJournalID), nullStr(row.GeneratedInspectionID), nullStr(row.GeneratedReviewID),
		row.CreatedAt, nullStr(row.SubmittedAt), nullStr(row.ApprovedAt),
	)
	return err
}

// GetChecklistSubmission 按 id 查询。
func GetChecklistSubmission(db *sql.DB, id string) (*schema.ChecklistSubmission, error) {
	row := db.QueryRow(`
		SELECT id, checklist_type, code, name, payload_json, payload_schema_version,
			status, submitted_by, tier_acknowledgement, emotion_self_check,
			risk_guardrail_result_json, exception_json, linked_library_ids_json,
			COALESCE(generated_journal_id,''), COALESCE(generated_inspection_id,''),
			COALESCE(generated_review_id,''), created_at,
			COALESCE(submitted_at,''), COALESCE(approved_at,'')
		FROM checklist_submissions WHERE id = ?`, id)

	var cs schema.ChecklistSubmission
	var tierAck sql.NullInt64
	err := row.Scan(
		&cs.ID, &cs.ChecklistType, &cs.Code, &cs.Name, &cs.PayloadJSON, &cs.PayloadSchemaVersion,
		&cs.Status, &cs.SubmittedBy, &tierAck, &cs.EmotionSelfCheck,
		&cs.RiskGuardrailResultJSON, &cs.ExceptionJSON, &cs.LinkedLibraryIDsJSON,
		&cs.GeneratedJournalID, &cs.GeneratedInspectionID, &cs.GeneratedReviewID,
		&cs.CreatedAt, &cs.SubmittedAt, &cs.ApprovedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if tierAck.Valid {
		v := int(tierAck.Int64)
		cs.TierAcknowledgement = &v
	}
	return &cs, nil
}

// ChecklistListFilter 列表过滤。
type ChecklistListFilter struct {
	Status string
	Type   string
	Code   string
	Limit  int
}

// ListChecklistSubmissions 列表（新→旧）。
func ListChecklistSubmissions(db *sql.DB, f ChecklistListFilter) ([]schema.ChecklistSubmission, error) {
	q := `
		SELECT id, checklist_type, code, name, payload_json, payload_schema_version,
			status, submitted_by, tier_acknowledgement, emotion_self_check,
			risk_guardrail_result_json, exception_json, linked_library_ids_json,
			COALESCE(generated_journal_id,''), COALESCE(generated_inspection_id,''),
			COALESCE(generated_review_id,''), created_at,
			COALESCE(submitted_at,''), COALESCE(approved_at,'')
		FROM checklist_submissions WHERE 1=1`
	var args []any
	if f.Status != "" {
		q += " AND status = ?"
		args = append(args, f.Status)
	}
	if f.Type != "" {
		q += " AND checklist_type = ?"
		args = append(args, f.Type)
	}
	if f.Code != "" {
		q += " AND code = ?"
		args = append(args, f.Code)
	}
	q += " ORDER BY created_at DESC"
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	q += fmt.Sprintf(" LIMIT %d", limit)

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []schema.ChecklistSubmission
	for rows.Next() {
		var cs schema.ChecklistSubmission
		var tierAck sql.NullInt64
		if err := rows.Scan(
			&cs.ID, &cs.ChecklistType, &cs.Code, &cs.Name, &cs.PayloadJSON, &cs.PayloadSchemaVersion,
			&cs.Status, &cs.SubmittedBy, &tierAck, &cs.EmotionSelfCheck,
			&cs.RiskGuardrailResultJSON, &cs.ExceptionJSON, &cs.LinkedLibraryIDsJSON,
			&cs.GeneratedJournalID, &cs.GeneratedInspectionID, &cs.GeneratedReviewID,
			&cs.CreatedAt, &cs.SubmittedAt, &cs.ApprovedAt,
		); err != nil {
			return nil, err
		}
		if tierAck.Valid {
			v := int(tierAck.Int64)
			cs.TierAcknowledgement = &v
		}
		out = append(out, cs)
	}
	return out, rows.Err()
}

// UpdateChecklistSubmitted 提交后更新行。
func UpdateChecklistSubmitted(db execer, id, riskJSON, exceptionJSON, emotionSelfCheck string, tierAck *int, submittedAt string) error {
	_, err := db.Exec(`
		UPDATE checklist_submissions SET
			status = 'submitted',
			risk_guardrail_result_json = ?,
			exception_json = ?,
			emotion_self_check = ?,
			tier_acknowledgement = ?,
			submitted_at = ?
		WHERE id = ? AND status = 'draft'`,
		riskJSON, exceptionJSON, emotionSelfCheck, tierAck, submittedAt, id,
	)
	return err
}

// UpdateChecklistRejected 作废 checklist（draft/submitted → rejected）。
func UpdateChecklistRejected(db *sql.DB, id, payloadJSON string) error {
	res, err := db.Exec(`
		UPDATE checklist_submissions SET status = 'rejected', payload_json = ?
		WHERE id = ? AND status IN ('draft', 'submitted')`,
		payloadJSON, id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("reject 失败：id=%s 非 draft/submitted 或不存在", id)
	}
	return nil
}

// ApproveChecklistUpdate approve 后更新 checklist 行。
func ApproveChecklistUpdate(db execer, id, approvedAt, journalID, inspectionID, reviewID string) error {
	_, err := db.Exec(`
		UPDATE checklist_submissions SET
			status = 'approved',
			approved_at = ?,
			generated_journal_id = ?,
			generated_inspection_id = ?,
			generated_review_id = ?
		WHERE id = ? AND status = 'submitted'`,
		approvedAt,
		nullStr(journalID), nullStr(inspectionID), nullStr(reviewID),
		id,
	)
	return err
}

// InsertRiskException 写入 risk_exceptions。
func InsertRiskException(db *sql.DB, row *schema.RiskException) error {
	_, err := db.Exec(`
		INSERT INTO risk_exceptions (
			id, severity, rule_source, rule_id, checklist_submission_id,
			journal_id, exception_reason, expected_compensation, review_date,
			outcome_note, created_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		row.ID, row.Severity, row.RuleSource, row.RuleID, row.ChecklistSubmissionID,
		nullStr(row.JournalID), row.ExceptionReason, row.ExpectedCompensation,
		row.ReviewDate, row.OutcomeNote, row.CreatedAt,
	)
	return err
}

func nullStr(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
