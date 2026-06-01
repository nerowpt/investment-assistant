// Package doctor 实现 YAML ↔ SQLite 一致性检查（03 §10B.8）。
package doctor

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
)

const portfolioHintDoc = "详见 docs/manual/H1-portfolio与doctor.md §故障排查"

// CheckPortfolio 校验 portfolio.yaml 与 SQLite lots/journals 一致。
func CheckPortfolio(db *sql.DB, p *yamlstore.Portfolio) []Issue {
	var issues []Issue
	issues = append(issues, wrapPortfolioYAMLIssues(yamlstore.ValidatePortfolio(p))...)

	for _, pos := range p.Positions {
		if pos.State != "holding" && pos.State != "closed" {
			continue
		}
		issues = append(issues, checkLotIDs(db, pos)...)
		issues = append(issues, checkJournalIDs(db, pos.Code, pos.JournalIDs)...)
		if pos.State == "holding" {
			issues = append(issues, checkOpenLotsPct(db, pos)...)
		}
	}
	return issues
}

func wrapPortfolioYAMLIssues(raw []string) []Issue {
	var out []Issue
	for _, msg := range raw {
		out = append(out, yamlPortfolioIssue(msg))
	}
	return out
}

func yamlPortfolioIssue(msg string) Issue {
	// 保持与 yamlstore.ValidatePortfolio 消息同步，按前缀映射规则码与处理建议。
	switch {
	case strings.HasPrefix(msg, "schema_version"):
		return Issue{
			Code: "P010", Subject: "portfolio.yaml", Title: "schema 版本不匹配",
			Detail: msg,
			Hint:   "勿手改 schema_version；从 config/portfolio.yaml.example 复制或走 checklist approve 写入。",
		}
	case strings.HasPrefix(msg, "meta.updated_at"):
		return Issue{
			Code: "P011", Subject: "portfolio.yaml", Title: "缺少 meta.updated_at",
			Detail: msg,
			Hint:   "补 ISO8601 时间戳，或通过 inv checklist approve 成功同步后自动更新。",
		}
	case strings.HasPrefix(msg, "positions["):
		return Issue{
			Code: "P012", Subject: "portfolio.yaml", Title: "position 字段缺失",
			Detail: msg, Hint: "检查 ref-portfolio-yaml-fields.md，补全 code 等必填字段。",
		}
	case strings.HasPrefix(msg, "重复 code"):
		return Issue{
			Code: "P013", Subject: "portfolio.yaml", Title: "标的 code 重复",
			Detail: msg, Hint: "同一 code 只能有一条 position；合并或删除重复项。",
		}
	case strings.Contains(msg, "state=watching"):
		return Issue{
			Code: "P014", Subject: extractCodeFromMsg(msg), Title: "portfolio 不应含 watching",
			Detail: msg,
			Hint:   "观察池标的应在 watchlist.yaml；portfolio 仅 holding/closed。",
		}
	case strings.Contains(msg, "closed 标的 position_pct"):
		return Issue{
			Code: "P015", Subject: extractCodeFromMsg(msg), Title: "closed 仓位须为 0",
			Detail: msg, Hint: "将 position_pct 改为 0，或若仍持仓则改 state=holding。",
		}
	case strings.Contains(msg, "closed 标的缺少 closed_at"):
		return Issue{
			Code: "P016", Subject: extractCodeFromMsg(msg), Title: "closed 缺少 closed_at",
			Detail: msg, Hint: "补 YYYY-MM-DD 清仓日期。",
		}
	default:
		return Issue{
			Code: "P000", Subject: "portfolio.yaml", Title: "YAML 约束不满足",
			Detail: msg, Hint: portfolioHintDoc,
		}
	}
}

