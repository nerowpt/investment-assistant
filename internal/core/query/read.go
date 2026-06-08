// Package query 提供 MCP/CLI 只读查询（H7）。
package query

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/investment-assistant/investment-assistant/internal/core/account"
	chksvc "github.com/investment-assistant/investment-assistant/internal/core/checklist"
	"github.com/investment-assistant/investment-assistant/internal/core/risk"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
)

// Reader 只读查询入口。
type Reader struct {
	ac *account.Context
	db *sql.DB
}

// NewReader 构造 Reader（须已 migrate）。
func NewReader(ac *account.Context, db *sql.DB) *Reader {
	return &Reader{ac: ac, db: db}
}

// GetPortfolio 读取 portfolio.yaml，可选过滤单标的。
func (r *Reader) GetPortfolio(code string, includeClosed bool) (any, error) {
	p, err := yamlstore.LoadPortfolio(r.ac.PortfolioPath())
	if err != nil {
		return nil, err
	}
	if code == "" && includeClosed {
		return p, nil
	}
	var filtered []yamlstore.PortfolioPosition
	for _, pos := range p.Positions {
		if code != "" && pos.Code != code {
			continue
		}
		if !includeClosed && pos.State == "closed" {
			continue
		}
		filtered = append(filtered, pos)
	}
	out := map[string]any{
		"schema_version": p.SchemaVersion,
		"meta":           p.Meta,
		"positions":      filtered,
	}
	return out, nil
}

// GetWatchlist 读取 watchlist.yaml。
func (r *Reader) GetWatchlist(state, code string) (any, error) {
	wl, err := yamlstore.LoadWatchlist(r.ac.WatchlistPath())
	if err != nil {
		return nil, err
	}
	if state == "" {
		state = "watching"
	}
	var items []yamlstore.WatchlistItem
	for _, it := range wl.Items {
		if code != "" && it.Code != code {
			continue
		}
		switch state {
		case "all":
			items = append(items, it)
		case "removed":
			if it.RemovedReason != "" {
				items = append(items, it)
			}
		default:
			if it.RemovedReason == "" {
				items = append(items, it)
			}
		}
	}
	return map[string]any{"items": items}, nil
}

// SearchLibrary L1 素材检索。
func (r *Reader) SearchLibrary(query, stock string, limit int) (any, error) {
	var rows []sqlstore.LibraryItemRow
	var err error
	if query == "" && stock == "" {
		rows, err = sqlstore.ListLibraryItems(r.db, "active", "", "")
	} else if query == "" {
		rows, err = sqlstore.ListLibraryItems(r.db, "active", stock, "")
	} else {
		rows, err = sqlstore.SearchLibraryItems(r.db, query, stock)
	}
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	}
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return map[string]any{"items": rows}, nil
}

// GetLibraryItem 按 lib_id 返回素材。
func (r *Reader) GetLibraryItem(id string) (any, error) {
	item, err := sqlstore.GetLibraryItem(r.db, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return map[string]any{"error": map[string]string{"code": "not_found", "message": "library_item 不存在"}}, nil
	}
	return item, nil
}

// SearchJournal 检索决策 journal。
func (r *Reader) SearchJournal(code, actionType string, limit int) (any, error) {
	rows, err := sqlstore.SearchJournals(r.db, code, actionType, limit)
	if err != nil {
		return nil, err
	}
	return map[string]any{"journals": rows}, nil
}

// GetJournal 单条 journal + snapshot 摘要。
func (r *Reader) GetJournal(id string) (any, error) {
	row, err := sqlstore.GetJournal(r.db, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return map[string]any{"error": map[string]string{"code": "not_found", "message": "journal 不存在"}}, nil
	}
	payload, _ := sqlstore.GetJournalPayload(r.db, id)
	snap, _ := sqlstore.GetDataSnapshotSummary(r.db, row.DataSnapshotID)
	var snapSummary any
	if snap != "" {
		_ = json.Unmarshal([]byte(snap), &snapSummary)
	}
	var lotIDs []string
	if row.LotID != "" {
		lotIDs = []string{row.LotID}
	}
	return map[string]any{
		"journal":          row,
		"payload_json":     jsonRaw(payload),
		"snapshot_summary": snapSummary,
		"lot_ids":          lotIDs,
	}, nil
}

