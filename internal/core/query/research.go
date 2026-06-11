package query

import (
	"context"
	"fmt"
	"strings"
	"time"

	dataworkerv1 "github.com/investment-assistant/investment-assistant/gen/go/dataworker/v1"
	"github.com/investment-assistant/investment-assistant/internal/worker"
)

// ResearchPackDef 可点击拉取的研究数据包定义。
type ResearchPackDef struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// ResearchDossier 研究档案页聚合视图。
type ResearchDossier struct {
	Code          string                  `json:"code"`
	Name          string                  `json:"name"`
	Zones         []string                `json:"zones"`
	LibraryItems  []BuyContextLibraryItem `json:"library_items"`
	Packs         []ResearchPackDef       `json:"packs"`
	WorkerOK      bool                    `json:"worker_ok"`
	WorkerMessage string                  `json:"worker_message,omitempty"`
}

// ResearchPackResult 单次拉取结果（事实数据，不含买卖结论）。
type ResearchPackResult struct {
	PackID      string         `json:"pack_id"`
	Code        string         `json:"code"`
	Title       string         `json:"title"`
	Summary     string         `json:"summary"`
	Body        string         `json:"body"`
	Source      string         `json:"source"`
	Tier        string         `json:"tier"`
	CapturedAt  string         `json:"captured_at"`
	SuggestTier string         `json:"suggest_tier"`
	Raw         map[string]any `json:"raw,omitempty"`
}

var defaultResearchPacks = []ResearchPackDef{
	{ID: "quote", Title: "实时行情", Description: "现价、涨跌幅、交易日（data-worker）"},
	{ID: "company_valuation", Title: "公司估值", Description: "个股 PE/PB/PS 等指标；历史分位 MVP-1 暂为 0"},
	{ID: "sector_valuation", Title: "板块估值", Description: "所属行业板块 PE/PB 快照（事实数据，不下结论）"},
	{ID: "volume", Title: "成交量", Description: "近日成交量 vs 60 日均量，标注缩量/放量高峰"},
	{ID: "market_benchmark", Title: "市场基准", Description: "沪深300 当日表现，用于机会成本对照"},
	{ID: "news", Title: "重大新闻", Description: "近期新闻标题列表（用户确认后可入库 L1）"},
}

// BuildResearchDossier 构建研究档案；zones 来自当前 pool 聚合。
func (r *Reader) BuildResearchDossier(ctx context.Context, wc *worker.Client, code string) (*ResearchDossier, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, errPoolCodeRequired
	}
	ctxInfo, err := r.BuildBuyContext(code, "research", "")
	if err != nil {
		return nil, err
	}
	pool, err := r.BuildPool("")
	if err != nil {
		return nil, err
	}
	zones := make([]string, 0, 4)
	for _, z := range pool.Zones {
		for _, it := range z.Items {
			if it.Code == code {
				zones = append(zones, z.ID)
			}
		}
	}

	name := ctxInfo.Name
	if name == "" || name == code {
		name = r.ResolveStockName(ctx, wc, code)
	}

	out := &ResearchDossier{
		Code:         code,
		Name:         name,
		Zones:        zones,
		LibraryItems: ctxInfo.LibraryItems,
		Packs:        append([]ResearchPackDef(nil), defaultResearchPacks...),
	}
	if wc != nil {
		if _, err := wc.HealthCheck(ctx); err != nil {
			out.WorkerOK = false
			out.WorkerMessage = "data-worker 不可用，请在本机启动 worker 后再拉取"
		} else {
			out.WorkerOK = true
		}
	} else {
		out.WorkerOK = false
		out.WorkerMessage = "未配置 data-worker 客户端"
	}
	return out, nil
}

