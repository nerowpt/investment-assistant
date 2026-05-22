// Package doctor 实现 YAML ↔ SQLite 一致性检查（03 §10B.8）。
package doctor

import (
	"database/sql"
	"fmt"

	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
)

// CheckPortfolio 校验 portfolio.yaml 与 SQLite lots/journals 一致。
func CheckPortfolio(db *sql.DB, p *yamlstore.Portfolio) []string {
	var issues []string
	issues = append(issues, yamlstore.ValidatePortfolio(p)...)

	for _, pos := range p.Positions {
		if pos.State != "holding" && pos.State != "closed" {
			continue
		}
		issues = append(issues, checkLotIDs(db, pos)...)
		issues = append(issues, checkJournalIDs(db, pos.JournalIDs)...)
		if pos.State == "holding" {
			issues = append(issues, checkOpenLotsPct(db, pos)...)
		}
	}
	return issues
}

// checkJournalIDs 校验 portfolio 引用的 journal_ids 在 SQLite 中均存在（03 §10B.8）。
func checkJournalIDs(db *sql.DB, ids []string) []string {
	var issues []string
	for _, id := range ids {
		var found string
		err := db.QueryRow(`SELECT id FROM journals WHERE id = ?`, id).Scan(&found)
		if err == sql.ErrNoRows {
			issues = append(issues, fmt.Sprintf("journal_ids 引用不存在: %s", id))
		} else if err != nil {
			issues = append(issues, fmt.Sprintf("查询 journal %s: %v", id, err))
		}
	}
	return issues
}

// checkLotIDs 校验 lot_ids 存在且 lots.code 与 position.code 一致（03 §10B.8）。
func checkLotIDs(db *sql.DB, pos yamlstore.PortfolioPosition) []string {
	var issues []string
	for _, id := range pos.LotIDs {
		var code string
		err := db.QueryRow(`SELECT code FROM lots WHERE id = ?`, id).Scan(&code)
		if err == sql.ErrNoRows {
			issues = append(issues, fmt.Sprintf("%s: lot_ids 引用不存在 %s", pos.Code, id))
			continue
		}
		if err != nil {
			issues = append(issues, fmt.Sprintf("查询 lot %s: %v", id, err))
			continue
		}
		if code != pos.Code {
			issues = append(issues, fmt.Sprintf("%s: lot %s code 不匹配（DB=%s）", pos.Code, id, code))
		}
	}
	return issues
}

// checkOpenLotsPct 校验 holding 标的：open/partial lot 的 current_pct 之和等于 position_pct（03 §10B.8）。
// 关键：SQLite REAL 列必须经 CAST AS TEXT + decimal.NewFromString 通道读取，
// 禁止 float64 中转（T24 / docs/06 §D11）。
func checkOpenLotsPct(db *sql.DB, pos yamlstore.PortfolioPosition) []string {
	sum, err := sqlstore.SumDecimalColumn(
		db,
		`SELECT CAST(current_pct AS TEXT) FROM lots WHERE code = ? AND status IN ('open', 'partial')`,
		pos.Code,
	)
	if err != nil {
		return []string{fmt.Sprintf("%s: 查询 open lots: %v", pos.Code, err)}
	}
	if !sum.Equal(pos.PositionPct) {
		return []string{fmt.Sprintf(
			"%s: sum(open lots.current_pct)=%s != position_pct=%s",
			pos.Code, sum.String(), pos.PositionPct.String(),
		)}
	}
	return nil
}
