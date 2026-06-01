package checklist

import "github.com/shopspring/decimal"

// WeightedAvgCost 加权平均成本价（元/股）。
func WeightedAvgCost(oldCost, oldShares, newCost, newShares decimal.Decimal) decimal.Decimal {
	totalShares := oldShares.Add(newShares)
	if totalShares.LessThanOrEqual(decimal.Zero) {
		return newCost
	}
	return oldCost.Mul(oldShares).Add(newCost.Mul(newShares)).Div(totalShares)
}

// PctAfterShareSell 卖出后 lot/portfolio 剩余仓位 %。
func PctAfterShareSell(currentPct, currentShares, soldShares decimal.Decimal) decimal.Decimal {
	if currentShares.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}
	after := currentShares.Sub(soldShares)
	if after.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}
	return currentPct.Mul(after).Div(currentShares)
}

// SellPctFromShares 由卖出股数推算占组合仓位百分点（portfolio 层）。
func SellPctFromShares(positionPct, totalShares, sellShares decimal.Decimal) decimal.Decimal {
	if totalShares.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}
	return positionPct.Mul(sellShares).Div(totalShares)
}