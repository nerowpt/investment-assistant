package checklist

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/investment-assistant/investment-assistant/internal/core/account"
	"github.com/investment-assistant/investment-assistant/internal/core/ids"
	"github.com/investment-assistant/investment-assistant/internal/core/risk"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore/schema"
	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
)

type stubMarketFetcher struct{}

func (stubMarketFetcher) FetchQuote(_ context.Context, _ string) (float64, float64, string, string, error) {
	return 1500, 1.2, "stub", "A", nil
}

func (stubMarketFetcher) FetchValuation(_ context.Context, _ string) (float64, float64, string, string, error) {
	return 25, 8, "stub", "A", nil
}

func setupChecklistTest(t *testing.T) (*account.Context, *sql.DB, *Service) {
	t.Helper()
	t.Setenv("IA_CONFIG_ROOT", filepath.Join("..", "..", "..", "config"))
	root := t.TempDir()
	ac, err := account.WithAccount(root, "default")
	if err != nil {
		t.Fatal(err)
	}
	if err := ac.EnsureInitialized(); err != nil {
		t.Fatal(err)
	}
	db, err := sqlstore.Open(ac.DBPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := sqlstore.MigrateUp(db); err != nil {
		t.Fatal(err)
	}
	svc := NewService(ac, db)
	svc.SetMarketFetcher(stubMarketFetcher{})
	return ac, db, svc
}

func buyPayloadJSON() string {
	return `{
  "source_entry":"manual","position_type":"core","buy_reason_summary":"测试建仓",
  "investment_thesis":"测试 thesis","expected_return_driver":["earnings_growth"],
  "target_price":1800,"stop_loss":1400,"reversal_conditions":["条件A"],
  "position_size_plan":{"initial_pct":5,"max_pct":10},
  "opportunity_cost_benchmark":"HS300","confidence":"medium","emotion_tag":"calm",
  "identified_risks":["风险A"],"related_library_ids":[],
  "no_library_reason":"测试无 L1","tier_acknowledgement":false,"emotion_retrospect":null
}`
}

func TestApproveBuyCreatesJournalLotPortfolio(t *testing.T) {
	ac, db, svc := setupChecklistTest(t)

	draft, err := svc.CreateDraft(DraftInput{
		ChecklistType: "buy",
		Code:          "600519",
		Name:          "贵州茅台",
		PayloadJSON:   buyPayloadJSON(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Submit(SubmitInput{ID: draft.ID}); err != nil {
		t.Fatal(err)
	}

	res, err := svc.Approve(context.Background(), draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res.JournalID == "" || res.LotID == "" || res.SnapshotID == "" {
		t.Fatalf("missing ids: %+v", res)
	}
	if !res.YAMLSynced {
		t.Fatalf("expected yaml synced sync_repair=%s res=%+v", res.SyncRepairID, res)
	}

	ok, err := sqlstore.JournalExists(db, res.JournalID)
	if err != nil || !ok {
		t.Fatalf("journal missing: ok=%v err=%v", ok, err)
	}
	ok, err = sqlstore.LotExists(db, res.LotID)
	if err != nil || !ok {
		t.Fatalf("lot missing: ok=%v err=%v", ok, err)
	}

	cs, err := sqlstore.GetChecklistSubmission(db, draft.ID)
	if err != nil || cs.Status != "approved" || cs.GeneratedJournalID != res.JournalID {
		t.Fatalf("checklist not approved: %+v err=%v", cs, err)
	}

	port, err := yamlstore.LoadPortfolio(ac.PortfolioPath())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range port.Positions {
		if p.Code == "600519" && p.State == "holding" {
			found = true
			if len(p.LotIDs) != 1 || p.LotIDs[0] != res.LotID {
				t.Fatalf("lot_ids mismatch: %+v", p.LotIDs)
			}
		}
	}
	if !found {
		t.Fatal("portfolio missing holding position")
	}

	res2, err := svc.Approve(context.Background(), draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res2.JournalID != res.JournalID {
		t.Fatalf("idempotent journal mismatch: %s vs %s", res2.JournalID, res.JournalID)
	}
}

func TestApproveBlockedWithoutException(t *testing.T) {
	ac, db, svc := setupChecklistTest(t)
	_ = ac

	id, err := ids.Next(db, "cs")
	if err != nil {
		t.Fatal(err)
	}
	guard := risk.GuardrailResult{
		Scenario:       "buy",
		ApproveBlocked: true,
		HardBlocks: []risk.CheckResult{{
			RuleID: "r004", Severity: "hard_block", Message: "无 L1 引用",
		}},
	}
	guardJSON, _ := json.Marshal(guard)
	row := &schema.ChecklistSubmission{
		ID: id, ChecklistType: "buy", Code: "600519", Name: "贵州茅台",
		PayloadJSON: buyPayloadJSON(), PayloadSchemaVersion: PayloadSchemaVersion,
		Status: "submitted", SubmittedBy: "user",
		RiskGuardrailResultJSON: string(guardJSON),
		CreatedAt: nowISO(), SubmittedAt: nowISO(),
	}
	if err := sqlstore.InsertChecklistSubmission(db, row); err != nil {
		t.Fatal(err)
	}

	_, err = svc.Approve(context.Background(), id)
	if err == nil {
		t.Fatal("expected approve gate error")
	}
}
