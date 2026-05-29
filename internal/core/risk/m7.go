package risk

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
	"github.com/shopspring/decimal"
)

// CheckResult 单条 M7 / 禁区检查结果（写入 risk_guardrail_result_json.checks[]）。
type CheckResult struct {
	RuleID     string `json:"rule_id"`     // 规则 id：m7_single_stock / r004 等
	RuleSource string `json:"rule_source"` // risk_rules | personal_redlines
	Severity   string `json:"severity"`    // warning | hard_block
	Message    string `json:"message"`     // 人类可读说明（CLI show 直接展示）
}

// GuardrailResult M7 完整结果快照（持久化到 checklist_submissions.risk_guardrail_result_json）。
type GuardrailResult struct {
	Scenario       string        `json:"scenario"`        // 触发场景：buy / add 等
	Checks         []CheckResult `json:"checks"`          // 全部检查项（含 warning 与 hard_block）
	HardBlocks     []CheckResult `json:"hard_blocks"`     // hard_block 子集，便于 H5 approve 门禁
	Warnings       []CheckResult `json:"warnings"`        // warning 子集
	ApproveBlocked bool          `json:"approve_blocked"` // true 时 H5 approve 须例外或拒绝
}

// EvaluateInput M7 评估输入（submit 时由 checklist 服务组装）。
type EvaluateInput struct {
	Scenario    string                  // checklist_type：buy | add | …
	Code        string                  // 标的代码
	SectorID    string                  // 来自 portfolio 或 payload（新建仓可空）
	ThesisID    string                  // 来自 portfolio 或 payload（新建仓可空）
	ProposedPct float64                 // 拟增仓位 %：buy=initial_pct，add=add_pct
	Portfolio   *yamlstore.Portfolio    // 当前 Layer A 持仓视图
	RiskRules   *yamlstore.RiskRules    // M7 阈值
	Redlines    *yamlstore.PersonalRedlines // 个人禁区
	PayloadJSON string                  // 完整 checklist payload，供 builtin redline 判断
}

// Evaluate 运行 M7 仓位护栏 + 已启用 personal_redlines（builtin 简版，见 02 §18.4）。
func Evaluate(in EvaluateInput) (*GuardrailResult, error) {
	if in.RiskRules == nil {
		return nil, fmt.Errorf("risk_rules 未加载")
	}
	res := &GuardrailResult{Scenario: in.Scenario}
	pct := func(code, sector, thesis string) (stock, sectorSum, thesisSum, total float64) {
		if in.Portfolio == nil {
			return 0, 0, 0, 0
		}
		for _, p := range in.Portfolio.Positions {
			if p.State != "holding" {
				continue
			}
			v, _ := p.PositionPct.Float64()
			total += v
			if p.Code == code {
				stock += v
			}
			if sector != "" && p.SectorID == sector {
				sectorSum += v
			}
			if thesis != "" && p.ThesisID == thesis {
				thesisSum += v
			}
		}
		return stock, sectorSum, thesisSum, total
	}

	stock, sectorSum, thesisSum, total := pct(in.Code, in.SectorID, in.ThesisID)
	afterStock := stock + in.ProposedPct
	afterSector := sectorSum + in.ProposedPct
	afterThesis := thesisSum + in.ProposedPct
	afterTotal := total + in.ProposedPct

	limits := in.RiskRules.PositionLimits
	addLimitCheck := func(ruleID, label string, after, warn, hard float64) {
		if hard > 0 && after > hard {
			if in.hasLegacyBlock(in.Code, in.SectorID) && in.RiskRules.LegacyOverLimit.BlockExpansionOnLegacy {
				c := CheckResult{
					RuleID: ruleID, RuleSource: "risk_rules", Severity: "hard_block",
					Message: fmt.Sprintf("%s 拟达 %.2f%%，超过 hard_block %.0f%%（存量 legacy 禁止继续扩大）", label, after, hard),
				}
				res.Checks = append(res.Checks, c)
				res.HardBlocks = append(res.HardBlocks, c)
				res.ApproveBlocked = true
				return
			}
			c := CheckResult{
				RuleID: ruleID, RuleSource: "risk_rules", Severity: "hard_block",
				Message: fmt.Sprintf("%s 拟达 %.2f%%，超过 hard_block %.0f%%", label, after, hard),
			}
			res.Checks = append(res.Checks, c)
			res.HardBlocks = append(res.HardBlocks, c)
			res.ApproveBlocked = true
		} else if warn > 0 && after > warn {
			c := CheckResult{
				RuleID: ruleID, RuleSource: "risk_rules", Severity: "warning",
				Message: fmt.Sprintf("%s 拟达 %.2f%%，超过 warning %.0f%%", label, after, warn),
			}
			res.Checks = append(res.Checks, c)
			res.Warnings = append(res.Warnings, c)
		}
	}

	if in.Scenario == "buy" || in.Scenario == "add" {
		addLimitCheck("m7_single_stock", "单标的仓位", afterStock,
			limits.SingleStock.WarningPct, limits.SingleStock.HardBlockPct)
		if in.SectorID != "" {
			addLimitCheck("m7_single_sector", "单板块集中度", afterSector,
				limits.SingleSector.WarningPct, limits.SingleSector.HardBlockPct)
		}
		if in.ThesisID != "" {
			addLimitCheck("m7_single_thesis", "单一 thesis 暴露", afterThesis,
				limits.SingleThesis.WarningPct, limits.SingleThesis.HardBlockPct)
		}
		addLimitCheck("m7_total_equity", "整体股票仓位", afterTotal,
			limits.TotalEquity.WarningPct, limits.TotalEquity.HardBlockPct)
	}

	evalRedlines(in, res)
	return res, nil
}

