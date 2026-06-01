// Package sqlstore 提供 SQLite 连接与迁移（对齐 04 §10.1、§二十三）。
package sqlstore

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/investment-assistant/investment-assistant/migrations"

	_ "modernc.org/sqlite"
)

const initialMigration = "001_initial.up.sql"

// Open 打开或创建 assistant.sqlite。
func Open(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("创建 db 目录: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite: %w", err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		db.Close()
		return nil, fmt.Errorf("设置 WAL: %w", err)
	}
	return db, nil
}

// MigrateUp 应用嵌入的 up 迁移（001 幂等；002+ 按 schema_meta 追踪，列变更可重复执行）。
func MigrateUp(db *sql.DB) error {
	content, err := fs.ReadFile(migrations.UpFS, initialMigration)
	if err != nil {
		return fmt.Errorf("读取迁移 %s: %w", initialMigration, err)
	}
	if _, err := db.Exec(string(content)); err != nil {
		return fmt.Errorf("执行迁移: %w", err)
	}
	if err := migrateLotsD8Columns(db); err != nil {
		return err
	}
	if err := migrateLotAllocationShares(db); err != nil {
		return err
	}
	return nil
}

const migration003 = "003_lot_allocation_shares"

// migrateLotAllocationShares 为 lot_allocations 补股数/金额盈亏列（B 模型）。
func migrateLotAllocationShares(db *sql.DB) error {
	applied, err := migrationApplied(db, migration003)
	if err != nil {
		return err
	}
	if applied {
		return nil
	}
	cols := []struct{ name, typ string }{
		{"allocated_shares", "REAL"},
		{"proceeds_amount", "REAL"},
		{"realized_pnl_amount", "REAL"},
	}
	for _, c := range cols {
		if err := addColumnIfMissing(db, "lot_allocations", c.name, c.typ); err != nil {
			return fmt.Errorf("迁移 %s lot_allocations.%s: %w", migration003, c.name, err)
		}
	}
	if err := markMigrationApplied(db, migration003); err != nil {
		return err
	}
	_, _ = db.Exec(`UPDATE schema_meta SET value = '3' WHERE key = 'schema_version'`)
	return nil
}

const migration002 = "002_lots_d8_columns"

// migrateLotsD8Columns 为旧版 lots 表补 docs/06 §D8 分红/复权列（幂等）。
func migrateLotsD8Columns(db *sql.DB) error {
	applied, err := migrationApplied(db, migration002)
	if err != nil {
		return err
	}
	if applied {
		return nil
	}
	cols := []struct{ name, typ string }{
		{"dividends_received", "REAL"},
		{"adjusted_cost_basis", "REAL"},
		{"corporate_actions_json", "TEXT"},
	}
	for _, c := range cols {
		if err := addColumnIfMissing(db, "lots", c.name, c.typ); err != nil {
			return fmt.Errorf("迁移 %s lots.%s: %w", migration002, c.name, err)
		}
	}
	if err := markMigrationApplied(db, migration002); err != nil {
		return err
	}
	_, _ = db.Exec(`UPDATE schema_meta SET value = '2' WHERE key = 'schema_version'`)
	return nil
}

func migrationApplied(db *sql.DB, name string) (bool, error) {
	var v string
	err := db.QueryRow(`SELECT value FROM schema_meta WHERE key = ?`, "migration:"+name).Scan(&v)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return v == "1", nil
}

func markMigrationApplied(db *sql.DB, name string) error {
	_, err := db.Exec(`INSERT OR REPLACE INTO schema_meta (key, value) VALUES (?, '1')`, "migration:"+name)
	return err
}

func addColumnIfMissing(db *sql.DB, table, column, colType string) error {
	has, err := tableHasColumn(db, table, column)
	if err != nil {
		return err
	}
	if has {
		return nil
	}
	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, colType))
	return err
}

func tableHasColumn(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

// SchemaVersion 读取 schema_meta.schema_version。
func SchemaVersion(db *sql.DB) (string, error) {
	var v string
	err := db.QueryRow(`SELECT value FROM schema_meta WHERE key = 'schema_version'`).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return v, nil
}

// RequiredTables MVP-1 H0 必须存在的表。
var RequiredTables = []string{
	"id_sequences",
	"sync_repairs",
	"checklist_submissions",
	"journals",
	"data_snapshots",
	"lots",
	"lot_allocations",
	"library_candidates",
	"library_items",
	"schema_meta",
}

// VerifyTables 检查核心表是否齐全。
func VerifyTables(db *sql.DB) ([]string, error) {
	var missing []string
	for _, name := range RequiredTables {
		var n string
		err := db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, name,
		).Scan(&n)
		if err == sql.ErrNoRows {
			missing = append(missing, name)
			continue
		}
		if err != nil {
			return nil, err
		}
	}
	return missing, nil
}

// FormatMissing 格式化缺失表列表供 CLI 输出。
func FormatMissing(missing []string) string {
	if len(missing) == 0 {
		return ""
	}
	return strings.Join(missing, ", ")
}
