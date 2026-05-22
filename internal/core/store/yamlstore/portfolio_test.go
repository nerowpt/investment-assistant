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

// TestPortfolio_MonitoringRoundTrip 防止 monitoring 子结构在 load → save 时丢失（docs/06 §D12）。
func TestPortfolio_MonitoringRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "portfolio.yaml")
	original := &Portfolio{
		SchemaVersion: 1,
		Meta:          PortfolioMeta{UpdatedAt: "2026-05-22T10:00:00+08:00", Currency: "CNY"},
		Positions: []PortfolioPosition{
			{
				Code:          "002624",
				Name:          "完美世界",
				State:         "holding",
				PositionType:  "swing",
				PositionPct:   decimal.NewFromInt(8),
				CostBasis:     decimal.NewFromFloat(18.20),
				EntryDate:     "2026-05-18",
				ThesisVersion: 2,
				StopLoss:      decimal.NewFromFloat(12.0),
				Monitoring: &PositionMonitoring{
					LastInspectionID:  "insp_20260618_001",
					LastInspectionAt:  "2026-06-18T09:00:00+08:00",
					NextInspectionDue: "2026-07-18",
					Classification:    "wait_for_style_switch",
					PlannedAction:     "hold",
				},
			},
		},
	}
	if err := SavePortfolio(path, original); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadPortfolio(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Positions) != 1 {
		t.Fatalf("positions=%d", len(loaded.Positions))
	}
	m := loaded.Positions[0].Monitoring
	if m == nil {
		t.Fatal("monitoring 在 round-trip 后丢失")
	}
	if m.LastInspectionID != "insp_20260618_001" || m.Classification != "wait_for_style_switch" || m.PlannedAction != "hold" {
		t.Fatalf("monitoring 字段不一致: %+v", m)
	}

	// 二次 save → load 仍保留
	if err := SavePortfolio(path, loaded); err != nil {
		t.Fatal(err)
	}
	loaded2, err := LoadPortfolio(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded2.Positions[0].Monitoring == nil {
		t.Fatal("monitoring 在二次 round-trip 后丢失")
	}
}
