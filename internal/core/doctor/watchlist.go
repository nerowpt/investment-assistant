// Package doctor 实现 YAML ↔ SQLite 一致性检查（03 §10B.8）。
package doctor

import (
	"database/sql"
	"fmt"

	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
)

// CheckWatchlist 校验 watchlist.yaml 与 SQLite journals 一致，并与 portfolio 做 Q2A 交叉检查。
func CheckWatchlist(db *sql.DB, w *yamlstore.Watchlist, p *yamlstore.Portfolio) []string {
	var issues []string
	issues = append(issues, yamlstore.ValidateWatchlist(w)...)

	for _, item := range w.Items {
		if item.RemovedReason == "promoted_to_holding" || item.PromotedJournalID != "" {
			issues = append(issues, checkPromotedJournal(db, item)...)
		}
	}

	if p != nil {
		issues = append(issues, checkWatchlistPortfolioOverlap(w, p)...)
	}
	return issues
}

// checkPromotedJournal 校验升级建仓链路：promoted_journal_id 须在 journals 存在（03 §10B.8）。
func checkPromotedJournal(db *sql.DB, item yamlstore.WatchlistItem) []string {
	if item.PromotedJournalID == "" {
		if item.RemovedReason == "promoted_to_holding" {
			return []string{fmt.Sprintf("%s: promoted_to_holding 缺少 promoted_journal_id", item.ID)}
		}
		return nil
	}
	if item.RemovedReason != "" && item.RemovedReason != "promoted_to_holding" {
		return []string{fmt.Sprintf(
			"%s: 有 promoted_journal_id 但 removed_reason=%s（应为 promoted_to_holding）",
			item.ID, item.RemovedReason,
		)}
	}
	var found string
	err := db.QueryRow(`SELECT id FROM journals WHERE id = ?`, item.PromotedJournalID).Scan(&found)
	if err == sql.ErrNoRows {
		return []string{fmt.Sprintf("%s: promoted_journal_id 不存在: %s", item.ID, item.PromotedJournalID)}
	}
	if err != nil {
		return []string{fmt.Sprintf("%s: 查询 journal %s: %v", item.ID, item.PromotedJournalID, err)}
	}
	return nil
}

// checkWatchlistPortfolioOverlap 同一 code 不能同时在 watchlist(watching) 与 portfolio(holding)（Q2A 强化）。
func checkWatchlistPortfolioOverlap(w *yamlstore.Watchlist, p *yamlstore.Portfolio) []string {
	holding := map[string]struct{}{}
	for _, pos := range p.Positions {
		if pos.State == "holding" && pos.Code != "" {
			holding[pos.Code] = struct{}{}
		}
	}
	var issues []string
	for _, item := range w.Items {
		if item.State != "watching" || item.Code == "" {
			continue
		}
		if _, ok := holding[item.Code]; ok {
			issues = append(issues, fmt.Sprintf(
				"%s: code %s 同时在 watchlist(watching) 与 portfolio(holding)；升级建仓后 watchlist 应 removed",
				item.ID, item.Code,
			))
		}
	}
	return issues
}
