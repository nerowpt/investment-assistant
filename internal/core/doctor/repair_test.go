package doctor

import (
	"testing"

	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
	"github.com/shopspring/decimal"
)

func TestBuildAndApplyPortfolioRepairs(t *testing.T) {
	db := setupDoctorDB(t)
	p := &yamlstore.Portfolio{
		SchemaVersion: yamlstore.PortfolioSchemaVersion,
		Meta:          yamlstore.PortfolioMeta{UpdatedAt: "2026-05-22T12:00:00+08:00", Currency: "CNY"},
		Positions: []yamlstore.PortfolioPosition{{
			Code: "002624", Name: "完美世界", State: "holding",
			PositionPct: decimal.NewFromInt(8),
			LotIDs:      []string{"lot_20260518_001"},
			JournalIDs:  []string{"j_20260518_001"},
		}},
	}
	plan := BuildPortfolioRepairPlan(db, p)
	if len(plan) < 2 {
		t.Fatalf("expected repair plan, got %d: %+v", len(plan), plan)
	}
	applies := make([]RepairApply, 0, len(plan))
	for _, a := range plan {
		applies = append(applies, RepairApply{ID: a.ID, Enabled: true})
	}
	fixed, err := ApplyPortfolioRepairs(p, plan, applies)
	if err != nil {
		t.Fatal(err)
	}
	remaining := CheckPortfolio(db, fixed)
	if len(remaining) > 0 {
		t.Fatalf("expected clean after repair, got: %s", FormatIssues(remaining))
	}
	if len(fixed.Positions) != 0 && !fixed.Positions[0].PositionPct.IsZero() {
		// 若未删 position，则 pct 应为 0
		for _, pos := range fixed.Positions {
			if pos.Code == "002624" && !pos.PositionPct.IsZero() {
				t.Fatalf("position_pct=%s", pos.PositionPct)
			}
		}
	}
}
