// Package ids 生成带日期的业务主键（对齐 03 §9.1）。
package ids

import (
	"database/sql"
	"fmt"
	"time"
)

// Next 分配下一个 ID，格式 {prefix}_{YYYYMMDD}_{seq:03d}。
func Next(db *sql.DB, prefix string) (string, error) {
	date := time.Now().Format("20060102")
	tx, err := db.Begin()
	if err != nil {
		return "", fmt.Errorf("开启事务: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`INSERT OR IGNORE INTO id_sequences (prefix, seq_date, next_seq) VALUES (?, ?, 1)`,
		prefix, date,
	); err != nil {
		return "", fmt.Errorf("初始化序列: %w", err)
	}

	var seq int
	if err := tx.QueryRow(
		`SELECT next_seq FROM id_sequences WHERE prefix = ? AND seq_date = ?`,
		prefix, date,
	).Scan(&seq); err != nil {
		return "", fmt.Errorf("读取序列: %w", err)
	}

	if _, err := tx.Exec(
		`UPDATE id_sequences SET next_seq = next_seq + 1 WHERE prefix = ? AND seq_date = ?`,
		prefix, date,
	); err != nil {
		return "", fmt.Errorf("递增序列: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("提交序列: %w", err)
	}
	return fmt.Sprintf("%s_%s_%03d", prefix, date, seq), nil
}
