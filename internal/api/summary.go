package api

import (
	"encoding/json"
	"fmt"
	"strings"
)

// summarizeChecklist 从 payload 提取列表展示用一行摘要。
func summarizeChecklist(typ, payloadJSON string) string {
	if payloadJSON == "" {
		return ""
	}
	var p map[string]any
	if json.Unmarshal([]byte(payloadJSON), &p) != nil {
		return ""
	}
	switch typ {
	case "sell":
		return fmt.Sprintf("卖出 %s 股 @ ¥%s · %s",
			fmtVal(p["sell_shares"]), fmtVal(p["execution_price"]), sellReasonText(p))
	case "buy":
		return fmt.Sprintf("买入 %s 股 @ ¥%s · %s",
			fmtVal(p["shares"]), fmtVal(p["execution_price"]), clipStr(strVal(p["buy_reason_summary"]), 40))
	case "add":
		return fmt.Sprintf("加仓 %s 股 @ ¥%s · %s",
			fmtVal(p["shares"]), fmtVal(p["execution_price"]), clipStr(strVal(p["add_reason_summary"]), 40))
	case "inspect":
		return fmt.Sprintf("巡检 · %s · 计划 %s",
			classificationText(p["classification"]), fmtVal(p["planned_action"]))
	case "watch":
		return clipStr(strVal(p["watch_reason"]), 60)
	default:
		return clipStr(strVal(p["buy_reason_summary"]), 60)
	}
}

func fmtVal(v any) string {
	if v == nil {
		return "—"
	}
	switch x := v.(type) {
	case string:
		if x == "" {
			return "—"
		}
		return x
	case float64:
		if x == float64(int64(x)) {
			return fmt.Sprintf("%d", int64(x))
		}
		return fmt.Sprintf("%.2f", x)
	case int:
		return fmt.Sprintf("%d", x)
	default:
		return fmt.Sprint(v)
	}
}

func strVal(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func clipStr(s string, n int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "—"
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func sellReasonText(p map[string]any) string {
	detail := strVal(p["sell_reason_detail"])
	if detail != "" {
		return clipStr(detail, 40)
	}
	reason := strVal(p["sell_reason"])
	labels := map[string]string{
		"target_achieved": "目标达成",
		"stop_loss":       "止损",
		"thesis_broken":   "逻辑破坏",
		"rebalance":       "再平衡",
		"other":           "其他",
	}
	if lbl, ok := labels[reason]; ok {
		return lbl
	}
	if reason != "" {
		return reason
	}
	return "—"
}

func classificationText(v any) string {
	labels := map[string]string{
		"thesis_intact":          "逻辑成立",
		"thesis_weakened":        "逻辑减弱",
		"thesis_broken":          "逻辑破坏",
		"wait_for_style_switch":  "等待风格切换",
		"opportunity_cost_alert": "机会成本预警",
	}
	s := strVal(v)
	if lbl, ok := labels[s]; ok {
		return lbl
	}
	if s == "" {
		return "—"
	}
	return s
}
