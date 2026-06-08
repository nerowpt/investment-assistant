package backup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/investment-assistant/investment-assistant/internal/core/account"
)

func TestCreateAndListLite(t *testing.T) {
	root := t.TempDir()
	t.Setenv("IA_CONFIG_ROOT", filepath.Join("..", "..", "..", "config"))
	ac, err := account.WithAccount(root, "default")
	if err != nil {
		t.Fatal(err)
	}
	if err := ac.EnsureInitialized(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ac.PortfolioPath(), []byte("schema_version: 1\nmeta:\n  updated_at: \"2026-05-22T12:00:00+08:00\"\n  currency: CNY\npositions: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := Create(ac, ModeLite, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(ac.BackupRoot, ac.AccountID, m.BackupID, "manifest.json")); err != nil {
		t.Fatal(err)
	}
	list, err := List(ac)
	if err != nil || len(list) != 1 {
		t.Fatalf("list=%v err=%v", list, err)
	}
}

func TestPruneKeepsLatest(t *testing.T) {
	root := t.TempDir()
	ac, _ := account.WithAccount(root, "default")
	_ = ac.EnsureInitialized()
	for _, id := range []string{"20260101_010000", "20260102_010000", "20260103_010000"} {
		dir := filepath.Join(ac.BackupRoot, ac.AccountID, id)
		_ = os.MkdirAll(dir, 0o755)
		_ = os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"backup_id":"`+id+`","account_id":"default","mode":"lite"}`), 0o644)
	}
	n, err := Prune(ac, 2)
	if err != nil || n != 1 {
		t.Fatalf("removed=%d err=%v", n, err)
	}
	list, _ := List(ac)
	if len(list) != 2 {
		t.Fatalf("want 2 left, got %d", len(list))
	}
}
