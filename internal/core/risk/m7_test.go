package risk

import (
	"strings"
	"testing"

	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
	"github.com/shopspring/decimal"
)

func TestM7SingleStockHardBlock(t *testing.T) {
	rules := &yamlstore.RiskRules{
		PositionLimits: yamlstore.PositionLimits{
			SingleStock: yamlstore.LimitPair{WarningPct: 10, HardBlockPct: 15},
		},
	}
	portfolio := &yamlstore.Portfolio{
		Positions: []yamlstore.PortfolioPosition{
			{Code: "600519", State: "holding", PositionPct: decimal.NewFromInt(12)},
		},
	}
	res, err := Evaluate(EvaluateInput{
		Scenario: "buy", Code: "600519", ProposedPct: 5,
		Portfolio: portfolio, RiskRules: rules,
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.ApproveBlocked {
		t.Fatalf("expected hard block, checks=%+v", res.Checks)
	}
}

func TestR004SkippedWhenNoLibraryReason(t *testing.T) {
	rules := &yamlstore.RiskRules{
		PositionLimits: yamlstore.PositionLimits{
			SingleStock: yamlstore.LimitPair{WarningPct: 10, HardBlockPct: 15},
		},
	}
	redlines := &yamlstore.PersonalRedlines{
		Redlines: []yamlstore.Redline{{
			ID: "r004", Rule: "无素材建仓", Severity: "hard", Enabled: true,
			RelatedScenarios: []string{"buy"},
		}},
	}
	payload := `{"related_library_ids":[],"no_library_reason":"个人判断"}`
	res, err := Evaluate(EvaluateInput{
		Scenario: "buy", Code: "600519", ProposedPct: 5,
		Portfolio: nil, RiskRules: rules, Redlines: redlines,
		PayloadJSON: payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range res.HardBlocks {
		if c.RuleID == "r004" {
			t.Fatalf("r004 should not fire when no_library_reason set: %+v", res)
		}
	}
}

func TestValidateExceptionShowsRuleSummary(t *testing.T) {
	hard := []CheckResult{{RuleID: "r004", RuleSource: "personal_redlines", Severity: "hard_block", Message: "无素材建仓"}}
	err := ValidateException(hard, nil, "")
	if err == nil || !strings.Contains(err.Error(), "r004") {
		t.Fatalf("expected r004 in error, got %v", err)
	}
}

func TestValidateExceptionHardBlock(t *testing.T) {
	hard := []CheckResult{{RuleID: "m7_single_stock", Severity: "hard_block"}}
	err := ValidateException(hard, nil, `{}`)
	if err == nil {
		t.Fatal("expected error for missing exception")
	}
}
