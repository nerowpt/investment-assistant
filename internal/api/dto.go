package api

import (
	"encoding/json"

	chksvc "github.com/investment-assistant/investment-assistant/internal/core/checklist"
	"github.com/investment-assistant/investment-assistant/internal/core/doctor"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore/schema"
	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
	"github.com/shopspring/decimal"
	"gopkg.in/yaml.v3"
)

// JournalListItem journal 列表项（H8 前端 snake_case）。
type JournalListItem struct {
	ID                    string `json:"id"`
	ActionType            string `json:"action_type"`
	Code                  string `json:"code"`
	Name                  string `json:"name"`
	ChecklistSubmissionID string `json:"checklist_submission_id,omitempty"`
	LotID                 string `json:"lot_id,omitempty"`
	Summary               string `json:"summary"`
	CreatedAt             string `json:"created_at"`
}

func toJournalListItem(r sqlstore.JournalRow) JournalListItem {
	return JournalListItem{
		ID:                    r.ID,
		ActionType:            r.ActionType,
		Code:                  r.Code,
		Name:                  r.Name,
		ChecklistSubmissionID: r.ChecklistSubmissionID,
		LotID:                 r.LotID,
		Summary:               r.Summary,
		CreatedAt:             r.CreatedAt,
	}
}

func toJournalList(rows []sqlstore.JournalRow) []JournalListItem {
	out := make([]JournalListItem, len(rows))
	for i, r := range rows {
		out[i] = toJournalListItem(r)
	}
	return out
}

// ChecklistListItem checklist 列表项（列表页仅需摘要字段）。
type ChecklistListItem struct {
	ID            string `json:"id"`
	ChecklistType string `json:"checklist_type"`
	Code          string `json:"code"`
	Name          string `json:"name"`
	Status        string `json:"status"`
	Summary       string `json:"summary,omitempty"`
	CreatedAt     string `json:"created_at"`
	ApprovedAt    string `json:"approved_at,omitempty"`
	SubmittedAt   string `json:"submitted_at,omitempty"`
}

func toChecklistListItem(cs schema.ChecklistSubmission) ChecklistListItem {
	return ChecklistListItem{
		ID:            cs.ID,
		ChecklistType: cs.ChecklistType,
		Code:          cs.Code,
		Name:          cs.Name,
		Status:        cs.Status,
		Summary:       summarizeChecklist(cs.ChecklistType, cs.PayloadJSON),
		CreatedAt:     cs.CreatedAt,
		ApprovedAt:    cs.ApprovedAt,
		SubmittedAt:   cs.SubmittedAt,
	}
}

func toChecklistList(rows []schema.ChecklistSubmission) []ChecklistListItem {
	out := make([]ChecklistListItem, len(rows))
	for i, cs := range rows {
		out[i] = toChecklistListItem(cs)
	}
	return out
}

// PortfolioPositionJSON 持仓 JSON 视图（对齐 yaml 字段名）。
type PortfolioPositionJSON struct {
	Code                     string                `json:"code"`
	Name                     string                `json:"name"`
	State                    string                `json:"state"`
	PositionType             string                `json:"position_type,omitempty"`
	PositionPct              string                `json:"position_pct"`
	CostBasis                string                `json:"cost_basis,omitempty"`
	Shares                   *string               `json:"shares,omitempty"`
	SectorID                 string                `json:"sector_id,omitempty"`
	ThesisID                 string                `json:"thesis_id,omitempty"`
	EntryDate                string                `json:"entry_date,omitempty"`
	ClosedAt                 string                `json:"closed_at,omitempty"`
	ThesisVersion            int                   `json:"thesis_version,omitempty"`
	InvestmentThesis         string                `json:"investment_thesis,omitempty"`
	TargetPrice              any                   `json:"target_price,omitempty"`
	StopLoss                 string                `json:"stop_loss,omitempty"`
	ReversalConditions       []string              `json:"reversal_conditions,omitempty"`
	OpportunityCostBenchmark string                `json:"opportunity_cost_benchmark,omitempty"`
	Confidence               string                `json:"confidence,omitempty"`
	RelatedLibraryIDs        []string              `json:"related_library_ids,omitempty"`
	LotIDs                   []string              `json:"lot_ids,omitempty"`
	JournalIDs               []string              `json:"journal_ids,omitempty"`
	WatchlistOriginID        string                `json:"watchlist_origin_id,omitempty"`
	LegacyFlags              []string              `json:"legacy_flags,omitempty"`
	Monitoring               *PositionMonitoringJSON `json:"monitoring,omitempty"`
	Notes                    string                `json:"notes,omitempty"`
}

