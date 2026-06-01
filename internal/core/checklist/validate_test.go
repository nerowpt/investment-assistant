package checklist

import (
	"encoding/json"
	"testing"
)

func TestValidateBuyMissingInitialPct(t *testing.T) {
	payload := `{
  "source_entry":"manual","position_type":"core","buy_reason_summary":"x",
  "investment_thesis":"x","expected_return_driver":["earnings_growth"],
  "target_price":10,"stop_loss":8,"reversal_conditions":["a"],
  "position_size_plan":{"max_pct":10},
  "opportunity_cost_benchmark":"HS300","confidence":"medium","emotion_tag":"calm",
  "identified_risks":["r"],"no_library_reason":"个人判断",
  "execution_price":10,"shares":100,"emotion_retrospect":null
}`
	err := ValidatePayload(nil, "buy", "600519", payload)
	if err == nil || err.Error() != "buy payload 缺少 position_size_plan.initial_pct" {
		t.Fatalf("expected initial_pct error, got %v", err)
	}
}

func TestValidateBuyTierAck(t *testing.T) {
	// 无 DB 时跳过 tier 校验路径
	payload := `{
  "source_entry":"manual","position_type":"core","buy_reason_summary":"x",
  "investment_thesis":"x","expected_return_driver":["earnings_growth"],
  "target_price":10,"stop_loss":8,"reversal_conditions":["a"],
  "position_size_plan":{"initial_pct":5,"max_pct":10},
  "opportunity_cost_benchmark":"HS300","confidence":"medium","emotion_tag":"calm",
  "identified_risks":["r"],"related_library_ids":["lib_x"],
  "tier_acknowledgement":false,"execution_price":10,"shares":100,"emotion_retrospect":null
}`
	err := ValidatePayload(nil, "buy", "600519", payload)
	if err != nil {
		t.Fatalf("without db lib tier check skipped: %v", err)
	}
}

func TestEmotionNeedsSelfCheck(t *testing.T) {
	if !EmotionNeedsSelfCheck("fomo") {
		t.Fatal("fomo should need self check")
	}
	if EmotionNeedsSelfCheck("calm") {
		t.Fatal("calm should not")
	}
}

func TestDefaultPayloadHasEmotionRetrospect(t *testing.T) {
	raw := DefaultPayloadTemplate("buy")
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["emotion_retrospect"]; !ok {
		t.Fatal("missing emotion_retrospect")
	}
}
