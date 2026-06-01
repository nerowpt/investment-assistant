package sqlstore

import (
	"path/filepath"
	"testing"
)

func TestMigrateUpAddsLotsD8Columns(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "assistant.sqlite")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// 模拟旧版 001：lots 无 D8 列
	if _, err := db.Exec(`
		CREATE TABLE schema_meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		INSERT INTO schema_meta (key, value) VALUES ('schema_version', '1');
		CREATE TABLE lots (
			id TEXT PRIMARY KEY, code TEXT NOT NULL, name TEXT,
			journal_id TEXT NOT NULL, action_type TEXT NOT NULL, position_type TEXT NOT NULL,
			open_at TEXT NOT NULL, close_at TEXT,
			initial_pct REAL NOT NULL, current_pct REAL NOT NULL, cost_basis REAL NOT NULL,
			shares REAL, status TEXT NOT NULL, linked_buy_journal_id TEXT,
			created_at TEXT NOT NULL
		);
	`); err != nil {
		t.Fatal(err)
	}

	if err := MigrateUp(db); err != nil {
		t.Fatal(err)
	}
	for _, col := range []string{"dividends_received", "adjusted_cost_basis", "corporate_actions_json"} {
		ok, err := tableHasColumn(db, "lots", col)
		if err != nil || !ok {
			t.Fatalf("column %s missing: ok=%v err=%v", col, ok, err)
		}
	}
	applied, err := migrationApplied(db, migration002)
	if err != nil || !applied {
		t.Fatalf("migration 002 not marked applied: %v", err)
	}
	ver, _ := SchemaVersion(db)
	if ver != "3" {
		t.Fatalf("schema_version=%q want 3", ver)
	}

	// 幂等：再跑不应报错
	if err := MigrateUp(db); err != nil {
		t.Fatal(err)
	}
}
