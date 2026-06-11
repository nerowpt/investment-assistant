package checklist

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/investment-assistant/investment-assistant/internal/core/lot"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"github.com/shopspring/decimal"
)

const PayloadSchemaVersion = 1

var validTypes = map[string]bool{
	"watch": true, "buy": true, "add": true, "inspect": true,
	"sell": true, "review": true, "import": true,
}

// ValidateType 校验 checklist 类型名。
func ValidateType(t string) error {
	if !validTypes[t] {
		return fmt.Errorf("未知 checklist_type: %s", t)
	}
	return nil
}

// ValidateDraftPayload draft 阶段仅校验 JSON 与预留位。
func ValidateDraftPayload(checklistType, payloadJSON string) error {
	if err := ValidateType(checklistType); err != nil {
		return err
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(payloadJSON), &raw); err != nil {
		return fmt.Errorf("payload_json 不是合法 JSON: %w", err)
	}
	if _, ok := raw["emotion_retrospect"]; !ok {
		return fmt.Errorf("payload 须含 emotion_retrospect 预留位（可设为 null）")
	}
	return nil
}

// ValidatePayload 按类型校验 payload_json（submit 用）。
func ValidatePayload(db *sql.DB, checklistType, code, payloadJSON string) error {
	if err := ValidateType(checklistType); err != nil {
		return err
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(payloadJSON), &raw); err != nil {
		return fmt.Errorf("payload_json 不是合法 JSON: %w", err)
	}
	if _, ok := raw["emotion_retrospect"]; !ok {
		return fmt.Errorf("payload 须含 emotion_retrospect 预留位（可设为 null）")
	}
	switch checklistType {
	case "buy":
		return validateBuy(db, code, raw)
	case "add":
		return validateAdd(db, raw)
	case "watch":
		return validateWatch(raw)
	case "inspect":
		return validateInspect(raw)
	case "sell":
		return validateSell(db, code, raw)
	case "review":
		return validateReview(raw)
	case "import":
		return validateImport(raw)
	default:
		return nil
	}
}

func validateBuy(db *sql.DB, code string, raw map[string]any) error {
	req := []string{
		"source_entry", "position_type", "buy_reason_summary", "investment_thesis",
		"expected_return_driver", "target_price", "stop_loss", "reversal_conditions",
		"position_size_plan", "opportunity_cost_benchmark", "confidence", "emotion_tag",
		"identified_risks", "execution_price", "shares",
	}
	for _, k := range req {
		if !hasKey(raw, k) {
			return fmt.Errorf("buy payload 缺少必填字段: %s", k)
		}
	}
	plan, ok := raw["position_size_plan"].(map[string]any)
	if !ok {
		return fmt.Errorf("position_size_plan 须为 object")
	}
	if !hasNumber(plan, "initial_pct") {
		return fmt.Errorf("buy payload 缺少 position_size_plan.initial_pct")
	}
	if !hasNumber(plan, "max_pct") {
		return fmt.Errorf("buy payload 缺少 position_size_plan.max_pct")
	}
	if !hasNumber(raw, "execution_price") || numberVal(raw["execution_price"]) <= 0 {
		return fmt.Errorf("execution_price 须 > 0（实际成交价，手动填写）")
	}
	if !hasNumber(raw, "shares") || numberVal(raw["shares"]) <= 0 {
		return fmt.Errorf("shares 须 > 0")
	}
	drivers, ok := raw["expected_return_driver"].([]any)
	if !ok || len(drivers) == 0 {
		return fmt.Errorf("expected_return_driver 至少 1 项")
	}
	revs, ok := raw["reversal_conditions"].([]any)
	if !ok || len(revs) == 0 {
		return fmt.Errorf("reversal_conditions 至少 1 项")
	}
	risks, ok := raw["identified_risks"].([]any)
	if !ok || len(risks) == 0 {
		return fmt.Errorf("identified_risks 至少 1 项")
	}
	libIDs := stringList(raw["related_library_ids"])
	if len(libIDs) == 0 {
		if strVal(raw["no_library_reason"]) == "" {
			return fmt.Errorf("无 related_library_ids 时须填写 no_library_reason")
		}
	} else {
		if err := validateLibraryRefs(db, libIDs); err != nil {
			return err
		}
		if needTierAck(db, libIDs) && !boolVal(raw["tier_acknowledgement"]) {
			return fmt.Errorf("主要依据素材最高 tier 为 C/D 时须 tier_acknowledgement=true")
		}
	}
	_ = code
	return nil
}

