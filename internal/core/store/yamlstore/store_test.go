package yamlstore

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/shopspring/decimal"
)

func samplePortfolio() *Portfolio {
	return &Portfolio{
		SchemaVersion: 1,
		Meta:          PortfolioMeta{UpdatedAt: "2026-05-22T10:00:00+08:00", Currency: "CNY"},
		Positions: []PortfolioPosition{
			{
				Code:        "002624",
				Name:        "完美世界",
				State:       "holding",
				PositionPct: decimal.NewFromInt(8),
				CostBasis:   decimal.NewFromFloat(18.20),
				EntryDate:   "2026-05-18",
			},
		},
	}
}

func TestFilePortfolioStore_RoundTrip(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "portfolio.yaml")
	s := NewFilePortfolioStore()

	if err := s.Save(ctx, path, samplePortfolio()); err != nil {
		t.Fatal(err)
	}

	loaded, err := s.Load(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Positions) != 1 || loaded.Positions[0].Code != "002624" {
		t.Fatalf("file store load 不一致: %+v", loaded.Positions)
	}
}

func TestFilePortfolioStore_NotFound(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "missing.yaml")
	s := NewFilePortfolioStore()
	_, err := s.Load(ctx, path)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("缺失文件期望 ErrNotFound，实际 %v", err)
	}
}

func TestMemoryPortfolioStore_IsolationFromExternalMutation(t *testing.T) {
	ctx := context.Background()
	path := "/virtual/default/portfolio.yaml"
	s := NewMemoryPortfolioStore()

	p := samplePortfolio()
	if err := s.Save(ctx, path, p); err != nil {
		t.Fatal(err)
	}

	// 外部修改原对象不应影响 store 内部副本
	p.Positions[0].Code = "TAMPERED"

	loaded, err := s.Load(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Positions[0].Code != "002624" {
		t.Fatalf("memory store 未做深拷贝，外部修改污染了内部状态")
	}
}

func TestMemoryPortfolioStore_NotFound(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryPortfolioStore()
	_, err := s.Load(ctx, "/virtual/none")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("空 store 期望 ErrNotFound，实际 %v", err)
	}
}
