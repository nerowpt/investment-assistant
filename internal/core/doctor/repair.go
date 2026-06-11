package doctor

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
	"github.com/shopspring/decimal"
)

// RepairField 修复表单字段（供 H8 前端渲染）。
type RepairField struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Type    string `json:"type"` // checkbox | text | number | hidden
	Value   string `json:"value,omitempty"`
	Default bool   `json:"default,omitempty"`
	Tip     string `json:"tip,omitempty"`
}

// RepairAction 单条可修复项；与 doctor Issue 一一对应。
type RepairAction struct {
	ID         string        `json:"id"`
	Code       string        `json:"code"`
	Subject    string        `json:"subject"`
	Title      string        `json:"title"`
	Detail     string        `json:"detail"`
	Hint       string        `json:"hint"`
	ActionType string        `json:"action_type"`
	Fields     []RepairField `json:"fields"`
}

// RepairApply 用户提交的修复确认。
type RepairApply struct {
	ID      string            `json:"id"`
	Enabled bool              `json:"enabled"`
	Values  map[string]string `json:"values,omitempty"`
}

var (
	reLotIDInDetail     = regexp.MustCompile(`lot_ids 含 ([^，]+)`)
	reJournalIDInDetail = regexp.MustCompile(`journal_ids 含 ([^，]+)`)
	rePctMismatch       = regexp.MustCompile(`portfolio\.position_pct=([^，]+)，但 SQLite 中 code=([^ ]+) 的 open/partial lots 的 current_pct 之和=([^。]+)`)
	reLotCodeMismatch   = regexp.MustCompile(`lot ([^ ]+) 在 DB 中 code=([^，]+)`)
)

// BuildPortfolioRepairPlan 为 portfolio doctor 问题生成可页面操作的修复表单。
func BuildPortfolioRepairPlan(db *sql.DB, p *yamlstore.Portfolio) []RepairAction {
	issues := CheckPortfolio(db, p)
	actions := make([]RepairAction, 0, len(issues))
	seen := map[string]bool{}
	for _, iss := range issues {
		action, ok := repairActionForIssue(db, p, iss)
		if !ok || seen[action.ID] {
			continue
		}
		seen[action.ID] = true
		actions = append(actions, action)
	}
	return actions
}