func validateAdd(db *sql.DB, raw map[string]any) error {
	req := []string{
		"linked_buy_journal_id", "add_trigger", "add_reason_summary",
		"thesis_change", "add_pct", "max_pct_after_add", "emotion_tag",
		"execution_price", "shares",
	}
	for _, k := range req {
		if !hasKey(raw, k) {
			return fmt.Errorf("add payload 缺少必填字段: %s", k)
		}
	}
	if !hasNumber(raw, "add_pct") {
		return fmt.Errorf("add_pct 须为数字")
	}
	if !hasNumber(raw, "execution_price") || numberVal(raw["execution_price"]) <= 0 {
		return fmt.Errorf("execution_price 须 > 0（实际成交价，手动填写）")
	}
	if !hasNumber(raw, "shares") || numberVal(raw["shares"]) <= 0 {
		return fmt.Errorf("shares 须 > 0")
	}
	libIDs := stringList(raw["related_library_ids"])
	if len(libIDs) > 0 {
		if err := validateLibraryRefs(db, libIDs); err != nil {
			return err
		}
		if needTierAck(db, libIDs) && !boolVal(raw["tier_acknowledgement"]) {
			return fmt.Errorf("主要依据素材最高 tier 为 C/D 时须 tier_acknowledgement=true")
		}
	}
	return nil
}

func validateWatch(raw map[string]any) error {
	for _, k := range []string{"watch_reason", "hypothesis", "review_date", "priority"} {
		if !hasKey(raw, k) {
			return fmt.Errorf("watch payload 缺少必填字段: %s", k)
		}
	}
	return nil
}

func validateInspect(raw map[string]any) error {
	for _, k := range []string{
		"inspection_type", "linked_buy_journal_id", "classification", "planned_action",
	} {
		if !hasKey(raw, k) {
			return fmt.Errorf("inspect payload 缺少必填字段: %s", k)
		}
	}
	return nil
}

func validateSell(db *sql.DB, code string, raw map[string]any) error {
	for _, k := range []string{
		"sell_type", "sell_trigger", "sell_reason", "sell_shares", "execution_price",
		"emotion_tag", "lesson",
	} {
		if !hasKey(raw, k) {
			return fmt.Errorf("sell payload 缺少必填字段: %s", k)
		}
	}
	if !hasNumber(raw, "sell_shares") || numberVal(raw["sell_shares"]) <= 0 {
		return fmt.Errorf("sell_shares 须 > 0")
	}
	if !hasNumber(raw, "execution_price") || numberVal(raw["execution_price"]) <= 0 {
		return fmt.Errorf("execution_price 须 > 0（实际成交价，手动填写）")
	}
	sellShares := decimal.NewFromFloat(numberVal(raw["sell_shares"]))
	if db != nil && code != "" {
		openLots, err := loadOpenLotsForValidate(db, code)
		if err != nil {
			return err
		}
		sum := decimal.Zero
		for _, l := range openLots {
			sum = sum.Add(l.CurrentShares)
		}
		if sum.LessThan(sellShares) {
			return fmt.Errorf("sell_shares=%s 超过 open lot 可卖股数 %s", sellShares.String(), sum.String())
		}
	}
	if planRaw, ok := raw["lot_allocation_plan"].([]any); ok && len(planRaw) > 0 {
		if db == nil || code == "" {
			return nil
		}
		openLots, err := loadOpenLotsForValidate(db, code)
		if err != nil {
			return err
		}
		plan := payloadMapToPlanItemsShares(planRaw, openLots)
		if err := lot.ValidatePlanShares(sellShares, openLots, plan); err != nil {
			return fmt.Errorf("lot_allocation_plan: %w", err)
		}
	}
	return nil
}

func payloadMapToPlanItemsShares(planRaw []any, openLots []lot.OpenLot) []lot.PlanItem {
	byID := map[string]lot.OpenLot{}
	for _, l := range openLots {
		byID[l.ID] = l
	}
	var plan []lot.PlanItem
	for _, it := range planRaw {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		shares := decimal.NewFromFloat(numberVal(m["allocated_shares"]))
		pct := decimal.NewFromFloat(numberVal(m["allocated_pct"]))
		lotID := strVal(m["lot_id"])
		if ol, ok := byID[lotID]; ok {
			if shares.IsZero() && !pct.IsZero() && !ol.CurrentPct.IsZero() {
				shares = ol.CurrentShares.Mul(pct).Div(ol.CurrentPct)
			}
			if pct.IsZero() && !shares.IsZero() && !ol.CurrentShares.IsZero() {
				pct = ol.CurrentPct.Mul(shares).Div(ol.CurrentShares)
			}
		}
		plan = append(plan, lot.PlanItem{
			LotID:           lotID,
			AllocatedShares: shares,
			AllocatedPct:    pct,
			UserAdjusted:    boolVal(m["user_adjusted"]),
		})
	}
	return plan
}