// PositionMonitoringJSON 巡检摘要 JSON。
type PositionMonitoringJSON struct {
	LastInspectionID  string `json:"last_inspection_id,omitempty"`
	LastInspectionAt  string `json:"last_inspection_at,omitempty"`
	NextInspectionDue string `json:"next_inspection_due,omitempty"`
	Classification    string `json:"classification,omitempty"`
	PlannedAction     string `json:"planned_action,omitempty"`
}

// PortfolioMetaJSON portfolio meta JSON。
type PortfolioMetaJSON struct {
	UpdatedAt      string  `json:"updated_at,omitempty"`
	TotalEquityRef *string `json:"total_equity_ref,omitempty"`
	Currency       string  `json:"currency,omitempty"`
}

func decStr(d decimal.Decimal) string {
	if d.IsZero() {
		return "0"
	}
	return d.String()
}

func decPtrStr(d *decimal.Decimal) *string {
	if d == nil {
		return nil
	}
	s := d.String()
	return &s
}

func decodeYAMLNode(n yaml.Node) any {
	if n.Kind == 0 {
		return nil
	}
	var v any
	if err := n.Decode(&v); err != nil {
		return nil
	}
	return v
}

func toPortfolioPositionJSON(p yamlstore.PortfolioPosition) PortfolioPositionJSON {
	var mon *PositionMonitoringJSON
	if p.Monitoring != nil {
		mon = &PositionMonitoringJSON{
			LastInspectionID:  p.Monitoring.LastInspectionID,
			LastInspectionAt:  p.Monitoring.LastInspectionAt,
			NextInspectionDue: p.Monitoring.NextInspectionDue,
			Classification:    p.Monitoring.Classification,
			PlannedAction:     p.Monitoring.PlannedAction,
		}
	}
	return PortfolioPositionJSON{
		Code:                     p.Code,
		Name:                     p.Name,
		State:                    p.State,
		PositionType:             p.PositionType,
		PositionPct:              decStr(p.PositionPct),
		CostBasis:                decStr(p.CostBasis),
		Shares:                   decPtrStr(p.Shares),
		SectorID:                 p.SectorID,
		ThesisID:                 p.ThesisID,
		EntryDate:                p.EntryDate,
		ClosedAt:                 p.ClosedAt,
		ThesisVersion:            p.ThesisVersion,
		InvestmentThesis:         p.InvestmentThesis,
		TargetPrice:              decodeYAMLNode(p.TargetPrice),
		StopLoss:                 decStr(p.StopLoss),
		ReversalConditions:       p.ReversalConditions,
		OpportunityCostBenchmark: p.OpportunityCostBenchmark,
		Confidence:               p.Confidence,
		RelatedLibraryIDs:        p.RelatedLibraryIDs,
		LotIDs:                   p.LotIDs,
		JournalIDs:               p.JournalIDs,
		WatchlistOriginID:        p.WatchlistOriginID,
		LegacyFlags:              p.LegacyFlags,
		Monitoring:               mon,
		Notes:                    p.Notes,
	}
}