func repairActionForIssue(db *sql.DB, p *yamlstore.Portfolio, iss Issue) (RepairAction, bool) {
	base := RepairAction{
		Code:    iss.Code,
		Subject: iss.Subject,
		Title:   iss.Title,
		Detail:  iss.Detail,
		Hint:    iss.Hint,
	}
	switch iss.Code {
	case "P001":
		lotID := extractSubmatch(reLotIDInDetail, iss.Detail, 1)
		if lotID == "" {
			return RepairAction{}, false
		}
		base.ID = fmt.Sprintf("P001|%s|%s", iss.Subject, lotID)
		base.ActionType = "remove_lot_id"
		base.Fields = []RepairField{
			{Key: "enabled", Label: fmt.Sprintf("从 %s 的 lot_ids 移除无效引用 %s", iss.Subject, lotID), Type: "checkbox", Default: true},
			{Key: "code", Value: iss.Subject, Type: "hidden"},
			{Key: "lot_id", Value: lotID, Type: "hidden"},
		}
		return base, true
	case "P002":
		jid := extractSubmatch(reJournalIDInDetail, iss.Detail, 1)
		if jid == "" {
			return RepairAction{}, false
		}
		base.ID = fmt.Sprintf("P002|%s|%s", iss.Subject, jid)
		base.ActionType = "remove_journal_id"
		base.Fields = []RepairField{
			{Key: "enabled", Label: fmt.Sprintf("从 %s 的 journal_ids 移除无效引用 %s", iss.Subject, jid), Type: "checkbox", Default: true},
			{Key: "code", Value: iss.Subject, Type: "hidden"},
			{Key: "journal_id", Value: jid, Type: "hidden"},
		}
		return base, true
	case "P003":
		lotID := extractSubmatch(reLotCodeMismatch, iss.Detail, 1)
		if lotID == "" {
			return RepairAction{}, false
		}
		base.ID = fmt.Sprintf("P003|%s|%s", iss.Subject, lotID)
		base.ActionType = "remove_lot_id"
		base.Fields = []RepairField{
			{Key: "enabled", Label: fmt.Sprintf("从 %s 的 lot_ids 移除错误 lot %s", iss.Subject, lotID), Type: "checkbox", Default: true},
			{Key: "code", Value: iss.Subject, Type: "hidden"},
			{Key: "lot_id", Value: lotID, Type: "hidden"},
		}
		return base, true
	case "P004":
		m := rePctMismatch.FindStringSubmatch(iss.Detail)
		if len(m) < 4 {
			return RepairAction{}, false
		}
		code := m[2]
		suggested := strings.TrimSpace(m[3])
		base.ID = fmt.Sprintf("P004|%s|%s", code, suggested)
		base.Subject = code
		base.ActionType = "set_position_pct"
		base.Fields = []RepairField{
			{Key: "enabled", Label: fmt.Sprintf("将 %s 的 position_pct 对齐为账本合计 %s%%", code, suggested), Type: "checkbox", Default: true},
			{Key: "code", Value: code, Type: "hidden"},
			{Key: "position_pct", Label: "仓位 %", Type: "number", Value: suggested, Tip: "以 SQLite open/partial lots 之和为准"},
		}
		// 若账本合计为 0 且仅有无效引用，额外提供删除整条 position
		if suggested == "0" && positionOnlyHasBrokenRefs(db, p, code) {
			base.Fields = append(base.Fields, RepairField{
				Key: "remove_position", Label: fmt.Sprintf("或：从 portfolio 删除 %s 整条记录（无真实持仓）", code),
				Type: "checkbox", Default: false, Tip: "适用于 example 模板残留；勾选后将忽略上方改仓位",
			})
		}
		return base, true
	case "P011":
		base.ID = "P011|portfolio.yaml|updated_at"
		base.ActionType = "set_updated_at"
		now := time.Now().Format(time.RFC3339)
		base.Fields = []RepairField{
			{Key: "enabled", Label: "补全 meta.updated_at 为当前时间", Type: "checkbox", Default: true},
			{Key: "updated_at", Label: "更新时间", Type: "text", Value: now},
		}
		return base, true
	case "P014":
		base.ID = fmt.Sprintf("P014|%s|remove", iss.Subject)
		base.ActionType = "remove_position"
		base.Fields = []RepairField{
			{Key: "enabled", Label: fmt.Sprintf("从 portfolio 移除 %s（观察池标的应在 watchlist.yaml）", iss.Subject), Type: "checkbox", Default: true},
			{Key: "code", Value: iss.Subject, Type: "hidden"},
		}
		return base, true
	case "P015":
		base.ID = fmt.Sprintf("P015|%s|pct", iss.Subject)
		base.ActionType = "set_position_pct"
		base.Fields = []RepairField{
			{Key: "enabled", Label: fmt.Sprintf("将 %s 的 position_pct 设为 0（closed 标的须为 0）", iss.Subject), Type: "checkbox", Default: true},
			{Key: "code", Value: iss.Subject, Type: "hidden"},
			{Key: "position_pct", Label: "仓位 %", Type: "number", Value: "0"},
		}
		return base, true
	case "P016":
		today := time.Now().Format("2006-01-02")
		base.ID = fmt.Sprintf("P016|%s|closed_at", iss.Subject)
		base.ActionType = "set_closed_at"
		base.Fields = []RepairField{
			{Key: "enabled", Label: fmt.Sprintf("为 %s 补全 closed_at 清仓日期", iss.Subject), Type: "checkbox", Default: true},
			{Key: "code", Value: iss.Subject, Type: "hidden"},
			{Key: "closed_at", Label: "清仓日期", Type: "text", Value: today, Tip: "格式 YYYY-MM-DD"},
		}
		return base, true
	default:
		return RepairAction{}, false
	}
}

// ApplyPortfolioRepairs 按用户确认应用修复；返回新 portfolio 副本。
func ApplyPortfolioRepairs(p *yamlstore.Portfolio, plan []RepairAction, applies []RepairApply) (*yamlstore.Portfolio, error) {
	planMap := make(map[string]RepairAction, len(plan))
	for _, a := range plan {
		planMap[a.ID] = a
	}
	out := clonePortfolioYAML(p)
	for _, apply := range applies {
		if !apply.Enabled {
			continue
		}
		action, ok := planMap[apply.ID]
		if !ok {
			return nil, fmt.Errorf("未知修复项: %s", apply.ID)
		}
		vals := fieldValues(action.Fields, apply.Values)
		if vals["enabled"] == "false" {
			continue
		}
		switch action.ActionType {
		case "remove_lot_id":
			if err := removeLotID(out, vals["code"], vals["lot_id"]); err != nil {
				return nil, err
			}
		case "remove_journal_id":
			if err := removeJournalID(out, vals["code"], vals["journal_id"]); err != nil {
				return nil, err
			}
		case "set_position_pct":
			if vals["remove_position"] == "true" {
				removePosition(out, vals["code"])
			} else if err := setPositionPct(out, vals["code"], vals["position_pct"]); err != nil {
				return nil, err
			}
		case "remove_position":
			removePosition(out, vals["code"])
		case "set_closed_at":
			if err := setClosedAt(out, vals["code"], vals["closed_at"]); err != nil {
				return nil, err
			}
		case "set_updated_at":
			out.Meta.UpdatedAt = vals["updated_at"]
		default:
			return nil, fmt.Errorf("不支持的修复类型: %s", action.ActionType)
		}
	}
	if out.Meta.UpdatedAt == "" {
		out.Meta.UpdatedAt = time.Now().Format(time.RFC3339)
	}
	return out, nil
}

