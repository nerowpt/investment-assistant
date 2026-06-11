package checklist

import (
	"encoding/json"
	"testing"
)

func TestGetFormSchemaReview(t *testing.T) {
	s, err := GetFormSchema("review")
	if err != nil {
		t.Fatal(err)
	}
	if s.ChecklistType != "review" || len(s.Fields) < 5 {
		t.Fatalf("unexpected schema: %+v", s)
	}
}

func TestGetFormSchemaBuy(t *testing.T) {
	s, err := GetFormSchema("buy")
	if err != nil {
		t.Fatal(err)
	}
	if s.ChecklistType != "buy" || len(s.Fields) < 5 {
		t.Fatalf("unexpected schema: %+v", s)
	}
}

func TestBuildPayloadFromFlat(t *testing.T) {
	flat := map[string]any{
		"position_size_plan.initial_pct": 5.0,
		"execution_price":                1680.0,
		"shares":                         100.0,
	}
	out := BuildPayloadFromFlat(flat)
	psp, ok := out["position_size_plan"].(map[string]any)
	if !ok || psp["initial_pct"] != 5.0 {
		t.Fatalf("nested field missing: %+v", out)
	}

	// 空字符串不应污染数字嵌套字段
	skipEmpty := map[string]any{
		"position_size_plan.initial_pct": "",
		"position_size_plan.max_pct":     10.0,
	}
	out2 := BuildPayloadFromFlat(skipEmpty)
	psp2, ok := out2["position_size_plan"].(map[string]any)
	if !ok {
		t.Fatalf("position_size_plan missing: %+v", out2)
	}
	if _, has := psp2["initial_pct"]; has {
		t.Fatalf("empty initial_pct should be skipped: %+v", psp2)
	}
	if psp2["max_pct"] != 10.0 {
		t.Fatalf("max_pct: %+v", psp2)
	}

	// 数字字符串应转为 float64
	strNum := map[string]any{"position_size_plan.initial_pct": "7.5"}
	out3 := BuildPayloadFromFlat(strNum)
	psp3 := out3["position_size_plan"].(map[string]any)
	if psp3["initial_pct"] != 7.5 {
		t.Fatalf("string number coerce: %+v", psp3)
	}
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateDraftPayload("buy", string(b)); err != nil {
		t.Fatalf("built payload invalid: %v raw=%s", err, string(b))
	}
}