func toPortfolioJSON(data any) any {
	raw, err := json.Marshal(data)
	if err != nil {
		return data
	}
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return data
	}
	positions, _ := m["positions"].([]any)
	if positions == nil {
		return data
	}
	// 重新从原始结构转换，避免 PascalCase 泄漏。
	p, ok := data.(map[string]any)
	if !ok {
		return data
	}
	posSlice, ok := p["positions"].([]yamlstore.PortfolioPosition)
	if !ok {
		return data
	}
	out := make([]PortfolioPositionJSON, len(posSlice))
	for i, pos := range posSlice {
		out[i] = toPortfolioPositionJSON(pos)
	}
	metaJSON := PortfolioMetaJSON{}
	if meta, ok := p["meta"].(yamlstore.PortfolioMeta); ok {
		metaJSON.UpdatedAt = meta.UpdatedAt
		metaJSON.Currency = meta.Currency
		if meta.TotalEquityRef != nil {
			s := meta.TotalEquityRef.String()
			metaJSON.TotalEquityRef = &s
		}
	}
	schemaVer, _ := p["schema_version"].(int)
	return map[string]any{
		"schema_version": schemaVer,
		"meta":           metaJSON,
		"positions":      out,
	}
}

// WatchlistItemJSON 观察池条目 JSON。
type WatchlistItemJSON struct {
	ID        string `json:"id"`
	Code      string `json:"code,omitempty"`
	Name      string `json:"name"`
	WatchType string `json:"watch_type,omitempty"`
	State     string `json:"state,omitempty"`
}

func toWatchlistJSON(data any) any {
	m, ok := data.(map[string]any)
	if !ok {
		return data
	}
	items, ok := m["items"].([]yamlstore.WatchlistItem)
	if !ok {
		return data
	}
	out := make([]WatchlistItemJSON, len(items))
	for i, it := range items {
		out[i] = WatchlistItemJSON{
			ID:        it.ID,
			Code:      it.Code,
			Name:      it.Name,
			WatchType: it.WatchType,
			State:     it.State,
		}
	}
	return map[string]any{"items": out}
}

// DraftResultJSON draft 创建响应。
type DraftResultJSON struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func toDraftResultJSON(r *chksvc.DraftResult) DraftResultJSON {
	return DraftResultJSON{ID: r.ID, Status: r.Status}
}

// SubmitResultJSON submit 响应。
type SubmitResultJSON struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	ApproveBlocked bool   `json:"approve_blocked"`
	HardBlockCount int    `json:"hard_block_count"`
	WarningCount   int    `json:"warning_count"`
}

func toSubmitResultJSON(r *chksvc.SubmitResult) SubmitResultJSON {
	return SubmitResultJSON{
		ID:             r.ID,
		Status:         r.Status,
		ApproveBlocked: r.ApproveBlocked,
		HardBlockCount: r.HardBlockCount,
		WarningCount:   r.WarningCount,
	}
}

// ApproveResultJSON approve 响应。
type ApproveResultJSON struct {
	ChecklistID  string `json:"checklist_id"`
	JournalID    string `json:"journal_id,omitempty"`
	LotID        string `json:"lot_id,omitempty"`
	SnapshotID   string `json:"snapshot_id,omitempty"`
	InspectionID string `json:"inspection_id,omitempty"`
	YAMLSynced   bool   `json:"yaml_synced"`
	SyncRepairID string `json:"sync_repair_id,omitempty"`
}

func toApproveResultJSON(r *chksvc.ApproveResult) ApproveResultJSON {
	return ApproveResultJSON{
		ChecklistID:  r.ChecklistID,
		JournalID:    r.JournalID,
		LotID:        r.LotID,
		SnapshotID:   r.SnapshotID,
		InspectionID: r.InspectionID,
		YAMLSynced:   r.YAMLSynced,
		SyncRepairID: r.SyncRepairID,
	}
}

// DoctorIssueJSON 数据体检问题项。
type DoctorIssueJSON struct {
	Code    string `json:"code,omitempty"`
	Subject string `json:"subject,omitempty"`
	Title   string `json:"title"`
	Detail  string `json:"detail,omitempty"`
	Hint    string `json:"hint,omitempty"`
}

func toDoctorIssueJSON(iss doctor.Issue) DoctorIssueJSON {
	return DoctorIssueJSON{
		Code:    iss.Code,
		Subject: iss.Subject,
		Title:   iss.Title,
		Detail:  iss.Detail,
		Hint:    iss.Hint,
	}
}

func toDoctorIssueFromText(s string) DoctorIssueJSON {
	return DoctorIssueJSON{Title: s}
}
