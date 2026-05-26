package doctor

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
	"github.com/shopspring/decimal"
)

func TestCheckWatchlist_promotedJournalExists(t *testing.T) {
	db, err := sqlstore.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := sqlstore.MigrateUp(db); err != nil {
		t.Fatal(err)
	}

	w := &yamlstore.Watchlist{
		SchemaVersion: 1,
		Meta:          yamlstore.WatchlistMeta{UpdatedAt: "2026-05-22T10:00:00+08:00"},
		Items: []yamlstore.WatchlistItem{{
			ID:                  "w_20260518_001",
			Code:                "600519",
			Name:                "茅台",
			WatchType:           "stock",
			State:               "removed",
			SourceEntry:         "manual",
			WatchReason:         "r",
			Hypothesis:          "h",
			KeyMetricsToWatch:   []string{"m"},
			ExpectedTrigger:     "t",
			InvalidCondition:    "i",
			ReviewDate:          "2026-06-01",
			CreatedAt:           "2026-05-22T10:00:00+08:00",
			RemovedAt:           "2026-05-23T10:00:00+08:00",
			RemovedReason:       "promoted_to_holding",
			PromotedJournalID:   "j_missing",
		}},
	}
	issues := CheckWatchlist(db, w, nil)
	if len(issues) == 0 {
		t.Fatal("期望 journal 不存在报错")
	}

	_, err = db.Exec(`INSERT INTO journals (id, action_type, code, payload_json, created_at)
		VALUES ('j_missing', 'buy', '600519', '{}', '2026-05-22T10:00:00+08:00')`)
	if err != nil {
		t.Fatal(err)
	}
	issues = CheckWatchlist(db, w, nil)
	for _, iss := range issues {
		if strings.Contains(iss, "promoted_journal_id 不存在") {
			t.Fatalf("仍报错: %s", iss)
		}
	}
}

func TestCheckWatchlist_overlapWithPortfolio(t *testing.T) {
	db := emptyDB(t)
	w := &yamlstore.Watchlist{
		SchemaVersion: 1,
		Meta:          yamlstore.WatchlistMeta{UpdatedAt: "2026-05-22T10:00:00+08:00"},
		Items: []yamlstore.WatchlistItem{{
			ID: "w_1", Code: "600519", Name: "茅台", WatchType: "stock", State: "watching",
			SourceEntry: "manual", WatchReason: "r", Hypothesis: "h",
			KeyMetricsToWatch: []string{"m"}, ExpectedTrigger: "t", InvalidCondition: "i",
			ReviewDate: "2026-06-01", CreatedAt: "2026-05-22T10:00:00+08:00",
		}},
	}
	p := &yamlstore.Portfolio{
		SchemaVersion: 1,
		Meta:          yamlstore.PortfolioMeta{UpdatedAt: "2026-05-22T10:00:00+08:00"},
		Positions: []yamlstore.PortfolioPosition{{
			Code: "600519", Name: "茅台", State: "holding", PositionType: "core",
			PositionPct: decimal.NewFromInt(10),
			EntryDate: "2024-01-01", ThesisVersion: 1, InvestmentThesis: "t",
			StopLoss: decimal.NewFromInt(1), ReversalConditions: []string{"x"},
			OpportunityCostBenchmark: "HS300",
			LotIDs: []string{}, JournalIDs: []string{},
		}},
	}
	issues := CheckWatchlist(db, w, p)
	found := false
	for _, iss := range issues {
		if strings.Contains(iss, "同时在 watchlist(watching) 与 portfolio(holding)") {
			found = true
		}
	}
	if !found {
		t.Fatal("期望 watching/holding 重叠报错")
	}
}

func emptyDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqlstore.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := sqlstore.MigrateUp(db); err != nil {
		t.Fatal(err)
	}
	return db
}
