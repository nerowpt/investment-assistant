package checklist

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"github.com/investment-assistant/investment-assistant/internal/core/ids"
	"github.com/investment-assistant/investment-assistant/internal/core/risk"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore/schema"
	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
	"github.com/shopspring/decimal"
)

// ApproveResult approve 流水线结果（04 §20.7）。
type ApproveResult struct {
	ChecklistID   string // cs_*
	JournalID     string // j_*（buy/add/import）
	LotID         string // lot_*（buy/add/import）
	SnapshotID  string // snap_*
	InspectionID  string // insp_*
	ReviewID      string // rev_*
	WatchID       string // w_*
	YAMLSynced    bool   // Layer A 是否全部写入成功
	SyncRepairID  string // 非空表示 YAML 写失败，已入 sync_repairs
}

// Approve 将 submitted checklist 落库为 journal/lot/snapshot 并同步 YAML（H5 核心）。
func (s *Service) Approve(ctx context.Context, checklistID string) (*ApproveResult, error) {
	cs, err := sqlstore.GetChecklistSubmission(s.db, checklistID)
	if err != nil {
		return nil, err
	}
	if cs == nil {
		return nil, fmt.Errorf("checklist 不存在: %s", checklistID)
	}
	if cs.Status == "approved" {
		return s.approveIdempotentResult(cs), nil
	}
	if cs.Status != "submitted" {
		return nil, fmt.Errorf("仅 submitted 可 approve（当前 status=%s）", cs.Status)
	}
	if cs.SubmittedBy != "user" {
		return nil, fmt.Errorf("approve 前须 author:user（当前 submitted_by=%s）", cs.SubmittedBy)
	}
	if err := s.checkApproveGate(cs); err != nil {
		return nil, err
	}

	switch cs.ChecklistType {
	case "buy":
		return s.approveBuy(ctx, cs)
	case "add":
		return s.approveAdd(ctx, cs)
	case "watch":
		return s.approveWatch(ctx, cs)
	case "inspect":
		return s.approveInspect(ctx, cs)
	case "review":
		return s.approveReview(ctx, cs)
	case "import":
		return s.approveImport(ctx, cs)
	case "sell":
		return nil, fmt.Errorf("sell approve 在 H6 实现（lot FIFO）")
	default:
		return nil, fmt.Errorf("未支持的 checklist_type: %s", cs.ChecklistType)
	}
}

func (s *Service) checkApproveGate(cs *schema.ChecklistSubmission) error {
	if cs.RiskGuardrailResultJSON == "" || cs.RiskGuardrailResultJSON == "{}" {
		return nil
	}
	var guard risk.GuardrailResult
	if err := json.Unmarshal([]byte(cs.RiskGuardrailResultJSON), &guard); err != nil {
		return nil
	}
	if !guard.ApproveBlocked {
		return nil
	}
	if cs.ExceptionJSON == "" {
		return fmt.Errorf("approve 被 M7 门禁拦截：须先在 submit 时提供 exception_json\n%s",
			risk.FormatChecksSummary(guard.HardBlocks, guard.Warnings))
	}
	return risk.ValidateException(guard.HardBlocks, guard.Warnings, cs.ExceptionJSON)
}

func (s *Service) approveIdempotentResult(cs *schema.ChecklistSubmission) *ApproveResult {
	return &ApproveResult{
		ChecklistID:  cs.ID,
		JournalID:    cs.GeneratedJournalID,
		InspectionID: cs.GeneratedInspectionID,
		ReviewID:     cs.GeneratedReviewID,
		YAMLSynced:   true,
	}
}

