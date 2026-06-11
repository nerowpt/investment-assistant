package query

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
)

// PoolZone 选股看板分区。
type PoolZone struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Count       int        `json:"count"`
	Items       []PoolItem `json:"items"`
}

// PoolItem 看板卡片条目。
type PoolItem struct {
	Code          string   `json:"code"`
	Name          string   `json:"name"`
	Zone          string   `json:"zone"`
	RefID         string   `json:"ref_id,omitempty"`
	Summary       string   `json:"summary,omitempty"`
	LibraryCount  int      `json:"library_count"`
	ReviewDate    string   `json:"review_date,omitempty"`
	PositionPct   string   `json:"position_pct,omitempty"`
	PositionType  string   `json:"position_type,omitempty"`
	State         string   `json:"state,omitempty"`
	EntryDate     string   `json:"entry_date,omitempty"`
	ClosedAt      string   `json:"closed_at,omitempty"`
	SellCount     int      `json:"sell_count,omitempty"` // 已 approve 的卖出次数（含减仓）
	PoolTags      []string `json:"pool_tags,omitempty"`
	Actions       []string `json:"actions"`
}

// PoolResponse 选股看板 API 响应体。
type PoolResponse struct {
	Zones     []PoolZone `json:"zones"`
	UpdatedAt string     `json:"updated_at"`
}

// BuyContextPrefill 买入向导预填字段（扁平，对齐 buy checklist）。
type BuyContextPrefill struct {
	SourceEntry         string   `json:"source_entry"`
	WatchlistOriginID   string   `json:"watchlist_origin_id,omitempty"`
	RelatedLibraryIDs   []string `json:"related_library_ids"`
	BuyReasonSummary    string   `json:"buy_reason_summary,omitempty"`
	InvestmentThesis    string   `json:"investment_thesis,omitempty"`
}

// BuyContextLibraryItem 买入上下文展示的 L1 摘要。
type BuyContextLibraryItem struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Tier    string `json:"tier"`
	Summary string `json:"summary,omitempty"`
}

// BuyContextResponse 从观察区/研究区进入买入时的上下文。
type BuyContextResponse struct {
	Code          string                  `json:"code"`
	Name          string                  `json:"name"`
	From          string                  `json:"from"`
	WatchID       string                  `json:"watch_id,omitempty"`
	Prefill       BuyContextPrefill       `json:"prefill"`
	LibraryItems  []BuyContextLibraryItem `json:"library_items"`
}

var poolZoneDefs = []struct {
	id, title, desc string
}{
	{"watching", "观察区", "待验证假设，尚未建仓"},
	{"researching", "研究区", "已有 L1 素材，未进观察池与持仓"},
	{"holding", "已建仓", "当前持有中"},
	{"closed", "已卖出", "仅已清仓标的；减仓后仍留在「已建仓」"},
	{"swing", "波段/做T", "position_type=swing 的持仓或历史"},
}

// BuildPool 聚合 watchlist + portfolio + L1 为五分区看板。
func (r *Reader) BuildPool(zoneFilter string) (*PoolResponse, error) {
	p, err := yamlstore.LoadPortfolio(r.ac.PortfolioPath())
	if err != nil {
		return nil, err
	}
	wl, err := yamlstore.LoadWatchlist(r.ac.WatchlistPath())
	if err != nil {
		return nil, err
	}
	libCounts, err := r.libraryCountByCode()
	if err != nil {
		return nil, err
	}
	sellCounts, err := sqlstore.CountSellJournalsByCode(r.db)
	if err != nil {
		return nil, err
	}

	watchingCodes := map[string]bool{}
	portfolioCodes := map[string]bool{}
	zoneItems := map[string][]PoolItem{
		"watching":    {},
		"researching": {},
		"holding":     {},
		"closed":      {},
		"swing":       {},
	}

	for _, it := range wl.Items {
		if it.RemovedReason != "" || it.Code == "" {
			continue
		}
		watchingCodes[it.Code] = true
		summary := firstNonEmpty(it.Hypothesis, it.WatchReason)
		zoneItems["watching"] = append(zoneItems["watching"], PoolItem{
			Code:         it.Code,
			Name:         it.Name,
			Zone:         "watching",
			RefID:        it.ID,
			Summary:      summary,
			LibraryCount: libCounts[it.Code],
			ReviewDate:   it.ReviewDate,
			State:        "watching",
			Actions:      []string{"research", "buy", "records"},
		})
	}

	for _, pos := range p.Positions {
		if pos.Code == "" {
			continue
		}
		portfolioCodes[pos.Code] = true
		summary := truncateRunes(pos.InvestmentThesis, 80)
		if summary == "" {
			summary = pos.Notes
		}
		item := PoolItem{
			Code:         pos.Code,
			Name:         pos.Name,
			Summary:      summary,
			LibraryCount: libCounts[pos.Code],
			PositionPct:  pos.PositionPct.String(),
			PositionType: pos.PositionType,
			EntryDate:    pos.EntryDate,
		}
		switch pos.State {
		case "holding":
			item.Zone = "holding"
			item.State = "holding"
			item.SellCount = sellCounts[pos.Code]
			item.Actions = []string{"add", "inspect", "sell", "records"}
			zoneItems["holding"] = append(zoneItems["holding"], item)
			if pos.PositionType == "swing" {
				swing := item
				swing.Zone = "swing"
				swing.PoolTags = []string{"swing"}
				zoneItems["swing"] = append(zoneItems["swing"], swing)
			}
		case "closed":
			item.Zone = "closed"
			item.State = "closed"
			item.ClosedAt = pos.ClosedAt
			item.Actions = []string{"review", "records"}
			zoneItems["closed"] = append(zoneItems["closed"], item)
			if pos.PositionType == "swing" {
				swing := item
				swing.Zone = "swing"
				swing.PoolTags = []string{"swing"}
				zoneItems["swing"] = append(zoneItems["swing"], swing)
			}
		}
	}

	for code, cnt := range libCounts {
		if cnt == 0 || watchingCodes[code] || portfolioCodes[code] {
			continue
		}
		name := code
		zoneItems["researching"] = append(zoneItems["researching"], PoolItem{
			Code:         code,
			Name:         name,
			Zone:         "researching",
			Summary:      "已有研究素材，可建仓或先入观察池",
			LibraryCount: cnt,
			State:        "researching",
			Actions:      []string{"research", "buy"},
		})
	}

	updatedAt := time.Now().Format(time.RFC3339)
	if p.Meta.UpdatedAt != "" {
		updatedAt = p.Meta.UpdatedAt
	}

	zones := make([]PoolZone, 0, len(poolZoneDefs))
	for _, def := range poolZoneDefs {
		if zoneFilter != "" && zoneFilter != def.id {
			continue
		}
		items := zoneItems[def.id]
		if items == nil {
			items = []PoolItem{}
		}
		zones = append(zones, PoolZone{
			ID:          def.id,
			Title:       def.title,
			Description: def.desc,
			Count:       len(items),
			Items:       items,
		})
	}
	return &PoolResponse{Zones: zones, UpdatedAt: updatedAt}, nil
}

