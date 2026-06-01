package doctor

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
	"github.com/shopspring/decimal"
)

func setupDoctorDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "assistant.sqlite")
	db, err := sqlstore.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := sqlstore.MigrateUp(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestCheckPortfolioMissingLotHasActionableHint(t *testing.T) {
	db := setupDoctorDB(t)
	p := &yamlstore.Portfolio{
		SchemaVersion: yamlstore.PortfolioSchemaVersion,
		Meta:          yamlstore.PortfolioMeta{UpdatedAt: "2026-05-22T12:00:00+08:00", Currency: "CNY"},
		Positions: []yamlstore.PortfolioPosition{{
			Code: "600519", Name: "贵州茅台", State: "holding",
			PositionPct: decimal.NewFromInt(5),
			LotIDs:      []string{"lot_ghost"},
			JournalIDs:  []string{"j_ghost"},
		}},
	}
	issues := CheckPortfolio(db, p)
	if len(issues) < 2 {
		t.Fatalf("expected lot+journal issues, got %d", len(issues))
	}
	if issues[0].Code != "P001" || issues[0].Hint == "" {
		t.Fatalf("lot issue: %+v", issues[0])
	}
	out := FormatIssues(issues)
	if !containsAll(out, "发现:", "处理:", "P001", "600519") {
		t.Fatalf("format missing fields:\n%s", out)
	}
}

func TestCheckPortfolioPctMismatch(t *testing.T) {
	db := setupDoctorDB(t)
	_, err := db.Exec(`
		INSERT INTO journals (id, action_type, code, payload_json, created_at)
		VALUES ('j_1', 'buy', '600519', '{}', '2026-05-27T10:00:00+08:00');
		INSERT INTO lots (id, code, journal_id, action_type, position_type, open_at,
			initial_pct, current_pct, cost_basis, status, created_at)
		VALUES ('lot_1', '600519', 'j_1', 'buy', 'core', '2026-05-27', 2.5, 2.5, 1680, 'open', '2026-05-27T10:00:00+08:00');
	`)
	if err != nil {
		t.Fatal(err)
	}
	p := &yamlstore.Portfolio{
		SchemaVersion: yamlstore.PortfolioSchemaVersion,
		Meta:          yamlstore.PortfolioMeta{UpdatedAt: "2026-05-22T12:00:00+08:00", Currency: "CNY"},
		Positions: []yamlstore.PortfolioPosition{{
			Code: "600519", State: "holding",
			PositionPct: decimal.NewFromInt(5),
			LotIDs:      []string{"lot_1"},
			JournalIDs:  []string{"j_1"},
		}},
	}
	issues := CheckPortfolio(db, p)
	var pctIssue *Issue
	for i := range issues {
		if issues[i].Code == "P004" {
			pctIssue = &issues[i]
			break
		}
	}
	if pctIssue == nil {
		t.Fatalf("expected P004, got %+v", issues)
	}
	if !containsAll(pctIssue.Detail, "2.5", "5") {
		t.Fatalf("detail=%q", pctIssue.Detail)
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
