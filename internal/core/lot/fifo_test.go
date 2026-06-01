package lot

import (
	"testing"

	"github.com/shopspring/decimal"
)

func d(v float64) decimal.Decimal { return decimal.NewFromFloat(v) }

func TestRecommendAllocationsSharesFIFO(t *testing.T) {
	lots := []OpenLot{
		{ID: "lot_1", OpenAt: "2026-01-01", CurrentPct: d(3), CurrentShares: d(300), CostBasis: d(10)},
		{ID: "lot_2", OpenAt: "2026-02-01", CurrentPct: d(5), CurrentShares: d(500), CostBasis: d(12)},
	}
	plan, err := RecommendAllocationsShares(d(400), lots)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 2 || plan[0].LotID != "lot_1" || !plan[0].AllocatedShares.Equal(d(300)) {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	if !plan[1].AllocatedShares.Equal(d(100)) {
		t.Fatalf("lot2 shares=%s", plan[1].AllocatedShares)
	}
}

func TestValidatePlanSharesMismatch(t *testing.T) {
	lots := []OpenLot{{ID: "lot_1", OpenAt: "2026-01-01", CurrentShares: d(500), CurrentPct: d(5)}}
	plan := []PlanItem{{LotID: "lot_1", AllocatedShares: d(100)}}
	if err := ValidatePlanShares(d(200), lots, plan); err == nil {
		t.Fatal("expected sum mismatch")
	}
}

func TestRealizedPnLAmount(t *testing.T) {
	got := RealizedPnLAmount(d(11), d(10), d(100))
	if !got.Equal(d(100)) {
		t.Fatalf("got %s", got)
	}
}

func TestLotStatusAfterShareSell(t *testing.T) {
	if LotStatusAfterShareSell(d(100), d(100)) != "closed" {
		t.Fatal("full sell should close")
	}
	if LotStatusAfterShareSell(d(100), d(40)) != "partial" {
		t.Fatal("partial sell")
	}
}
