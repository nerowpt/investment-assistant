package checklist

import (
	"context"
	"testing"

	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
	"github.com/shopspring/decimal"
)

func TestApproveSellFIFO(t *testing.T) {
	ac, db, svc := setupChecklistTest(t)

	// 建仓 5%
	draft, err := svc.CreateDraft(DraftInput{
		ChecklistType: "buy", Code: "600519", Name: "贵州茅台", PayloadJSON: buyPayloadJSON(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Submit(SubmitInput{ID: draft.ID}); err != nil {
		t.Fatal(err)
	}
	buyRes, err := svc.Approve(context.Background(), draft.ID)
	if err != nil {
		t.Fatal(err)
	}

	// 加仓 3% → 第二个 lot
	addPayload := `{
  "linked_buy_journal_id":"` + buyRes.JournalID + `",
  "add_trigger":"thesis_strengthened","add_reason_summary":"验证通过",
  "thesis_change":"strengthened","add_pct":3,"max_pct_after_add":10,
  "execution_price":1520,"shares":300,
  "emotion_tag":"calm","related_library_ids":[],"tier_acknowledgement":false,"emotion_retrospect":null
}`
	addDraft, err := svc.CreateDraft(DraftInput{
		ChecklistType: "add", Code: "600519", Name: "贵州茅台", PayloadJSON: addPayload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Submit(SubmitInput{ID: addDraft.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Approve(context.Background(), addDraft.ID); err != nil {
		t.Fatal(err)
	}

	sellPayload := `{
  "sell_type":"reduce","sell_trigger":"target_reached","sell_reason":"target_achieved",
  "sell_reason_detail":"测试减仓","sell_shares":400,"execution_price":1600,
  "emotion_tag":"calm","lesson":"测试教训",
  "lot_allocation_plan":[],"emotion_retrospect":null
}`
	sellDraft, err := svc.CreateDraft(DraftInput{
		ChecklistType: "sell", Code: "600519", Name: "贵州茅台", PayloadJSON: sellPayload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Submit(SubmitInput{ID: sellDraft.ID}); err != nil {
		t.Fatal(err)
	}

	plan, err := svc.PlanSell(sellDraft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Plan) != 1 || plan.MatchMethod != "recommended_fifo" {
		t.Fatalf("unexpected fifo plan: %+v", plan)
	}
	if plan.Plan[0].LotID != buyRes.LotID {
		t.Fatalf("fifo should hit first lot")
	}

	sellRes, err := svc.Approve(context.Background(), sellDraft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sellRes.JournalID == "" {
		t.Fatal("missing sell journal")
	}

	var allocCount int
	if err := db.QueryRow(`SELECT COUNT(1) FROM lot_allocations WHERE sell_journal_id = ?`, sellRes.JournalID).Scan(&allocCount); err != nil {
		t.Fatal(err)
	}
	if allocCount != 1 {
		t.Fatalf("allocations=%d", allocCount)
	}

	var matchMethod string
	if err := db.QueryRow(`SELECT match_method FROM lot_allocations WHERE sell_journal_id = ? LIMIT 1`, sellRes.JournalID).Scan(&matchMethod); err != nil {
		t.Fatal(err)
	}
	if matchMethod != "recommended_fifo" {
		t.Fatalf("match_method=%s", matchMethod)
	}

	port, err := yamlstore.LoadPortfolio(ac.PortfolioPath())
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range port.Positions {
		if p.Code == "600519" && p.State == "holding" {
			if !p.PositionPct.Equal(decimal.NewFromInt(4)) {
				t.Fatalf("position_pct=%s want 4", p.PositionPct)
			}
		}
	}

	// 第一个 lot 应 closed（initial 5 全卖）
	var status string
	if err := db.QueryRow(`SELECT status FROM lots WHERE id = ?`, buyRes.LotID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "partial" {
		t.Fatalf("first lot status=%s want partial", status)
	}
}

func TestApproveSellUserAdjusted(t *testing.T) {
	ac, db, svc := setupChecklistTest(t)
	_ = ac

	draft, err := svc.CreateDraft(DraftInput{
		ChecklistType: "buy", Code: "000001", Name: "平安银行", PayloadJSON: buyPayloadJSON(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Submit(SubmitInput{ID: draft.ID}); err != nil {
		t.Fatal(err)
	}
	buyRes, err := svc.Approve(context.Background(), draft.ID)
	if err != nil {
		t.Fatal(err)
	}

	sellPayload := `{
  "sell_type":"reduce","sell_trigger":"manual","sell_reason":"other",
  "sell_shares":200,"execution_price":1550,"emotion_tag":"calm","lesson":"用户调整 plan",
  "lot_allocation_plan":[{"lot_id":"` + buyRes.LotID + `","allocated_shares":200,"user_adjusted":true}],
  "emotion_retrospect":null
}`
	sellDraft, err := svc.CreateDraft(DraftInput{
		ChecklistType: "sell", Code: "000001", Name: "平安银行", PayloadJSON: sellPayload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Submit(SubmitInput{ID: sellDraft.ID}); err != nil {
		t.Fatal(err)
	}
	sellRes, err := svc.Approve(context.Background(), sellDraft.ID)
	if err != nil {
		t.Fatal(err)
	}
	var method string
	if err := db.QueryRow(`SELECT match_method FROM lot_allocations WHERE sell_journal_id=?`, sellRes.JournalID).Scan(&method); err != nil {
		t.Fatal(err)
	}
	if method != "user_adjusted" {
		t.Fatalf("method=%s", method)
	}
}