// FetchResearchPack 按 pack id 拉取外部事实数据。
func FetchResearchPack(ctx context.Context, wc *worker.Client, code, packID string) (*ResearchPackResult, error) {
	code = strings.TrimSpace(code)
	packID = strings.TrimSpace(packID)
	if code == "" {
		return nil, errPoolCodeRequired
	}
	if wc == nil {
		return nil, fmt.Errorf("data-worker 未配置")
	}
	switch packID {
	case "quote":
		return fetchQuotePack(ctx, wc, code)
	case "company_valuation":
		return fetchValuationPack(ctx, wc, code)
	case "market_benchmark":
		return fetchMarketPack(ctx, wc, code)
	case "news":
		return fetchNewsPack(ctx, wc, code)
	case "sector_valuation", "volume":
		return fetchExtendedPack(ctx, wc, code, packID)
	default:
		return nil, fmt.Errorf("未知数据包: %s", packID)
	}
}

func fetchExtendedPack(ctx context.Context, wc *worker.Client, code, packID string) (*ResearchPackResult, error) {
	res, err := wc.FetchResearchExtended(ctx, code, packID)
	if err != nil {
		return nil, err
	}
	suggestTier := "B"
	if packID == packIDVolume {
		suggestTier = "A"
	}
	return &ResearchPackResult{
		PackID: packID, Code: code,
		Title: res.Title, Summary: res.Summary, Body: res.Body,
		Source: res.Source, Tier: res.Tier, CapturedAt: res.CapturedAt, SuggestTier: suggestTier,
	}, nil
}

func fetchQuotePack(ctx context.Context, wc *worker.Client, code string) (*ResearchPackResult, error) {
	res, err := wc.FetchQuote(ctx, code)
	if err != nil {
		return nil, err
	}
	src, tier, captured := provenanceFields(res.GetProvenance())
	body := fmt.Sprintf(
		"标的：%s %s\n交易日：%s\n现价：%.2f\n涨跌幅：%.2f%%\n涨跌额：%.2f\n来源：%s tier=%s\n采集：%s",
		res.GetCode(), res.GetName(), res.GetTradeDate(),
		res.GetPrice(), res.GetChangePct(), res.GetChangeAmount(),
		src, tier, captured,
	)
	return &ResearchPackResult{
		PackID: packIDQuote, Code: code,
		Title: fmt.Sprintf("%s %s 行情 %s", res.GetName(), code, res.GetTradeDate()),
		Summary: fmt.Sprintf("现价 %.2f，涨跌 %.2f%%", res.GetPrice(), res.GetChangePct()),
		Body: body, Source: src, Tier: tier, CapturedAt: captured, SuggestTier: "A",
		Raw: map[string]any{
			"price": res.GetPrice(), "change_pct": res.GetChangePct(),
			"name": res.GetName(), "trade_date": res.GetTradeDate(),
		},
	}, nil
}

func fetchValuationPack(ctx context.Context, wc *worker.Client, code string) (*ResearchPackResult, error) {
	res, err := wc.FetchValuation(ctx, code)
	if err != nil {
		return nil, err
	}
	src, tier, captured := provenanceFields(res.GetProvenance())
	body := fmt.Sprintf(
		"标的：%s\n数据日期：%s\nPE(TTM)：%.2f\nPB：%.2f\nPS(TTM)：%.2f\nPE 历史分位：%.1f（MVP-1 待完善）\nPB 历史分位：%.1f\n来源：%s tier=%s\n采集：%s",
		code, res.GetAsOfDate(), res.GetPeTtm(), res.GetPb(), res.GetPsTtm(),
		res.GetPePercentile(), res.GetPbPercentile(), src, tier, captured,
	)
	return &ResearchPackResult{
		PackID: packIDValuation, Code: code,
		Title: fmt.Sprintf("%s 估值快照 %s", code, res.GetAsOfDate()),
		Summary: fmt.Sprintf("PE %.2f · PB %.2f · PS %.2f", res.GetPeTtm(), res.GetPb(), res.GetPsTtm()),
		Body: body, Source: src, Tier: tier, CapturedAt: captured, SuggestTier: "A",
		Raw: map[string]any{
			"pe_ttm": res.GetPeTtm(), "pb": res.GetPb(), "ps_ttm": res.GetPsTtm(),
			"as_of_date": res.GetAsOfDate(),
		},
	}, nil
}