func (s *Service) approveBuy(ctx context.Context, cs *schema.ChecklistSubmission) (*ApproveResult, error) {
	payload, err := ParseBuyPayload(cs.PayloadJSON)
	if err != nil {
		return nil, err
	}
	initialPct := decimalFromPayloadPlan(payload.PositionSizePlan, "initial_pct")
	snapJSON, err := BuildBuySnapshot(ctx, cs.Code, s.market)
	if err != nil {
		return nil, err
	}

	journalID, err := ids.Next(s.db, "j")
	if err != nil {
		return nil, err
	}
	lotID, err := ids.Next(s.db, "lot")
	if err != nil {
		return nil, err
	}
	snapID, err := ids.Next(s.db, "snap")
	if err != nil {
		return nil, err
	}

	costBasis := decimal.Zero
	if s.market != nil {
		if price, _, _, _, qErr := s.market.FetchQuote(ctx, cs.Code); qErr == nil {
			costBasis = decimal.NewFromFloat(price)
		}
	}
	now := nowISO()
	today := time.Now().Format("2006-01-02")
	initPctStr := initialPct.StringFixed(4)
	costStr := costBasis.StringFixed(4)

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if err := sqlstore.InsertJournal(tx, &schema.Journal{
		ID: journalID, ActionType: schema.JournalActionBuy,
		Code: cs.Code, Name: cs.Name, ChecklistSubmissionID: cs.ID,
		DataSnapshotID: snapID, PayloadJSON: cs.PayloadJSON,
		Summary: payload.BuyReasonSummary, LotID: lotID, CreatedAt: now,
	}); err != nil {
		return nil, err
	}
	if err := sqlstore.InsertDataSnapshot(tx, &schema.DataSnapshot{
		ID: snapID, JournalID: journalID, SnapshotJSON: snapJSON,
		SchemaVersion: snapshotSchemaVersion, CreatedAt: now,
	}); err != nil {
		return nil, err
	}
	if err := sqlstore.InsertLot(tx, &schema.Lot{
		ID: lotID, Code: cs.Code, Name: cs.Name, JournalID: journalID,
		ActionType: schema.JournalActionBuy, PositionType: payload.PositionType,
		OpenAt: today, InitialPct: initPctStr, CurrentPct: initPctStr,
		CostBasis: costStr, Status: schema.LotStatusOpen, CreatedAt: now,
	}); err != nil {
		return nil, err
	}
	if err := sqlstore.UpdateRiskExceptionsJournalID(tx, cs.ID, journalID); err != nil {
		return nil, err
	}
	if err := sqlstore.ApproveChecklistUpdate(tx, cs.ID, now, journalID, "", ""); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	port, _ := yamlstore.LoadPortfolio(s.ac.PortfolioPath())
	port, err = BuildBuyPortfolioPatch(port, cs, payload, journalID, lotID, costBasis, initialPct)
	if err != nil {
		return nil, err
	}
	bundle := &YamlBundle{Portfolio: port}
	res := &ApproveResult{
		ChecklistID: cs.ID, JournalID: journalID, LotID: lotID, SnapshotID: snapID,
	}
	if err := ApplyYamlBundle(s.ac, bundle); err != nil {
		srID, _ := RecordSyncRepair(s.db, s.ac, cs.ID, journalID, bundle, err)
		res.SyncRepairID = srID
	} else {
		res.YAMLSynced = true
	}
	return res, nil
}