func loadOpenLotsForValidate(db *sql.DB, code string) ([]lot.OpenLot, error) {
	rows, err := sqlstore.ListOpenLotsByCode(db, code)
	if err != nil {
		return nil, err
	}
	var out []lot.OpenLot
	for _, l := range rows {
		cp, _ := decimal.NewFromString(l.CurrentPct)
		sh, _ := decimal.NewFromString(l.Shares)
		cb, _ := decimal.NewFromString(l.CostBasis)
		out = append(out, lot.OpenLot{ID: l.ID, Code: l.Code, OpenAt: l.OpenAt, CurrentPct: cp, CurrentShares: sh, CostBasis: cb})
	}
	return out, nil
}

func validateReview(raw map[string]any) error {
	if !hasKey(raw, "review_type") {
		return fmt.Errorf("review payload 缺少必填字段: review_type")
	}
	reviewType := strings.TrimSpace(fmt.Sprint(raw["review_type"]))
	if reviewType == "lot_attribution" {
		for _, k := range []string{"target_lot_id", "target_code", "period_start", "period_end", "confirmed_patterns"} {
			if !hasKey(raw, k) {
				return fmt.Errorf("review payload 缺少必填字段: %s", k)
			}
		}
		if att, ok := raw["attribution"].(map[string]any); !ok || !hasKey(att, "result_category") {
			return fmt.Errorf("review payload 缺少必填字段: attribution.result_category")
		}
	} else {
		for _, k := range []string{"period_start", "period_end", "confirmed_patterns"} {
			if !hasKey(raw, k) {
				return fmt.Errorf("review payload 缺少必填字段: %s", k)
			}
		}
	}
	patterns, ok := raw["confirmed_patterns"].([]any)
	if !ok || len(patterns) == 0 {
		return fmt.Errorf("confirmed_patterns 至少 1 项")
	}
	return nil
}

func validateImport(raw map[string]any) error {
	positions, ok := raw["positions"].([]any)
	if !ok || len(positions) == 0 {
		return fmt.Errorf("import payload.positions 至少 1 项")
	}
	return nil
}

func validateLibraryRefs(db *sql.DB, ids []string) error {
	for _, id := range ids {
		if strings.HasPrefix(id, "lc_") {
			return fmt.Errorf("不得引用 candidate（lc_*）: %s", id)
		}
		if db == nil {
			continue
		}
		item, err := sqlstore.GetLibraryItem(db, id)
		if err != nil {
			return err
		}
		if item == nil {
			return fmt.Errorf("library_item 不存在: %s", id)
		}
		if item.Status != "active" {
			return fmt.Errorf("library_item 非 active: %s", id)
		}
	}
	return nil
}

var tierRank = map[string]int{"S": 5, "A": 4, "B": 3, "C": 2, "D": 1}

func needTierAck(db *sql.DB, libIDs []string) bool {
	if db == nil || len(libIDs) == 0 {
		return false
	}
	maxRank := 0
	for _, id := range libIDs {
		item, err := sqlstore.GetLibraryItem(db, id)
		if err != nil || item == nil {
			continue
		}
		if r, ok := tierRank[strings.ToUpper(item.Tier)]; ok && r > maxRank {
			maxRank = r
		}
	}
	return maxRank > 0 && maxRank <= tierRank["C"]
}

// EmotionNeedsSelfCheck 是否需二次确认文案。
func EmotionNeedsSelfCheck(emotionTag string) bool {
	switch strings.ToLower(emotionTag) {
	case "fomo", "greedy", "anxious":
		return true
	default:
		return false
	}
}

// ExtractEmotionTag 从 payload 读取 emotion_tag。
func ExtractEmotionTag(payloadJSON string) string {
	var raw map[string]any
	if json.Unmarshal([]byte(payloadJSON), &raw) != nil {
		return ""
	}
	return strings.ToLower(strVal(raw["emotion_tag"]))
}

func hasKey(m map[string]any, k string) bool {
	v, ok := m[k]
	return ok && v != nil
}

func hasNumber(m map[string]any, k string) bool {
	switch m[k].(type) {
	case float64, int, json.Number:
		return true
	default:
		return false
	}
}

func boolVal(v any) bool {
	b, _ := v.(bool)
	return b
}

func strVal(v any) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func stringList(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, it := range arr {
		if s, ok := it.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

// ProposedPct 从 payload 提取拟增仓位（buy/add）。
func ProposedPct(checklistType, payloadJSON string) (float64, error) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(payloadJSON), &raw); err != nil {
		return 0, err
	}
	switch checklistType {
	case "buy":
		plan, ok := raw["position_size_plan"].(map[string]any)
		if !ok {
			return 0, fmt.Errorf("缺少 position_size_plan")
		}
		return numberVal(plan["initial_pct"]), nil
	case "add":
		return numberVal(raw["add_pct"]), nil
	default:
		return 0, nil
	}
}

func numberVal(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}
