package query

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
	"github.com/investment-assistant/investment-assistant/internal/worker"
)

const stockNameCacheTTL = 15 * time.Minute

type stockNameCacheEntry struct {
	name string
	at   time.Time
}

var stockNameCache = struct {
	mu sync.Mutex
	m  map[string]stockNameCacheEntry
}{m: make(map[string]stockNameCacheEntry)}

func cachedStockName(code string) (string, bool) {
	stockNameCache.mu.Lock()
	defer stockNameCache.mu.Unlock()
	e, ok := stockNameCache.m[code]
	if !ok || time.Since(e.at) > stockNameCacheTTL {
		return "", false
	}
	return e.name, true
}

func storeStockName(code, name string) {
	if code == "" || name == "" || name == code {
		return
	}
	stockNameCache.mu.Lock()
	defer stockNameCache.mu.Unlock()
	stockNameCache.m[code] = stockNameCacheEntry{name: name, at: time.Now()}
}

// ResolveStockName 解析标的简称：portfolio/watchlist 优先，否则行情接口。
func (r *Reader) ResolveStockName(ctx context.Context, wc *worker.Client, code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return ""
	}
	if p, err := yamlstore.LoadPortfolio(r.ac.PortfolioPath()); err == nil {
		for _, pos := range p.Positions {
			if pos.Code == code && strings.TrimSpace(pos.Name) != "" && pos.Name != code {
				return pos.Name
			}
		}
	}
	if wl, err := yamlstore.LoadWatchlist(r.ac.WatchlistPath()); err == nil {
		for _, it := range wl.Items {
			if it.Code == code && it.RemovedReason == "" && strings.TrimSpace(it.Name) != "" && it.Name != code {
				return it.Name
			}
		}
	}
	if n, ok := cachedStockName(code); ok {
		return n
	}
	if wc != nil {
		res, err := wc.FetchQuote(ctx, code)
		if err == nil && strings.TrimSpace(res.GetName()) != "" && res.GetName() != code {
			storeStockName(code, res.GetName())
			return res.GetName()
		}
	}
	return code
}

// EnrichPoolNames 为 name==code 的看板条目补全公司简称（portfolio/watchlist 优先，否则行情）。
func (r *Reader) EnrichPoolNames(ctx context.Context, wc *worker.Client, pool *PoolResponse) {
	if pool == nil {
		return
	}
	seen := map[string]string{}
	for _, z := range pool.Zones {
		for i := range z.Items {
			it := &z.Items[i]
			if it.Name != "" && it.Name != it.Code {
				seen[it.Code] = it.Name
			}
		}
	}
	for zi := range pool.Zones {
		for ii := range pool.Zones[zi].Items {
			it := &pool.Zones[zi].Items[ii]
			if it.Name != "" && it.Name != it.Code {
				continue
			}
			if n, ok := seen[it.Code]; ok && n != "" && n != it.Code {
				it.Name = n
				continue
			}
			n := r.ResolveStockName(ctx, wc, it.Code)
			it.Name = n
			seen[it.Code] = n
		}
	}
}