func clonePortfolioYAML(p *yamlstore.Portfolio) *yamlstore.Portfolio {
	if p == nil {
		return &yamlstore.Portfolio{SchemaVersion: yamlstore.PortfolioSchemaVersion, Positions: []yamlstore.PortfolioPosition{}}
	}
	cp := *p
	cp.Positions = append([]yamlstore.PortfolioPosition(nil), p.Positions...)
	for i := range cp.Positions {
		cp.Positions[i].LotIDs = append([]string(nil), p.Positions[i].LotIDs...)
		cp.Positions[i].JournalIDs = append([]string(nil), p.Positions[i].JournalIDs...)
	}
	return &cp
}

func fieldValues(defaults []RepairField, overrides map[string]string) map[string]string {
	out := map[string]string{}
	for _, f := range defaults {
		if f.Type == "checkbox" && f.Key == "enabled" {
			if f.Default {
				out[f.Key] = "true"
			} else {
				out[f.Key] = "false"
			}
			continue
		}
		if f.Value != "" {
			out[f.Key] = f.Value
		}
	}
	for k, v := range overrides {
		out[k] = v
	}
	if _, ok := out["enabled"]; !ok {
		out["enabled"] = "true"
	}
	return out
}

func findPositionIndex(p *yamlstore.Portfolio, code string) (int, bool) {
	for i, pos := range p.Positions {
		if pos.Code == code {
			return i, true
		}
	}
	return -1, false
}

func removeLotID(p *yamlstore.Portfolio, code, lotID string) error {
	i, ok := findPositionIndex(p, code)
	if !ok {
		return fmt.Errorf("position %s 不存在", code)
	}
	next := make([]string, 0, len(p.Positions[i].LotIDs))
	for _, id := range p.Positions[i].LotIDs {
		if id != lotID {
			next = append(next, id)
		}
	}
	p.Positions[i].LotIDs = next
	return nil
}

func removeJournalID(p *yamlstore.Portfolio, code, journalID string) error {
	i, ok := findPositionIndex(p, code)
	if !ok {
		return fmt.Errorf("position %s 不存在", code)
	}
	next := make([]string, 0, len(p.Positions[i].JournalIDs))
	for _, id := range p.Positions[i].JournalIDs {
		if id != journalID {
			next = append(next, id)
		}
	}
	p.Positions[i].JournalIDs = next
	return nil
}

func setPositionPct(p *yamlstore.Portfolio, code, pctStr string) error {
	i, ok := findPositionIndex(p, code)
	if !ok {
		return fmt.Errorf("position %s 不存在", code)
	}
	pct, err := decimal.NewFromString(strings.TrimSpace(pctStr))
	if err != nil {
		return fmt.Errorf("position_pct 非法: %s", pctStr)
	}
	p.Positions[i].PositionPct = pct
	return nil
}

func setClosedAt(p *yamlstore.Portfolio, code, closedAt string) error {
	i, ok := findPositionIndex(p, code)
	if !ok {
		return fmt.Errorf("position %s 不存在", code)
	}
	p.Positions[i].ClosedAt = strings.TrimSpace(closedAt)
	return nil
}

func removePosition(p *yamlstore.Portfolio, code string) {
	next := make([]yamlstore.PortfolioPosition, 0, len(p.Positions))
	for _, pos := range p.Positions {
		if pos.Code != code {
			next = append(next, pos)
		}
	}
	p.Positions = next
}

func positionOnlyHasBrokenRefs(db *sql.DB, p *yamlstore.Portfolio, code string) bool {
	i, ok := findPositionIndex(p, code)
	if !ok {
		return false
	}
	pos := p.Positions[i]
	for _, id := range pos.LotIDs {
		var found string
		if db.QueryRow(`SELECT id FROM lots WHERE id = ?`, id).Scan(&found) == nil {
			return false
		}
	}
	for _, id := range pos.JournalIDs {
		var found string
		if db.QueryRow(`SELECT id FROM journals WHERE id = ?`, id).Scan(&found) == nil {
			return false
		}
	}
	return len(pos.LotIDs) > 0 || len(pos.JournalIDs) > 0
}

func extractSubmatch(re *regexp.Regexp, s string, idx int) string {
	m := re.FindStringSubmatch(s)
	if len(m) <= idx {
		return ""
	}
	return strings.TrimSpace(m[idx])
}

// SumOpenLotsPct 汇总标的 open/partial lot 仓位（供测试与 API 展示）。
func SumOpenLotsPct(db *sql.DB, code string) (decimal.Decimal, error) {
	return sqlstore.SumDecimalColumn(
		db,
		`SELECT CAST(current_pct AS TEXT) FROM lots WHERE code = ? AND status IN ('open', 'partial')`,
		code,
	)
}
