package checklist

import (
	"context"
	"encoding/json"
)

// BuyPayload 建仓 checklist payload 子集（02 §16.2）。
type BuyPayload struct {
	SourceEntry              string         `json:"source_entry"`               // 入口来源：manual / watchlist / library 等
	PositionType             string         `json:"position_type"`              // 仓位类型：core 主仓 | swing 波段
	BuyReasonSummary         string         `json:"buy_reason_summary"`         // 建仓理由摘要（journal.summary 来源）
	InvestmentThesis         string         `json:"investment_thesis"`          // 持有逻辑全文，写入 portfolio.investment_thesis
	TargetPrice              any            `json:"target_price"`               // 目标价：数值或 {lower, upper} 区间
	StopLoss                 any            `json:"stop_loss"`                  // 止损/减仓线（元/股）
	ReversalConditions       []string       `json:"reversal_conditions"`        // 逻辑反转条件，至少 1 项
	PositionSizePlan         map[string]any `json:"position_size_plan"`         // 仓位计划；initial_pct 用于 M7 与 lot.initial_pct
	OpportunityCostBenchmark string         `json:"opportunity_cost_benchmark"` // 机会成本基准：HS300 / CSI_TECH 等
	Confidence               string         `json:"confidence"`                 // 置信度：low | medium | high
	RelatedLibraryIDs        []string       `json:"related_library_ids"`        // 支撑依据的 L1 素材 id（lib_*）
	WatchlistOriginID        string         `json:"watchlist_origin_id"`        // 若从观察池升级，原 w_* id
}

// AddPayload 加仓 checklist payload 子集（02 §16.4）。
type AddPayload struct {
	LinkedBuyJournalID string         `json:"linked_buy_journal_id"` // 关联首次建仓 journal id（j_*）
	AddPct             float64        `json:"add_pct"`               // 本次加仓占总资产 %，用于 M7 与 lot.initial_pct
	PositionType       string         `json:"position_type"`         // 仓位类型；空则默认 core
	InvestmentThesis   string         `json:"investment_thesis"`     // 更新后的持有逻辑；非空时 portfolio.thesis_version +1
	RelatedLibraryIDs  []string       `json:"related_library_ids"`   // 本次加仓引用的 L1 素材 id
	PositionSizePlan   map[string]any `json:"position_size_plan"`    // 加仓后仓位计划（max_pct 等，可选）
}

// WatchPayload 观察 checklist payload 子集（02 §16.1）。
type WatchPayload struct {
	WatchReason       string   `json:"watch_reason"`         // 纳入观察池的原因
	Hypothesis        string   `json:"hypothesis"`           // 待验证假设
	KeyMetricsToWatch []string `json:"key_metrics_to_watch"` // 重点跟踪指标列表
	ExpectedTrigger   string   `json:"expected_trigger"`     // 预期触发买入/加深研究的条件
	InvalidCondition  string   `json:"invalid_condition"`    // 假设失效条件
	ReviewDate        string   `json:"review_date"`          // 下次回顾日期 YYYY-MM-DD
	Priority          string   `json:"priority"`             // 优先级：low | medium | high
	RelatedLibraryIDs []string `json:"related_library_ids"`  // 关联 L1 素材 id
	SourceEntry       string   `json:"source_entry"`         // 入口来源；空则 approve 时写 manual
}

// ParseBuyPayload 从 payload_json 解析 buy 字段。
func ParseBuyPayload(payloadJSON string) (*BuyPayload, error) {
	var p BuyPayload
	if err := json.Unmarshal([]byte(payloadJSON), &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// ParseAddPayload 从 payload_json 解析 add 字段。
func ParseAddPayload(payloadJSON string) (*AddPayload, error) {
	var p AddPayload
	if err := json.Unmarshal([]byte(payloadJSON), &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// ParseWatchPayload 从 payload_json 解析 watch 字段。
func ParseWatchPayload(payloadJSON string) (*WatchPayload, error) {
	var p WatchPayload
	if err := json.Unmarshal([]byte(payloadJSON), &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// MarketFetcher 拉取客观行情事实（H3 worker；approve 时冻结 snapshot）。
type MarketFetcher interface {
	FetchQuote(ctx context.Context, code string) (price float64, changePct float64, source, tier string, err error)
	FetchValuation(ctx context.Context, code string) (peTTM, pb float64, source, tier string, err error)
}