func (in EvaluateInput) hasLegacyBlock(code, sectorID string) bool {
	if in.Portfolio == nil {
		return false
	}
	for _, p := range in.Portfolio.Positions {
		if p.State != "holding" {
			continue
		}
		for _, f := range p.LegacyFlags {
			if f == "legacy_over_limit" && (p.Code == code || (sectorID != "" && p.SectorID == sectorID)) {
				return true
			}
		}
	}
	return false
}

// evalRedlines 评估已启用的 personal_redlines（MVP-1 仅 r003/r004 builtin）。
func evalRedlines(in EvaluateInput, res *GuardrailResult) {
	if in.Redlines == nil {
		return
	}
	var raw map[string]any
	_ = json.Unmarshal([]byte(in.PayloadJSON), &raw)

	for _, rl := range in.Redlines.Redlines {
		if !rl.Enabled {
			continue
		}
		if !scenarioMatch(rl.RelatedScenarios, in.Scenario) {
			continue
		}
		switch rl.ID {
		case "r003":
			if onlyValuationRepair(raw) {
				c := CheckResult{
					RuleID: rl.ID, RuleSource: "personal_redlines", Severity: rl.Severity,
					Message: "expected_return_driver 仅含 valuation_repair：" + rl.Rule,
				}
				res.Checks = append(res.Checks, c)
				if rl.Severity == "hard" {
					res.HardBlocks = append(res.HardBlocks, c)
					res.ApproveBlocked = true
				} else {
					res.Warnings = append(res.Warnings, c)
				}
			}
		case "r004":
			if in.Scenario == "buy" && len(stringList(raw["related_library_ids"])) == 0 {
				// 02 §16.2：填 no_library_reason 的纯个人判断建仓允许，不触发 r004
				if strVal(raw["no_library_reason"]) != "" {
					continue
				}
				c := CheckResult{
					RuleID: rl.ID, RuleSource: "personal_redlines", Severity: rl.Severity,
					Message: rl.Rule,
				}
				res.Checks = append(res.Checks, c)
				if rl.Severity == "hard" {
					res.HardBlocks = append(res.HardBlocks, c)
					res.ApproveBlocked = true
				} else {
					res.Warnings = append(res.Warnings, c)
				}
			}
		}
	}
}

func scenarioMatch(scenarios []string, scenario string) bool {
	for _, s := range scenarios {
		if s == scenario {
			return true
		}
	}
	return false
}

func onlyValuationRepair(raw map[string]any) bool {
	drivers, ok := raw["expected_return_driver"].([]any)
	if !ok || len(drivers) != 1 {
		return false
	}
	s, _ := drivers[0].(string)
	return strings.EqualFold(s, "valuation_repair")
}

