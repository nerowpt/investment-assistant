package checklist

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/investment-assistant/investment-assistant/internal/core/lot"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"github.com/shopspring/decimal"
)

// SellPlanResult FIFO 卖出计划预览（CLI plan 用）。
type SellPlanResult struct {
	ChecklistID    string                  `json:"checklist_id"`
	Code           string                  `json:"code"`
	SellShares     string                  `json:"sell_shares"`
	ExecutionPrice string                  `json:"execution_price"`
	MatchMethod    string                  `json:"match_method"`
	Plan           []LotAllocationPlanItem `json:"lot_allocation_plan"`
	OpenLots       []SellPlanOpenLot       `json:"open_lots"`
}

// SellPlanOpenLot 可卖 lot 摘要。
type SellPlanOpenLot struct {
	LotID         string `json:"lot_id"`
	OpenAt        string `json:"open_at"`
	CurrentShares string `json:"current_shares"`
	CurrentPct    string `json:"current_pct"`
	CostBasis     string `json:"cost_basis"`
}

// PlanSell 为 sell checklist 生成/校验 lot_allocation_plan（股数 FIFO）。
func (s *Service) PlanSell(checklistID string) (*SellPlanResult, error) {
	cs, err := sqlstore.GetChecklistSubmission(s.db, checklistID)
	if err != nil {
		return nil, err
	}
	if cs == nil {
		return nil, fmt.Errorf("checklist 不存在: %s", checklistID)
	}
	if cs.ChecklistType != "sell" {
		return nil, fmt.Errorf("仅 sell checklist 可 plan（当前 type=%s）", cs.ChecklistType)
	}
	payload, err := ParseSellPayload(cs.PayloadJSON)
	if err != nil {
		return nil, err
	}
	sellShares := decimal.NewFromFloat(payload.SellShares)
	openLots, err := loadOpenLots(s.db, cs.Code)
	if err != nil {
		return nil, err
	}
	plan, method, err := resolveSellPlanShares(sellShares, openLots, payload.LotAllocationPlan)
	if err != nil {
		return nil, err
	}
	return &SellPlanResult{
		ChecklistID:    cs.ID,
		Code:           cs.Code,
		SellShares:     sellShares.StringFixed(0),
		ExecutionPrice: decimal.NewFromFloat(payload.ExecutionPrice).StringFixed(4),
		MatchMethod:    method,
		Plan:           planItemsToPayload(plan),
		OpenLots:       openLotsSummary(openLots),
	}, nil
}

// SetPayload 更新 draft/submitted checklist 的 payload_json。
func (s *Service) SetPayload(checklistID, payloadJSON string) error {
	cs, err := sqlstore.GetChecklistSubmission(s.db, checklistID)
	if err != nil {
		return err
	}
	if cs == nil {
		return fmt.Errorf("checklist 不存在: %s", checklistID)
	}
	if cs.Status != "draft" && cs.Status != "submitted" {
		return fmt.Errorf("仅 draft/submitted 可更新 payload（当前 status=%s）", cs.Status)
	}
	if err := ValidateDraftPayload(cs.ChecklistType, payloadJSON); err != nil {
		return err
	}
	if cs.Status == "submitted" {
		if err := ValidatePayload(s.db, cs.ChecklistType, cs.Code, payloadJSON); err != nil {
			return err
		}
	}
	return sqlstore.UpdateChecklistPayload(s.db, checklistID, payloadJSON)
}

func loadOpenLots(db *sql.DB, code string) ([]lot.OpenLot, error) {
	rows, err := sqlstore.ListOpenLotsByCode(db, code)
	if err != nil {
		return nil, err
	}
	var out []lot.OpenLot
	for _, l := range rows {
		out = append(out, lot.OpenLot{
			ID:            l.ID,
			Code:          l.Code,
			OpenAt:        l.OpenAt,
			CurrentPct:    parseLotDecimal(l.CurrentPct),
			CurrentShares: parseLotDecimal(l.Shares),
			CostBasis:     parseLotDecimal(l.CostBasis),
		})
	}
	return out, nil
}

