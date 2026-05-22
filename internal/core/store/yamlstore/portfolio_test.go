package yamlstore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shopspring/decimal"
)

func TestAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	if err := atomicWrite(path, []byte("hello: world\n")); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "hello: world\n" {
		t.Fatalf("内容不符: %q", raw)
	}
}

func TestValidatePortfolio_closed(t *testing.T) {
	p := &Portfolio{
		SchemaVersion: 1,
		Meta:          PortfolioMeta{UpdatedAt: "2026-05-19T10:00:00+08:00"},
		Positions: []PortfolioPosition{
			{
				Code:        "600519",
				State:       "closed",
				PositionPct: decimal.NewFromInt(5),
			},
		},
	}
	issues := ValidatePortfolio(p)
	if len(issues) == 0 {
		t.Fatal("期望 closed position_pct 非零报错")
	}
}

func TestLoadSavePortfolio(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "portfolio.yaml")
	p := &Portfolio{
		SchemaVersion: 1,
		Meta: PortfolioMeta{
			UpdatedAt: "2026-05-19T10:00:00+08:00",
			Currency:  "CNY",
		},
		Positions: []PortfolioPosition{},
	}
	if err := SavePortfolio(path, p); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadPortfolio(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.SchemaVersion != 1 {
		t.Fatalf("schema_version=%d", loaded.SchemaVersion)
	}
}
