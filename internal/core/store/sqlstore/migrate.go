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

// MigrateUp 应用嵌入的 up 迁移（幂等：表已存在则跳过 DDL 错误由 IF NOT EXISTS 吸收）。
func MigrateUp(db *sql.DB) error {
	content, err := fs.ReadFile(migrations.UpFS, initialMigration)
	if err != nil {
		return fmt.Errorf("读取迁移 %s: %w", initialMigration, err)
	}
	if _, err := db.Exec(string(content)); err != nil {
		return fmt.Errorf("执行迁移: %w", err)
	}
	return nil
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