func fetchMarketPack(ctx context.Context, wc *worker.Client, code string) (*ResearchPackResult, error) {
	res, err := wc.FetchMarketSnapshot(ctx, []string{"000300"})
	if err != nil {
		return nil, err
	}
	src, tier, captured := provenanceFields(res.GetProvenance())
	var lines []string
	lines = append(lines, fmt.Sprintf("对照标的：%s（研究档案上下文）", code))
	lines = append(lines, fmt.Sprintf("采集：%s", captured))
	for _, idx := range res.GetIndices() {
		lines = append(lines, fmt.Sprintf("%s %s：收盘 %.2f，涨跌 %.2f%%", idx.GetName(), idx.GetCode(), idx.GetClose(), idx.GetChangePct()))
	}
	if res.GetSummary() != "" {
		lines = append(lines, res.GetSummary())
	}
	body := strings.Join(lines, "\n")
	summary := res.GetSummary()
	if summary == "" && len(res.GetIndices()) > 0 {
		idx := res.GetIndices()[0]
		summary = fmt.Sprintf("%s %.2f (%.2f%%)", idx.GetName(), idx.GetClose(), idx.GetChangePct())
	}
	return &ResearchPackResult{
		PackID: packIDMarket, Code: code,
		Title: fmt.Sprintf("市场基准 %s", time.Now().Format("2006-01-02")),
		Summary: summary,
		Body: body, Source: src, Tier: tier, CapturedAt: captured, SuggestTier: "B",
	}, nil
}

func fetchNewsPack(ctx context.Context, wc *worker.Client, code string) (*ResearchPackResult, error) {
	res, err := wc.FetchAnnouncements(ctx, &dataworkerv1.FetchAnnouncementsRequest{Codes: []string{code}})
	if err != nil {
		return nil, err
	}
	var lines []string
	for i, it := range res.GetItems() {
		if i >= 10 {
			break
		}
		lines = append(lines, fmt.Sprintf("- [%s] %s\n  %s", it.GetPublishedAt(), it.GetTitle(), it.GetUrl()))
	}
	if len(lines) == 0 {
		lines = append(lines, "暂无新闻条目")
	}
	for _, e := range res.GetErrors() {
		lines = append(lines, fmt.Sprintf("（拉取部分失败 %s: %s）", e.GetCode(), e.GetMessage()))
	}
	body := strings.Join(lines, "\n")
	src, tier, captured := "akshare", "B", time.Now().Format(time.RFC3339)
	if len(res.GetItems()) > 0 && res.GetItems()[0].GetProvenance() != nil {
		src, tier, captured = provenanceFields(res.GetItems()[0].GetProvenance())
	}
	return &ResearchPackResult{
		PackID: packIDNews, Code: code,
		Title: fmt.Sprintf("%s 新闻列表 %s", code, time.Now().Format("2006-01-02")),
		Summary: fmt.Sprintf("共 %d 条", len(res.GetItems())),
		Body: body, Source: src, Tier: tier, CapturedAt: captured, SuggestTier: "C",
	}, nil
}

const (
	packIDQuote      = "quote"
	packIDValuation  = "company_valuation"
	packIDSector     = "sector_valuation"
	packIDVolume     = "volume"
	packIDMarket     = "market_benchmark"
	packIDNews       = "news"
)

func provenanceFields(p interface {
	GetSource() string
	GetTier() string
	GetCapturedAt() string
}) (source, tier, captured string) {
	if p == nil {
		return "unknown", "B", time.Now().Format(time.RFC3339)
	}
	source, tier, captured = p.GetSource(), p.GetTier(), p.GetCapturedAt()
	if captured == "" {
		captured = time.Now().Format(time.RFC3339)
	}
	return source, tier, captured
}