// BuildBuyContext 生成从观察区/研究区进入买入向导的预填上下文。
func (r *Reader) BuildBuyContext(code, from, watchID string) (*BuyContextResponse, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, errPoolCodeRequired
	}
	if from == "" {
		from = "research"
	}

	wl, _ := yamlstore.LoadWatchlist(r.ac.WatchlistPath())
	p, _ := yamlstore.LoadPortfolio(r.ac.PortfolioPath())

	name := code
	prefill := BuyContextPrefill{
		SourceEntry:       "manual",
		RelatedLibraryIDs: []string{},
	}
	if from == "watchlist" {
		prefill.SourceEntry = "from_watchlist"
	}
	resp := &BuyContextResponse{
		Code: code,
		Name: name,
		From: from,
		WatchID: watchID,
		Prefill: prefill,
	}

	if wl != nil {
		for _, it := range wl.Items {
			if it.Code != code || it.RemovedReason != "" {
				continue
			}
			if watchID != "" && it.ID != watchID {
				continue
			}
			resp.WatchID = it.ID
			resp.Name = firstNonEmpty(it.Name, code)
			prefill.WatchlistOriginID = it.ID
			prefill.BuyReasonSummary = it.WatchReason
			prefill.InvestmentThesis = it.Hypothesis
			if len(it.RelatedLibraryIDs) > 0 {
				prefill.RelatedLibraryIDs = append([]string(nil), it.RelatedLibraryIDs...)
			}
			break
		}
	}
	if p != nil {
		for _, pos := range p.Positions {
			if pos.Code == code && pos.Name != "" {
				resp.Name = pos.Name
				break
			}
		}
	}

	rows, err := sqlstore.ListLibraryItems(r.db, "active", code, "")
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, id := range prefill.RelatedLibraryIDs {
		seen[id] = true
	}
	for _, row := range rows {
		if !seen[row.ID] {
			prefill.RelatedLibraryIDs = append(prefill.RelatedLibraryIDs, row.ID)
			seen[row.ID] = true
		}
		resp.LibraryItems = append(resp.LibraryItems, BuyContextLibraryItem{
			ID:      row.ID,
			Title:   row.Title,
			Tier:    row.Tier,
			Summary: row.SummaryByUser,
		})
	}
	if prefill.RelatedLibraryIDs == nil {
		prefill.RelatedLibraryIDs = []string{}
	}
	resp.Prefill = prefill
	return resp, nil
}

func (r *Reader) libraryCountByCode() (map[string]int, error) {
	rows, err := sqlstore.ListLibraryItems(r.db, "active", "", "")
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, row := range rows {
		var stocks []string
		_ = json.Unmarshal([]byte(row.RelatedStocksJSON), &stocks)
		for _, code := range stocks {
			code = strings.TrimSpace(code)
			if code != "" {
				out[code]++
			}
		}
	}
	return out, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func truncateRunes(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= max {
		return string(r)
	}
	return string(r[:max]) + "…"
}

type poolError string

func (e poolError) Error() string { return string(e) }

const errPoolCodeRequired poolError = "code 必填"
