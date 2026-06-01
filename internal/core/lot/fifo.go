// Package lot 提供 lot 级 FIFO 卖出匹配（03 §10A.7 Q4C）。
package lot

import (
	"fmt"
	"sort"

	"github.com/shopspring/decimal"
)

// OpenLot 可参与卖出匹配的 lot 视图（open/partial）。
type OpenLot struct {
	ID            string          // lot_* 主键
	Code          string          // 标的代码
	OpenAt        string          // 开启日期，FIFO 排序键
	CurrentPct    decimal.Decimal // 当前剩余仓位 %（与 portfolio 对齐）
	CurrentShares decimal.Decimal // 当前剩余股数（B 模型 FIFO 主键）
	CostBasis     decimal.Decimal // 成本价（元/股）
}

// PlanItem 卖出分配计划单项（写入 checklist payload lot_allocation_plan[]）。
type PlanItem struct {
	LotID           string          // 被匹配的 lot id
	AllocatedShares decimal.Decimal // 从该 lot 扣减的股数
	AllocatedPct    decimal.Decimal // 同步扣减的仓位 %（doctor/M7）
	UserAdjusted    bool            // true 表示用户改动了系统 FIFO 推荐
}

// RecommendAllocationsShares 按 open_at FIFO 生成股数分配计划。
func RecommendAllocationsShares(sellShares decimal.Decimal, lots []OpenLot) ([]PlanItem, error) {
	if sellShares.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("sell_shares 须 > 0")
	}
	if len(lots) == 0 {
		return nil, fmt.Errorf("无 open/partial lot 可匹配")
	}
	sorted := append([]OpenLot(nil), lots...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].OpenAt != sorted[j].OpenAt {
			return sorted[i].OpenAt < sorted[j].OpenAt
		}
		return sorted[i].ID < sorted[j].ID
	})
	remaining := sellShares
	var plan []PlanItem
	for _, l := range sorted {
		if remaining.LessThanOrEqual(decimal.Zero) {
			break
		}
		if l.CurrentShares.LessThanOrEqual(decimal.Zero) {
			continue
		}
		allocShares := decimal.Min(l.CurrentShares, remaining)
		if allocShares.LessThanOrEqual(decimal.Zero) {
			continue
		}
		allocPct := decimal.Zero
		if l.CurrentShares.GreaterThan(decimal.Zero) {
			allocPct = l.CurrentPct.Mul(allocShares).Div(l.CurrentShares)
		}
		plan = append(plan, PlanItem{
			LotID:           l.ID,
			AllocatedShares: allocShares,
			AllocatedPct:    allocPct,
			UserAdjusted:    false,
		})
		remaining = remaining.Sub(allocShares)
	}
	if remaining.GreaterThan(decimal.Zero) {
		return nil, fmt.Errorf("open lot 可卖股数不足：缺 %s 股", remaining.StringFixed(0))
	}
	return plan, nil
}

// ValidatePlanShares 校验股数计划。
func ValidatePlanShares(sellShares decimal.Decimal, lots []OpenLot, plan []PlanItem) error {
	if sellShares.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("sell_shares 须 > 0")
	}
	if len(plan) == 0 {
		return fmt.Errorf("lot_allocation_plan 为空")
	}
	byID := map[string]OpenLot{}
	for _, l := range lots {
		byID[l.ID] = l
	}
	sumShares := decimal.Zero
	for _, item := range plan {
		if item.LotID == "" {
			return fmt.Errorf("lot_allocation_plan 含空 lot_id")
		}
		l, ok := byID[item.LotID]
		if !ok {
			return fmt.Errorf("lot %s 不可卖（非 open/partial 或不属于该标的）", item.LotID)
		}
		if item.AllocatedShares.LessThanOrEqual(decimal.Zero) {
			return fmt.Errorf("lot %s allocated_shares 须 > 0", item.LotID)
		}
		if item.AllocatedShares.GreaterThan(l.CurrentShares) {
			return fmt.Errorf("lot %s allocated_shares=%s 超过 current_shares=%s",
				item.LotID, item.AllocatedShares.String(), l.CurrentShares.String())
		}
		sumShares = sumShares.Add(item.AllocatedShares)
	}
	if !sumShares.Equal(sellShares) {
		return fmt.Errorf("sum(allocated_shares)=%s != sell_shares=%s", sumShares.String(), sellShares.String())
	}
	return nil
}

// MatchMethod 根据 plan 是否含 user_adjusted 返回 match_method 常量值。
func MatchMethod(plan []PlanItem) string {
	for _, item := range plan {
		if item.UserAdjusted {
			return "user_adjusted"
		}
	}
	return "recommended_fifo"
}

// RealizedReturnPct 计算单笔 allocation 实现收益率 %：((price-cost)/cost)*100。
func RealizedReturnPct(price, cost decimal.Decimal) decimal.Decimal {
	if cost.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}
	return price.Sub(cost).Div(cost).Mul(decimal.NewFromInt(100))
}

// RealizedPnLAmount 实现盈亏（元）= (成交价 - 成本价) × 股数。
func RealizedPnLAmount(execPrice, costBasis, shares decimal.Decimal) decimal.Decimal {
	if shares.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}
	return execPrice.Sub(costBasis).Mul(shares)
}

// LotStatusAfterShareSell 扣减股数后 lot 的 status。
func LotStatusAfterShareSell(currentShares, allocatedShares decimal.Decimal) string {
	after := currentShares.Sub(allocatedShares)
	if after.LessThanOrEqual(decimal.Zero) {
		return "closed"
	}
	if allocatedShares.GreaterThan(decimal.Zero) && after.LessThan(currentShares) {
		return "partial"
	}
	return "open"
}

func pctFromShareSell(currentPct, currentShares, soldShares decimal.Decimal) decimal.Decimal {
	if currentShares.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}
	after := currentShares.Sub(soldShares)
	if after.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}
	return currentPct.Mul(after).Div(currentShares)
}

// SharesAfterSell 扣减后剩余股数。
func SharesAfterSell(currentShares, allocatedShares decimal.Decimal) decimal.Decimal {
	after := currentShares.Sub(allocatedShares)
	if after.LessThan(decimal.Zero) {
		return decimal.Zero
	}
	return after
}