func checkJournalIDs(db *sql.DB, code string, ids []string) []Issue {
	var issues []Issue
	for _, id := range ids {
		var found string
		err := db.QueryRow(`SELECT id FROM journals WHERE id = ?`, id).Scan(&found)
		if err == sql.ErrNoRows {
			issues = append(issues, Issue{
				Code: "P002", Subject: code, Title: "journal 引用断裂",
				Detail: fmt.Sprintf("portfolio.yaml 的 journal_ids 含 %s，但 SQLite journals 表无此记录。", id),
				Hint: fmt.Sprintf(
					"编辑 portfolio.yaml：从 %s 的 journal_ids 删除 %s；"+
						"若该笔交易真实存在则检查是否误删库；若为 example 模板残留可直接删整条 position 或清空 ids。%s",
					code, id, portfolioHintDoc),
			})
		} else if err != nil {
			issues = append(issues, Issue{
				Code: "P002", Subject: code, Title: "journal 查询失败",
				Detail: fmt.Sprintf("%s: %v", id, err),
				Hint:   "检查 assistant.sqlite 是否损坏或被锁定。",
			})
		}
	}
	return issues
}

func checkLotIDs(db *sql.DB, pos yamlstore.PortfolioPosition) []Issue {
	var issues []Issue
	for _, id := range pos.LotIDs {
		var code string
		err := db.QueryRow(`SELECT code FROM lots WHERE id = ?`, id).Scan(&code)
		if err == sql.ErrNoRows {
			issues = append(issues, Issue{
				Code: "P001", Subject: pos.Code, Title: "lot 引用断裂",
				Detail: fmt.Sprintf("portfolio.yaml 的 lot_ids 含 %s，但 SQLite lots 表无此记录。", id),
				Hint: fmt.Sprintf(
					"编辑 portfolio.yaml：从 %s 的 lot_ids 删除 %s；"+
						"常见为 config 模板虚构 id 或 approve 异常遗留，勿与 journals/lots 真实 id 混用。%s",
					pos.Code, id, portfolioHintDoc),
			})
			continue
		}
		if err != nil {
			issues = append(issues, Issue{
				Code: "P001", Subject: pos.Code, Title: "lot 查询失败",
				Detail: fmt.Sprintf("%s: %v", id, err),
				Hint:   "检查 assistant.sqlite 是否损坏或被锁定。",
			})
			continue
		}
		if code != pos.Code {
			issues = append(issues, Issue{
				Code: "P003", Subject: pos.Code, Title: "lot 标的 code 不一致",
				Detail: fmt.Sprintf("lot %s 在 DB 中 code=%s，与 portfolio position code=%s 不符。", id, code, pos.Code),
				Hint:   "从 lot_ids 移除错误 lot，或修正 portfolio 中 code 字段。",
			})
		}
	}
	return issues
}

func checkOpenLotsPct(db *sql.DB, pos yamlstore.PortfolioPosition) []Issue {
	sum, err := sqlstore.SumDecimalColumn(
		db,
		`SELECT CAST(current_pct AS TEXT) FROM lots WHERE code = ? AND status IN ('open', 'partial')`,
		pos.Code,
	)
	if err != nil {
		return []Issue{{
			Code: "P004", Subject: pos.Code, Title: "无法汇总 open lots",
			Detail: err.Error(),
			Hint:   "检查 lots 表是否存在、code 是否正确。",
		}}
	}
	if !sum.Equal(pos.PositionPct) {
		return []Issue{{
			Code: "P004", Subject: pos.Code, Title: "仓位比例不一致",
			Detail: fmt.Sprintf(
				"portfolio.position_pct=%s，但 SQLite 中 code=%s 的 open/partial lots 的 current_pct 之和=%s。",
				pos.PositionPct.String(), pos.Code, sum.String()),
			Hint: "以 lots 表为准修正 portfolio.position_pct 与 lot_ids；" +
				"若 DB 有多余 lot（误 approve 等），删除孤儿 lot/journal 或走 checklist sell/add 对齐。" +
				portfolioHintDoc,
		}}
	}
	return nil
}

func extractCodeFromMsg(msg string) string {
	if i := strings.Index(msg, ":"); i > 0 {
		return strings.TrimSpace(msg[:i])
	}
	return "—"
}
