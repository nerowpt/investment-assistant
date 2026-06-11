package query

import (
	"path/filepath"
	"testing"

	"github.com/investment-assistant/investment-assistant/internal/core/account"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
	"github.com/shopspring/decimal"
)

func TestBuildPoolWatchingAndResearching(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DATA_ROOT", root)
	t.Setenv("IA_CONFIG_ROOT", filepath.Join("..", "..", "..", "config"))
	t.Setenv("IA_ACCOUNT_ID", "pool-test")
	ac, err := account.WithAccount(root, "pool-test")
	if err != nil {
		t.Fatal(err)
	}
	if err := ac.EnsureInitialized(); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(root, "accounts", "pool-test", "db", "assistant.sqlite")
	db, err := sqlstore.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := sqlstore.MigrateUp(db); err != nil {
		t.Fatal(err)
	}

	wl := &yamlstore.Watchlist{
		SchemaVersion: 1,
		Meta:          yamlstore.WatchlistMeta{UpdatedAt: "2026-06-08T10:00:00+08:00"},
		Items: []yamlstore.WatchlistItem{{
			ID: "w_001", Code: "000858", Name: "五粮液", State: "watching",
			WatchReason: "估值修复", Hypothesis: "业绩拐点", ReviewDate: "2026-07-01",
			KeyMetricsToWatch: []string{"ROE"},
		}},
	}
	if err := yamlstore.SaveWatchlist(ac.WatchlistPath(), wl); err != nil {
		t.Fatal(err)
	}
	p := &yamlstore.Portfolio{
		SchemaVersion: 1,
		Meta:          yamlstore.PortfolioMeta{UpdatedAt: "2026-06-08T10:00:00+08:00", Currency: "CNY"},
		Positions: []yamlstore.PortfolioPosition{{
			Code: "600519", Name: "贵州茅台", State: "holding",
			PositionType: "swing", PositionPct: decimal.NewFromInt(5),
			InvestmentThesis: "长期持有", EntryDate: "2026-01-01",
			LotIDs: []string{}, JournalIDs: []string{},
		}},
	}
	if err := yamlstore.SavePortfolio(ac.PortfolioPath(), p); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		INSERT INTO library_items (id, status, title, source, tier, timestamp, collected_at,
			content_type, media_type, related_stocks_json, tags_json, dedup_key,
			schema_version, reference_count, created_at, updated_at)
		VALUES ('lib_x', 'active', '研报摘要', 'manual', 'B', '2026-06-08', '2026-06-08',
			'text', 'text', '["600519"]', '[]', 'dk1', 1, 0, '2026-06-08', '2026-06-08'),
		       ('lib_y', 'active', '新闻', 'manual', 'C', '2026-06-08', '2026-06-08',
			'text', 'text', '["000001"]', '[]', 'dk2', 1, 0, '2026-06-08', '2026-06-08')
	`)
	if err != nil {
		t.Fatal(err)
	}

	r := NewReader(ac, db)
	pool, err := r.BuildPool("")
	if err != nil {
		t.Fatal(err)
	}
	if len(pool.Zones) != 5 {
		t.Fatalf("zones=%d", len(pool.Zones))
	}
	var watchCnt, researchCnt, swingCnt int
	for _, z := range pool.Zones {
		switch z.ID {
		case "watching":
			watchCnt = z.Count
			if z.Count != 1 || z.Items[0].Code != "000858" {
				t.Fatalf("watching: %+v", z)
			}
		case "researching":
			researchCnt = z.Count
			if z.Count != 1 || z.Items[0].Code != "000001" {
				t.Fatalf("researching: %+v", z)
			}
		case "swing":
			swingCnt = z.Count
			if z.Count != 1 || z.Items[0].Code != "600519" {
				t.Fatalf("swing: %+v", z)
			}
		}
	}
	if watchCnt != 1 || researchCnt != 1 || swingCnt != 1 {
		t.Fatalf("counts watch=%d research=%d swing=%d", watchCnt, researchCnt, swingCnt)
	}

	ctx, err := r.BuildBuyContext("000858", "watchlist", "w_001")
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Prefill.SourceEntry != "from_watchlist" || ctx.Prefill.WatchlistOriginID != "w_001" {
		t.Fatalf("prefill: %+v", ctx.Prefill)
	}
	if ctx.Prefill.BuyReasonSummary != "估值修复" {
		t.Fatalf("reason=%q", ctx.Prefill.BuyReasonSummary)
	}
}
