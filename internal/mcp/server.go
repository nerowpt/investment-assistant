// Package mcp 实现 Cursor MCP 只读 tool server（H7，04 §二十二）。
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/investment-assistant/investment-assistant/internal/core/account"
	"github.com/investment-assistant/investment-assistant/internal/core/query"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

const aiBoundary = "【边界】本工具仅返回客观数据与历史记录，不构成投资建议；不得代替用户填写结论性字段（分类、行动建议、置信度等）。"

// forbiddenWriteTools MVP-1 禁止注册的写 tool。
var forbiddenWriteTools = []string{
	"approve_checklist", "promote_library_item", "supplement_library", "write_tags", "update_portfolio",
}

// RunStdio 启动 MCP stdio server（Cursor 配置 inv mcp）。
func RunStdio(ac *account.Context) error {
	for _, name := range forbiddenWriteTools {
		_ = name // 文档约束；注册时不得加入
	}
	if err := ac.EnsureInitialized(); err != nil {
		return err
	}
	db, err := sqlstore.Open(ac.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := sqlstore.MigrateUp(db); err != nil {
		return err
	}
	reader := query.NewReader(ac, db)

	s := mcpserver.NewMCPServer("investment-assistant", "0.1.0-h7", mcpserver.WithToolCapabilities(true))
	registerTools(s, reader)
	return mcpserver.ServeStdio(s)
}

func registerTools(s *mcpserver.MCPServer, q *query.Reader) {
	addJSONTool(s, "get_portfolio", "读取 portfolio.yaml 当前持仓。"+aiBoundary,
		handlePortfolio(q))
	addJSONTool(s, "get_watchlist", "读取 watchlist.yaml 观察池。"+aiBoundary,
		handleWatchlist(q))
	addJSONTool(s, "search_library", "L1 研究素材检索（title/tags/code）。"+aiBoundary,
		handleSearchLibrary(q))
	addJSONTool(s, "get_library_item", "按 lib_id 返回素材元数据。"+aiBoundary,
		handleGetLibraryItem(q))
	addJSONTool(s, "search_journal", "按标的/动作检索决策 journal。"+aiBoundary,
		handleSearchJournal(q))
	addJSONTool(s, "get_journal", "单条 journal 详情含 snapshot 摘要。"+aiBoundary,
		handleGetJournal(q))
	addJSONTool(s, "get_checklist_template", "返回 Checklist 类型默认 payload 模板。"+aiBoundary,
		handleChecklistTemplate(q))
	addJSONTool(s, "get_risk_rules", "读取 risk_rules 与 personal_redlines。"+aiBoundary,
		handleRiskRules(q))
	addJSONTool(s, "check_position_against_rules", "模拟 M7 仓位/禁区检查（不下结论）。"+aiBoundary,
		handleCheckRules(q))
}

type toolHandler func(ctx context.Context, args map[string]any) (any, error)

func addJSONTool(s *mcpserver.MCPServer, name, desc string, fn toolHandler) {
	s.AddTool(mcp.NewTool(name, mcp.WithDescription(desc)), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cctx, cancel := context.WithTimeout(ctx, query.ToolTimeout)
		defer cancel()
		out, err := fn(cctx, req.GetArguments())
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := json.Marshal(out)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(string(raw)), nil
	})
}

func handlePortfolio(q *query.Reader) toolHandler {
	return func(ctx context.Context, args map[string]any) (any, error) {
		_ = ctx
		code, _ := args["code"].(string)
		includeClosed, _ := args["include_closed"].(bool)
		return q.GetPortfolio(code, includeClosed)
	}
}

func handleWatchlist(q *query.Reader) toolHandler {
	return func(ctx context.Context, args map[string]any) (any, error) {
		_ = ctx
		state, _ := args["state"].(string)
		code, _ := args["code"].(string)
		return q.GetWatchlist(state, code)
	}
}

func handleSearchLibrary(q *query.Reader) toolHandler {
	return func(ctx context.Context, args map[string]any) (any, error) {
		_ = ctx
		queryStr, _ := args["query"].(string)
		stock, _ := args["code"].(string)
		limit := intArg(args, "limit", 20)
		return q.SearchLibrary(queryStr, stock, limit)
	}
}

func handleGetLibraryItem(q *query.Reader) toolHandler {
	return func(ctx context.Context, args map[string]any) (any, error) {
		_ = ctx
		id, _ := args["lib_id"].(string)
		if id == "" {
			return nil, fmt.Errorf("lib_id 必填")
		}
		return q.GetLibraryItem(id)
	}
}

func handleSearchJournal(q *query.Reader) toolHandler {
	return func(ctx context.Context, args map[string]any) (any, error) {
		_ = ctx
		code, _ := args["code"].(string)
		action, _ := args["action_type"].(string)
		limit := intArg(args, "limit", 20)
		return q.SearchJournal(code, action, limit)
	}
}

func handleGetJournal(q *query.Reader) toolHandler {
	return func(ctx context.Context, args map[string]any) (any, error) {
		_ = ctx
		id, _ := args["journal_id"].(string)
		if id == "" {
			return nil, fmt.Errorf("journal_id 必填")
		}
		return q.GetJournal(id)
	}
}

func handleChecklistTemplate(q *query.Reader) toolHandler {
	return func(ctx context.Context, args map[string]any) (any, error) {
		_ = ctx
		typ, _ := args["checklist_type"].(string)
		if typ == "" {
			return nil, fmt.Errorf("checklist_type 必填")
		}
		return q.GetChecklistTemplate(typ)
	}
}

func handleRiskRules(q *query.Reader) toolHandler {
	return func(_ context.Context, _ map[string]any) (any, error) {
		return q.GetRiskRules()
	}
}

func handleCheckRules(q *query.Reader) toolHandler {
	return func(ctx context.Context, args map[string]any) (any, error) {
		scenario, _ := args["scenario"].(string)
		code, _ := args["code"].(string)
		if scenario == "" || code == "" {
			return nil, fmt.Errorf("scenario 与 code 必填")
		}
		pct, ok := args["planned_position_pct_after"].(float64)
		if !ok {
			return nil, fmt.Errorf("planned_position_pct_after 须为数字")
		}
		sector, _ := args["sector_id"].(string)
		thesis, _ := args["thesis_id"].(string)
		return q.CheckPositionAgainstRules(ctx, query.CheckPositionInput{
			Scenario: scenario, Code: code, PlannedPositionPctAfter: pct,
			SectorID: sector, ThesisID: thesis,
		})
	}
}

func intArg(args map[string]any, key string, def int) int {
	if v, ok := args[key].(float64); ok && v > 0 {
		return int(v)
	}
	return def
}

// ToolNames 返回 MVP-1 注册的只读 tool 名（测试用）。
func ToolNames() []string {
	return []string{
		"get_portfolio", "get_watchlist", "search_library", "get_library_item",
		"search_journal", "get_journal", "get_checklist_template", "get_risk_rules",
		"check_position_against_rules",
	}
}

// ForbiddenWriteTools 返回禁止注册的写 tool。
func ForbiddenWriteTools() []string {
	out := make([]string, len(forbiddenWriteTools))
	copy(out, forbiddenWriteTools)
	return out
}