func jsonRaw(s string) any {
	if s == "" {
		return nil
	}
	var v any
	if json.Unmarshal([]byte(s), &v) == nil {
		return v
	}
	return s
}

// GetChecklistTemplate 返回 checklist 默认 payload 模板。
func (r *Reader) GetChecklistTemplate(checklistType string) (any, error) {
	if err := chksvc.ValidateType(checklistType); err != nil {
		return nil, err
	}
	raw := chksvc.DefaultPayloadTemplate(checklistType)
	var schema any
	_ = json.Unmarshal([]byte(raw), &schema)
	return map[string]any{
		"checklist_type":         checklistType,
		"payload_schema_version": chksvc.PayloadSchemaVersion,
		"template":               schema,
		"field_hints_zh":         "完整字段见 docs/manual/ref-checklist-types.md 与 02 §十六",
	}, nil
}

// GetRiskRules 读取 M7 配置。
func (r *Reader) GetRiskRules() (any, error) {
	rules, err := yamlstore.LoadRiskRules(r.ac.RiskRulesPath())
	if err != nil {
		return nil, err
	}
	redlines, err := yamlstore.LoadPersonalRedlines(r.ac.PersonalRedlinesPath())
	if err != nil {
		return nil, err
	}
	enabled := make([]yamlstore.Redline, 0, len(redlines.Redlines))
	for _, rl := range redlines.Redlines {
		if rl.Enabled {
			enabled = append(enabled, rl)
		}
	}
	return map[string]any{
		"position_limits":   rules.PositionLimits,
		"legacy_over_limit": rules.LegacyOverLimit,
		"redlines":          enabled,
	}, nil
}

// CheckPositionInput M7 模拟输入。
type CheckPositionInput struct {
	Scenario                string
	Code                    string
	PlannedPositionPctAfter float64
	SectorID                string
	ThesisID                string
}

// CheckPositionAgainstRules 模拟 M7（不下结论性建议）。
func (r *Reader) CheckPositionAgainstRules(ctx context.Context, in CheckPositionInput) (any, error) {
	_ = ctx
	portfolio, _ := yamlstore.LoadPortfolio(r.ac.PortfolioPath())
	rules, err := yamlstore.LoadRiskRules(r.ac.RiskRulesPath())
	if err != nil {
		return nil, err
	}
	redlines, err := yamlstore.LoadPersonalRedlines(r.ac.PersonalRedlinesPath())
	if err != nil {
		return nil, err
	}
	sectorID, thesisID := in.SectorID, in.ThesisID
	if sectorID == "" || thesisID == "" {
		s, t := risk.SectorThesisFromPortfolio(portfolio, in.Code)
		if sectorID == "" {
			sectorID = s
		}
		if thesisID == "" {
			thesisID = t
		}
	}
	proposed := in.PlannedPositionPctAfter
	guard, err := risk.Evaluate(risk.EvaluateInput{
		Scenario:    in.Scenario,
		Code:        in.Code,
		SectorID:    sectorID,
		ThesisID:    thesisID,
		ProposedPct: proposed,
		Portfolio:   portfolio,
		RiskRules:   rules,
		Redlines:    redlines,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"warnings":        guard.Warnings,
		"hard_blocks":     guard.HardBlocks,
		"approve_blocked": guard.ApproveBlocked,
		"disclaimer":      "模拟结果仅供参考；是否继续决策由用户自行判断",
	}, nil
}

// ToolTimeout 单 tool 超时。
const ToolTimeout = 10 * time.Second

// ErrJSON 构造 MCP 错误 JSON。
func ErrJSON(code, message string) string {
	b, _ := json.Marshal(map[string]any{"error": map[string]string{"code": code, "message": message}})
	return string(b)
}
