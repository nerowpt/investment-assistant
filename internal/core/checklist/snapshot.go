package checklist

import (
	"context"
	"encoding/json"
	"time"

	dataworkerv1 "github.com/investment-assistant/investment-assistant/gen/go/dataworker/v1"
)

const snapshotSchemaVersion = 1

// BuildBuySnapshot 组装 buy/add 决策快照 JSON（02 §4.5 + worker 事实）。
func BuildBuySnapshot(ctx context.Context, code string, fetcher MarketFetcher) (string, error) {
	now := time.Now().Format(time.RFC3339)
	out := map[string]any{
		"schema_version": snapshotSchemaVersion,
		"code":           code,
		"captured_at":    now,
	}
	if fetcher != nil {
		if price, chg, src, tier, err := fetcher.FetchQuote(ctx, code); err == nil {
			out["quote"] = map[string]any{
				"price": price, "change_pct": chg, "source": src, "tier": tier,
			}
		} else {
			out["quote_error"] = err.Error()
		}
		if pe, pb, src, tier, err := fetcher.FetchValuation(ctx, code); err == nil {
			out["valuation"] = map[string]any{
				"pe_ttm": pe, "pb": pb, "source": src, "tier": tier,
			}
		} else {
			out["valuation_error"] = err.Error()
		}
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// WorkerMarketFetcher 适配 internal/worker Client。
type WorkerMarketFetcher struct {
	Client interface {
		FetchQuote(ctx context.Context, code string) (*dataworkerv1.FetchQuoteResponse, error)
		FetchValuation(ctx context.Context, code string) (*dataworkerv1.FetchValuationResponse, error)
	}
}

// FetchQuote 实现 MarketFetcher。
func (w *WorkerMarketFetcher) FetchQuote(ctx context.Context, code string) (float64, float64, string, string, error) {
	res, err := w.Client.FetchQuote(ctx, code)
	if err != nil {
		return 0, 0, "", "", err
	}
	p := res.GetProvenance()
	return res.GetPrice(), res.GetChangePct(), p.GetSource(), p.GetTier(), nil
}

// FetchValuation 实现 MarketFetcher。
func (w *WorkerMarketFetcher) FetchValuation(ctx context.Context, code string) (float64, float64, string, string, error) {
	res, err := w.Client.FetchValuation(ctx, code)
	if err != nil {
		return 0, 0, "", "", err
	}
	p := res.GetProvenance()
	return res.GetPeTtm(), res.GetPb(), p.GetSource(), p.GetTier(), nil
}