func resolveSellPlanShares(sellShares decimal.Decimal, openLots []lot.OpenLot, fromPayload []LotAllocationPlanItem) ([]lot.PlanItem, string, error) {
	if len(fromPayload) > 0 {
		plan := payloadToPlanItemsShares(fromPayload, openLots)
		if err := lot.ValidatePlanShares(sellShares, openLots, plan); err != nil {
			return nil, "", err
		}
		return plan, lot.MatchMethod(plan), nil
	}
	plan, err := lot.RecommendAllocationsShares(sellShares, openLots)
	if err != nil {
		return nil, "", err
	}
	return plan, lot.MatchMethod(plan), nil
}

func payloadToPlanItemsShares(items []LotAllocationPlanItem, openLots []lot.OpenLot) []lot.PlanItem {
	byID := lotByID(openLots)
	var out []lot.PlanItem
	for _, it := range items {
		shares := decimal.NewFromFloat(it.AllocatedShares)
		pct := decimal.NewFromFloat(it.AllocatedPct)
		if ol, ok := byID[it.LotID]; ok {
			if shares.IsZero() && !pct.IsZero() && !ol.CurrentPct.IsZero() {
				shares = ol.CurrentShares.Mul(pct).Div(ol.CurrentPct)
			}
			if pct.IsZero() && !shares.IsZero() && !ol.CurrentShares.IsZero() {
				pct = ol.CurrentPct.Mul(shares).Div(ol.CurrentShares)
			}
		}
		out = append(out, lot.PlanItem{
			LotID:           it.LotID,
			AllocatedShares: shares,
			AllocatedPct:    pct,
			UserAdjusted:    it.UserAdjusted,
		})
	}
	return out
}

func planItemsToPayload(plan []lot.PlanItem) []LotAllocationPlanItem {
	var out []LotAllocationPlanItem
	for _, it := range plan {
		sh, _ := it.AllocatedShares.Float64()
		pct, _ := it.AllocatedPct.Float64()
		out = append(out, LotAllocationPlanItem{
			LotID:           it.LotID,
			AllocatedShares: sh,
			AllocatedPct:    pct,
			UserAdjusted:    it.UserAdjusted,
		})
	}
	return out
}

func openLotsSummary(lots []lot.OpenLot) []SellPlanOpenLot {
	var out []SellPlanOpenLot
	for _, l := range lots {
		out = append(out, SellPlanOpenLot{
			LotID:         l.ID,
			OpenAt:        l.OpenAt,
			CurrentShares: l.CurrentShares.StringFixed(0),
			CurrentPct:    l.CurrentPct.StringFixed(4),
			CostBasis:     l.CostBasis.StringFixed(4),
		})
	}
	return out
}

func parseLotDecimal(s string) decimal.Decimal {
	if s == "" {
		return decimal.Zero
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

func mergePlanIntoPayload(payloadJSON string, plan []lot.PlanItem) (string, error) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(payloadJSON), &raw); err != nil {
		return "", err
	}
	arr := make([]any, 0, len(plan))
	for _, it := range plan {
		sh, _ := it.AllocatedShares.Float64()
		pct, _ := it.AllocatedPct.Float64()
		arr = append(arr, map[string]any{
			"lot_id":           it.LotID,
			"allocated_shares": sh,
			"allocated_pct":    pct,
			"user_adjusted":    it.UserAdjusted,
		})
	}
	raw["lot_allocation_plan"] = arr
	b, err := json.Marshal(raw)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func lotByID(openLots []lot.OpenLot) map[string]lot.OpenLot {
	m := make(map[string]lot.OpenLot, len(openLots))
	for _, l := range openLots {
		m[l.ID] = l
	}
	return m
}