func (s *Service) approveAdd(ctx context.Context, cs *schema.ChecklistSubmission) (*ApproveResult, error) {
	payload, err := ParseAddPayload(cs.PayloadJSON)
	if err != nil {
		return nil, err
	}
	addPct := decimal.NewFromFloat(payload.AddPct)
	snapJSON, err := BuildBuySnapshot(ctx, cs.Code, s.market)
	if err != nil {
		return nil, err
	}

	journalID, err := ids.Next(s.db, "j")
	if err != nil {
		return nil, err
	}
	lotID, err := ids.Next(s.db, "lot")
	if err != nil {
		return nil, err
	}
	snapID, err := ids.Next(s.db, "snap")
	if err != nil {
		return nil, err
	}

	now := nowISO()
	today := time.Now().Format("2006-01-02")
	pctStr := addPct.StringFixed(4)
	posType := payload.PositionType
	if posType == "" {
		posType = "core"
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if err := sqlstore.InsertJournal(tx, &schema.Journal{
		ID: journalID, ActionType: schema.JournalActionAdd,
		Code: cs.Code, Name: cs.Name, ChecklistSubmissionID: cs.ID,
		DataSnapshotID: snapID, PayloadJSON: cs.PayloadJSON,
		Summary: "add " + cs.Code, LotID: lotID, CreatedAt: now,
	}); err != nil {
		return nil, err
	}
	if err := sqlstore.InsertDataSnapshot(tx, &schema.DataSnapshot{
		ID: snapID, JournalID: journalID, SnapshotJSON: snapJSON,
		SchemaVersion: snapshotSchemaVersion, CreatedAt: now,
	}); err != nil {
		return nil, err
	}
	if err := sqlstore.InsertLot(tx, &schema.Lot{
		ID: lotID, Code: cs.Code, Name: cs.Name, JournalID: journalID,
		ActionType: schema.JournalActionAdd, PositionType: posType,
		OpenAt: today, InitialPct: pctStr, CurrentPct: pctStr,
		CostBasis: "0", Status: schema.LotStatusOpen,
		LinkedBuyJournalID: payload.LinkedBuyJournalID, CreatedAt: now,
	}); err != nil {
		return nil, err
	}
	if err := sqlstore.UpdateRiskExceptionsJournalID(tx, cs.ID, journalID); err != nil {
		return nil, err
	}
	if err := sqlstore.ApproveChecklistUpdate(tx, cs.ID, now, journalID, "", ""); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	port, err := yamlstore.LoadPortfolio(s.ac.PortfolioPath())
	if err != nil {
		return nil, err
	}
	port, err = BuildAddPortfolioPatch(port, cs, payload, journalID, lotID, addPct)
	if err != nil {
		return nil, err
	}
	bundle := &YamlBundle{Portfolio: port}
	res := &ApproveResult{
		ChecklistID: cs.ID, JournalID: journalID, LotID: lotID, SnapshotID: snapID,
	}
	if err := ApplyYamlBundle(s.ac, bundle); err != nil {
		srID, _ := RecordSyncRepair(s.db, s.ac, cs.ID, journalID, bundle, err)
		res.SyncRepairID = srID
	} else {
		res.YAMLSynced = true
	}
	return res, nil
}

func (s *Service) approveWatch(ctx context.Context, cs *schema.ChecklistSubmission) (*ApproveResult, error) {
	_ = ctx
	payload, err := ParseWatchPayload(cs.PayloadJSON)
	if err != nil {
		return nil, err
	}
	watchID, err := ids.Next(s.db, "w")
	if err != nil {
		return nil, err
	}
	now := nowISO()

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := sqlstore.ApproveChecklistUpdate(tx, cs.ID, now, "", "", ""); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	wl, _ := yamlstore.LoadWatchlist(s.ac.WatchlistPath())
	wl, err = BuildWatchlistPatch(wl, cs, payload, watchID)
	if err != nil {
		return nil, err
	}
	bundle := &YamlBundle{Watchlist: wl}
	res := &ApproveResult{ChecklistID: cs.ID, WatchID: watchID}
	if err := ApplyYamlBundle(s.ac, bundle); err != nil {
		srID, _ := RecordSyncRepair(s.db, s.ac, cs.ID, "", bundle, err)
		res.SyncRepairID = srID
	} else {
		res.YAMLSynced = true
	}
	return res, nil
}

func (s *Service) approveInspect(ctx context.Context, cs *schema.ChecklistSubmission) (*ApproveResult, error) {
	_ = ctx
	var raw map[string]any
	if err := json.Unmarshal([]byte(cs.PayloadJSON), &raw); err != nil {
		return nil, err
	}
	inspID, err := ids.Next(s.db, "insp")
	if err != nil {
		return nil, err
	}
	now := nowISO()
	judgment, _ := json.Marshal(raw)

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := sqlstore.InsertInspectionRecord(tx, &schema.InspectionRecord{
		ID: inspID, ChecklistSubmissionID: cs.ID, Code: cs.Code,
		InspectionType: strMap(raw, "inspection_type"),
		LinkedBuyJournalID: strMap(raw, "linked_buy_journal_id"),
		UserJudgmentJSON: string(judgment), CreatedAt: now,
	}); err != nil {
		return nil, err
	}
	if err := sqlstore.ApproveChecklistUpdate(tx, cs.ID, now, "", inspID, ""); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &ApproveResult{ChecklistID: cs.ID, InspectionID: inspID, YAMLSynced: true}, nil
}

func (s *Service) approveReview(ctx context.Context, cs *schema.ChecklistSubmission) (*ApproveResult, error) {
	_ = ctx
	var raw map[string]any
	if err := json.Unmarshal([]byte(cs.PayloadJSON), &raw); err != nil {
		return nil, err
	}
	revID, err := ids.Next(s.db, "rev")
	if err != nil {
		return nil, err
	}
	now := nowISO()
	judgment, _ := json.Marshal(raw)

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := sqlstore.InsertReviewReport(tx, &schema.ReviewReport{
		ID: revID, ChecklistSubmissionID: cs.ID,
		ReviewType: strMap(raw, "review_type"),
		PeriodStart: strMap(raw, "period_start"),
		PeriodEnd:   strMap(raw, "period_end"),
		UserJudgmentJSON: string(judgment), CreatedAt: now,
	}); err != nil {
		return nil, err
	}
	if err := sqlstore.ApproveChecklistUpdate(tx, cs.ID, now, "", "", revID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &ApproveResult{ChecklistID: cs.ID, ReviewID: revID, YAMLSynced: true}, nil
}

func (s *Service) approveImport(ctx context.Context, cs *schema.ChecklistSubmission) (*ApproveResult, error) {
	_ = ctx
	var raw map[string]any
	if err := json.Unmarshal([]byte(cs.PayloadJSON), &raw); err != nil {
		return nil, err
	}
	positions, _ := raw["positions"].([]any)
	if len(positions) == 0 {
		return nil, fmt.Errorf("import payload.positions 为空")
	}

	port, _ := yamlstore.LoadPortfolio(s.ac.PortfolioPath())
	if port == nil {
		port = &yamlstore.Portfolio{SchemaVersion: yamlstore.PortfolioSchemaVersion}
	}
	now := nowISO()
	today := time.Now().Format("2006-01-02")
	var firstJournalID string

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	for _, item := range positions {
		pm, ok := item.(map[string]any)
		if !ok {
			continue
		}
		code := strMap(pm, "code")
		name := strMap(pm, "name")
		pct := anyToDecimal(pm["position_pct"])
		cost := anyToDecimal(pm["cost_basis"])
		posType := strMap(pm, "position_type")
		if posType == "" {
			posType = "swing"
		}

		journalID, err := ids.Next(s.db, "j")
		if err != nil {
			return nil, err
		}
		if firstJournalID == "" {
			firstJournalID = journalID
		}
		lotID, err := ids.Next(s.db, "lot")
		if err != nil {
			return nil, err
		}
		itemPayload, _ := json.Marshal(pm)
		pctStr := pct.StringFixed(4)
		costStr := cost.StringFixed(4)

		if err := sqlstore.InsertJournal(tx, &schema.Journal{
			ID: journalID, ActionType: schema.JournalActionImport,
			Code: code, Name: name, ChecklistSubmissionID: cs.ID,
			PayloadJSON: string(itemPayload), Summary: "import " + code,
			LotID: lotID, CreatedAt: now,
		}); err != nil {
			return nil, err
		}
		if err := sqlstore.InsertLot(tx, &schema.Lot{
			ID: lotID, Code: code, Name: name, JournalID: journalID,
			ActionType: schema.JournalActionImport, PositionType: posType,
			OpenAt: today, InitialPct: pctStr, CurrentPct: pctStr,
			CostBasis: costStr, Status: schema.LotStatusOpen, CreatedAt: now,
		}); err != nil {
			return nil, err
		}

		legacyFlags := stringListAny(pm["legacy_flags"])
		port.Positions = append(port.Positions, yamlstore.PortfolioPosition{
			Code: code, Name: name, State: "holding", PositionType: posType,
			PositionPct: pct, CostBasis: cost, EntryDate: today, ThesisVersion: 1,
			InvestmentThesis: strMap(pm, "import_thesis_summary"),
			LotIDs: []string{lotID}, JournalIDs: []string{journalID},
			LegacyFlags: legacyFlags,
			ReversalConditions: []string{"import 占位"},
		})
	}

	if err := sqlstore.UpdateRiskExceptionsJournalID(tx, cs.ID, firstJournalID); err != nil {
		return nil, err
	}
	if err := sqlstore.ApproveChecklistUpdate(tx, cs.ID, now, firstJournalID, "", ""); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	bundle := &YamlBundle{Portfolio: port}
	res := &ApproveResult{ChecklistID: cs.ID, JournalID: firstJournalID, YAMLSynced: true}
	if err := ApplyYamlBundle(s.ac, bundle); err != nil {
		srID, _ := RecordSyncRepair(s.db, s.ac, cs.ID, firstJournalID, bundle, err)
		res.SyncRepairID = srID
		res.YAMLSynced = false
	}
	return res, nil
}

func strMap(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func stringListAny(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, it := range arr {
		if s, ok := it.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// SetMarketFetcher 注入 worker 行情源（approve 时冻结 snapshot）。
func (s *Service) SetMarketFetcher(f MarketFetcher) {
	s.market = f
}