func stringList(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, it := range arr {
		if s, ok := it.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func strVal(v any) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// ToJSON 序列化 GuardrailResult 为 JSON 字符串。
func (g *GuardrailResult) ToJSON() (string, error) {
	b, err := json.Marshal(g)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ExceptionPayload hard/soft 例外说明（持久化到 checklist_submissions.exception_json，02 §18.7）。
type ExceptionPayload struct {
	TriggeredRuleID      string   `json:"triggered_rule_id,omitempty"`     // 触发的规则 id
	ExceptionReason      string   `json:"exception_reason,omitempty"`      // hard 时 ≥80 字
	ExpectedCompensation string   `json:"expected_compensation,omitempty"` // 若判断错误如何补偿/复盘
	ReviewDate           string   `json:"review_date,omitempty"`           // 计划复盘日 YYYY-MM-DD
	LibraryItemIDs       []string `json:"library_item_ids,omitempty"`      // 至少 1 条 S/A tier lib_* id
	ConfirmText          string   `json:"confirm_text,omitempty"`          // 用户确认句
	Acked                bool     `json:"acked,omitempty"`                 // soft 警示时须 true
	AckNote              string   `json:"ack_note,omitempty"`              // soft 可选说明，建议 ≥20 字
}

// ValidateException 校验 exception_json 是否满足 hard/soft 要求（submit 时调用）。
func ValidateException(hardBlocks, warnings []CheckResult, exceptionJSON string) error {
	if len(hardBlocks) == 0 && len(warnings) == 0 {
		return nil
	}
	var ex ExceptionPayload
	if exceptionJSON != "" {
		if err := json.Unmarshal([]byte(exceptionJSON), &ex); err != nil {
			return fmt.Errorf("exception_json 非法: %w\n%s", err, FormatChecksSummary(hardBlocks, warnings))
		}
	}
	if len(hardBlocks) > 0 {
		if exceptionJSON == "" {
			return fmt.Errorf("触发 hard_block，须提供 --exception-file\n%s\n要求：exception_reason≥80 字、expected_compensation、review_date、confirm_text、library_item_ids（S/A tier）",
				FormatChecksSummary(hardBlocks, warnings))
		}
		if len([]rune(ex.ExceptionReason)) < 80 {
			return fmt.Errorf("exception_reason 不足 80 字（当前 %d 字）\n%s",
				len([]rune(ex.ExceptionReason)), FormatChecksSummary(hardBlocks, warnings))
		}
		if ex.ExpectedCompensation == "" || ex.ReviewDate == "" || ex.ConfirmText == "" {
			return fmt.Errorf("hard_block 例外须填写 expected_compensation、review_date、confirm_text\n%s",
				FormatChecksSummary(hardBlocks, warnings))
		}
		if len(ex.LibraryItemIDs) == 0 {
			return fmt.Errorf("hard_block 例外须至少 1 条 S/A tier library_item_id\n%s",
				FormatChecksSummary(hardBlocks, warnings))
		}
		ex.TriggeredRuleID = hardBlocks[0].RuleID
		return nil
	}
	if len(warnings) > 0 && !ex.Acked {
		return fmt.Errorf("存在 soft 警示，须提供 --exception-file 且 acked=true\n%s",
			FormatChecksSummary(hardBlocks, warnings))
	}
	return nil
}

// FormatChecksSummary 格式化 M7/禁区结果，供 CLI 错误提示。
func FormatChecksSummary(hardBlocks, warnings []CheckResult) string {
	var b strings.Builder
	if len(hardBlocks) > 0 {
		b.WriteString("hard_block:\n")
		for _, c := range hardBlocks {
			fmt.Fprintf(&b, "  - [%s/%s] %s\n", c.RuleID, c.RuleSource, c.Message)
		}
	}
	if len(warnings) > 0 {
		b.WriteString("warning:\n")
		for _, c := range warnings {
			fmt.Fprintf(&b, "  - [%s/%s] %s\n", c.RuleID, c.RuleSource, c.Message)
		}
	}
	return strings.TrimSpace(b.String())
}

// SectorThesisFromPortfolio 从 portfolio 读取已有持仓的 sector/thesis（新建仓时通常为空）。
func SectorThesisFromPortfolio(p *yamlstore.Portfolio, code string) (sectorID, thesisID string) {
	if p == nil {
		return "", ""
	}
	for _, pos := range p.Positions {
		if pos.Code == code && pos.State == "holding" {
			return pos.SectorID, pos.ThesisID
		}
	}
	return "", ""
}

// SumEquityPct 当前 holding 标的 position_pct 之和。
func SumEquityPct(p *yamlstore.Portfolio) decimal.Decimal {
	if p == nil {
		return decimal.Zero
	}
	sum := decimal.Zero
	for _, pos := range p.Positions {
		if pos.State == "holding" {
			sum = sum.Add(pos.PositionPct)
		}
	}
	return sum
}
