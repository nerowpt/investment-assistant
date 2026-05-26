package doctor

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
)

// CheckLibrary 校验 L1 表一致性（03 §10C.11）。
func CheckLibrary(db *sql.DB) []string {
	var issues []string

	rows, err := db.Query(`
		SELECT li.id FROM library_items li
		WHERE li.status = 'active'
		AND (
			SELECT COUNT(*) FROM library_item_assets a
			WHERE a.library_item_id = li.id AND a.asset_role = 'primary'
		) != 1`)
	if err != nil {
		return []string{fmt.Sprintf("查询 primary asset: %v", err)}
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			issues = append(issues, err.Error())
			continue
		}
		n, _ := sqlstore.CountPrimaryAssets(db, id)
		issues = append(issues, fmt.Sprintf("library_item %s primary asset 数量=%d（须为 1）", id, n))
	}

	dupRows, err := db.Query(`
		SELECT dedup_key, COUNT(*) AS c FROM library_items
		WHERE status = 'active' GROUP BY dedup_key HAVING c > 1`)
	if err != nil {
		issues = append(issues, fmt.Sprintf("dedup_key 检查: %v", err))
	} else {
		defer dupRows.Close()
		for dupRows.Next() {
			var key string
			var c int
			if err := dupRows.Scan(&key, &c); err == nil {
				issues = append(issues, fmt.Sprintf("library_items dedup_key 重复: %s (%d)", key, c))
			}
		}
	}

	// checklist 不得引用 candidate（lc_ 前缀）
	refRows, err := db.Query(`
		SELECT id FROM checklist_submissions
		WHERE payload_json LIKE '%lc_%'`)
	if err == nil {
		defer refRows.Close()
		for refRows.Next() {
			var id string
			if err := refRows.Scan(&id); err == nil {
				issues = append(issues, fmt.Sprintf("checklist %s payload 疑似引用 candidate（lc_*）", id))
			}
		}
	}

	return issues
}

// FormatIssues 格式化 issue 列表。
func FormatIssues(issues []string) string {
	if len(issues) == 0 {
		return ""
	}
	return strings.Join(issues, "\n  - ")
}
