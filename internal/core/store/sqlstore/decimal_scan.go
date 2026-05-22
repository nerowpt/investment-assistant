// Package sqlstore 中关于 DECIMAL 字段读写的安全通道。
//
// 背景：T24 / docs/06 §D11 要求所有金融 DECIMAL 字段（仓位比例、成本价、收益率等）
// 在 Go 层必须用 shopspring/decimal，禁止经过 float64 中转。
//
// 实现策略：SQLite REAL 列读取时，统一用 SQL `CAST(col AS TEXT)` 把存储类强制
// 转为字符串再 Scan 到 sql.NullString，最后 decimal.NewFromString 解析。
// 这样即使物理存储是 REAL，Go 层也不会出现 0.1+0.2=0.30000000000000004 类误差。
package sqlstore

import (
	"database/sql"
	"fmt"

	"github.com/shopspring/decimal"
)

// ScanDecimalRow 读取单值 DECIMAL（query 必须用 CAST(col AS TEXT) 包裹目标列）。
// 适合 SELECT SUM(...) 之类只返回一行一列的查询。
func ScanDecimalRow(row *sql.Row) (decimal.Decimal, error) {
	var s sql.NullString
	if err := row.Scan(&s); err != nil {
		return decimal.Zero, err
	}
	return parseNullDecimal(s)
}

// ScanDecimalFromRows 用于 rows.Next() 循环中读取已经 CAST AS TEXT 的列。
// 调用方负责 rows.Next() 与列顺序；本函数只处理当前行第一列。
func ScanDecimalFromRows(rows *sql.Rows) (decimal.Decimal, error) {
	var s sql.NullString
	if err := rows.Scan(&s); err != nil {
		return decimal.Zero, err
	}
	return parseNullDecimal(s)
}

// parseNullDecimal 把 nullable text 安全转为 decimal；NULL 视作 0。
func parseNullDecimal(s sql.NullString) (decimal.Decimal, error) {
	if !s.Valid || s.String == "" {
		return decimal.Zero, nil
	}
	d, err := decimal.NewFromString(s.String)
	if err != nil {
		return decimal.Zero, fmt.Errorf("解析 decimal %q: %w", s.String, err)
	}
	return d, nil
}

// SumDecimalColumn 对单列 DECIMAL 做求和（应用层 decimal 累加，不依赖 SQLite SUM）。
// query 必须 SELECT 一个 CAST(col AS TEXT) 列。
func SumDecimalColumn(db *sql.DB, query string, args ...any) (decimal.Decimal, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return decimal.Zero, err
	}
	defer rows.Close()

	sum := decimal.Zero
	for rows.Next() {
		v, err := ScanDecimalFromRows(rows)
		if err != nil {
			return decimal.Zero, err
		}
		sum = sum.Add(v)
	}
	if err := rows.Err(); err != nil {
		return decimal.Zero, err
	}
	return sum, nil
}
