package checklist

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/investment-assistant/investment-assistant/internal/core/account"
	"github.com/investment-assistant/investment-assistant/internal/core/ids"
	"github.com/investment-assistant/investment-assistant/internal/core/risk"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore/schema"
	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
)

// Service Checklist draft/submit/approve 用例。
type Service struct {
	ac     *account.Context
	db     *sql.DB
	market MarketFetcher // 可选；approve 时拉 snapshot
}

// NewService 构造 checklist 服务。
func NewService(ac *account.Context, db *sql.DB) *Service {
	return &Service{ac: ac, db: db}
}

// DraftInput 创建 draft 参数。
type DraftInput struct {
	ChecklistType string // watch | buy | add | inspect | sell | review | import
	Code          string // 标的代码；review/import 可空
	Name          string // 标的或主题名称
	PayloadJSON   string // 完整 payload；空则写入 DefaultPayloadTemplate
}

// DraftResult 创建结果。
type DraftResult struct {
	ID     string // cs_{YYYYMMDD}_{seq}
	Status string // draft
}

// SubmitInput 提交参数。
type SubmitInput struct {
	ID               string // checklist id
	EmotionSelfCheck string // fomo/greedy/anxious 二次确认文案（写入 emotion_self_check 列）
	ExceptionJSON    string // hard/soft 例外 JSON（写入 exception_json 列）
}

// SubmitResult 提交结果。
type SubmitResult struct {
	ID             string // checklist id
	Status         string // submitted
	ApproveBlocked bool   // M7 hard_block 时 true（H5 approve 门禁）
	HardBlockCount int    // hard_block 条数
	WarningCount   int    // warning 条数
}

// CreateDraft 创建 draft checklist。
func (s *Service) CreateDraft(in DraftInput) (*DraftResult, error) {
	if err := ValidateType(in.ChecklistType); err != nil {
		return nil, err
	}
	payload := in.PayloadJSON
	if payload == "" {
		payload = DefaultPayloadTemplate(in.ChecklistType)
	}
	if err := ValidateDraftPayload(in.ChecklistType, payload); err != nil {
		return nil, err
	}
	id, err := ids.Next(s.db, "cs")
	if err != nil {
		return nil, err
	}
	linkedJSON, _ := json.Marshal(extractLinkedLibraryIDs(payload))
	now := nowISO()
	row := &schema.ChecklistSubmission{
		ID:                   id,
		ChecklistType:        in.ChecklistType,
		Code:                 in.Code,
		Name:                 in.Name,
		PayloadJSON:          payload,
		PayloadSchemaVersion: PayloadSchemaVersion,
		Status:               "draft",
		SubmittedBy:          "user",
		LinkedLibraryIDsJSON: string(linkedJSON),
		CreatedAt:            now,
	}
	if err := sqlstore.InsertChecklistSubmission(s.db, row); err != nil {
		return nil, err
	}
	return &DraftResult{ID: id, Status: "draft"}, nil
}

// Submit draft → submitted，运行 M7 并写 risk_exceptions。
func (s *Service) Submit(in SubmitInput) (*SubmitResult, error) {
	cs, err := sqlstore.GetChecklistSubmission(s.db, in.ID)
	if err != nil {
		return nil, err
	}
	if cs == nil {
		return nil, fmt.Errorf("checklist 不存在: %s", in.ID)
	}
	if cs.Status != "draft" {
		return nil, fmt.Errorf("仅 draft 可 submit（当前 status=%s）", cs.Status)
	}
	if err := ValidatePayload(s.db, cs.ChecklistType, cs.Code, cs.PayloadJSON); err != nil {
		return nil, err
	}
	emotionTag := ExtractEmotionTag(cs.PayloadJSON)
	if EmotionNeedsSelfCheck(emotionTag) && in.EmotionSelfCheck == "" {
		return nil, fmt.Errorf("emotion_tag=%s 须二次确认：请用 --emotion-check 填写自检说明", emotionTag)
	}

	portfolio, _ := yamlstore.LoadPortfolio(s.ac.PortfolioPath())
	rules, err := yamlstore.LoadRiskRules(s.ac.RiskRulesPath())
	if err != nil {
		return nil, fmt.Errorf("读取 risk_rules: %w", err)
	}
	redlines, err := yamlstore.LoadPersonalRedlines(s.ac.PersonalRedlinesPath())
	if err != nil {
		return nil, fmt.Errorf("读取 personal_redlines: %w", err)
	}
	proposed, err := ProposedPct(cs.ChecklistType, cs.PayloadJSON)
	if err != nil {
		return nil, err
	}
	sectorID, thesisID := risk.SectorThesisFromPortfolio(portfolio, cs.Code)
	guard, err := risk.Evaluate(risk.EvaluateInput{
		Scenario:    cs.ChecklistType,
		Code:        cs.Code,
		SectorID:    sectorID,
		ThesisID:    thesisID,
		ProposedPct: proposed,
		Portfolio:   portfolio,
		RiskRules:   rules,
		Redlines:    redlines,
		PayloadJSON: cs.PayloadJSON,
	})
	if err != nil {
		return nil, err
	}
	if err := risk.ValidateException(guard.HardBlocks, guard.Warnings, in.ExceptionJSON); err != nil {
		return nil, err
	}
	riskJSON, err := guard.ToJSON()
	if err != nil {
		return nil, err
	}

	submittedAt := nowISO()
	tierAck := tierAckFromPayload(cs.PayloadJSON)
	if err := sqlstore.UpdateChecklistSubmitted(s.db, cs.ID, riskJSON, in.ExceptionJSON, in.EmotionSelfCheck, tierAck, submittedAt); err != nil {
		return nil, err
	}
	if err := writeRiskExceptions(s.db, cs.ID, guard, in.ExceptionJSON); err != nil {
		return nil, err
	}
	return &SubmitResult{
		ID:             cs.ID,
		Status:         "submitted",
		ApproveBlocked: guard.ApproveBlocked,
		HardBlockCount: len(guard.HardBlocks),
		WarningCount:   len(guard.Warnings),
	}, nil
}

// Get 查询单条。
func (s *Service) Get(id string) (*schema.ChecklistSubmission, error) {
	return sqlstore.GetChecklistSubmission(s.db, id)
}

// List 列表。
func (s *Service) List(f sqlstore.ChecklistListFilter) ([]schema.ChecklistSubmission, error) {
	return sqlstore.ListChecklistSubmissions(s.db, f)
}

func writeRiskExceptions(db *sql.DB, checklistID string, guard *risk.GuardrailResult, exceptionJSON string) error {
	var ex risk.ExceptionPayload
	_ = json.Unmarshal([]byte(exceptionJSON), &ex)
	for _, c := range guard.Checks {
		rxID, err := ids.Next(db, "rx")
		if err != nil {
			return err
		}
		row := &schema.RiskException{
			ID:                    rxID,
			Severity:              c.Severity,
			RuleSource:            c.RuleSource,
			RuleID:                c.RuleID,
			ChecklistSubmissionID: checklistID,
			ExceptionReason:       ex.ExceptionReason,
			ExpectedCompensation:  ex.ExpectedCompensation,
			ReviewDate:            ex.ReviewDate,
			CreatedAt:             nowISO(),
		}
		if err := sqlstore.InsertRiskException(db, row); err != nil {
			return err
		}
	}
	return nil
}

func tierAckFromPayload(payloadJSON string) *int {
	var raw map[string]any
	if json.Unmarshal([]byte(payloadJSON), &raw) != nil {
		return nil
	}
	if b, ok := raw["tier_acknowledgement"].(bool); ok && b {
		v := 1
		return &v
	}
	return nil
}

func extractLinkedLibraryIDs(payloadJSON string) []string {
	var raw map[string]any
	if json.Unmarshal([]byte(payloadJSON), &raw) != nil {
		return nil
	}
	return stringList(raw["related_library_ids"])
}

func nowISO() string {
	return time.Now().Format(time.RFC3339)
}

// DefaultPayloadTemplate 返回带 emotion_retrospect 预留位的最小模板。
func DefaultPayloadTemplate(checklistType string) string {
	templates := map[string]string{
		"buy": `{
  "source_entry": "manual",
  "position_type": "core",
  "buy_reason_summary": "",
  "investment_thesis": "",
  "expected_return_driver": ["earnings_growth"],
  "target_price": 0,
  "stop_loss": 0,
  "reversal_conditions": ["待填写"],
  "position_size_plan": {"initial_pct": 5, "max_pct": 10},
  "opportunity_cost_benchmark": "HS300",
  "confidence": "medium",
  "emotion_tag": "calm",
  "identified_risks": ["待填写"],
  "related_library_ids": [],
  "no_library_reason": null,
  "tier_acknowledgement": false,
  "emotion_retrospect": null
}`,
		"watch": `{"watch_reason":"","hypothesis":"","review_date":"2026-06-30","priority":"medium","related_library_ids":[],"emotion_retrospect":null}`,
		"add":   `{"linked_buy_journal_id":"","add_trigger":"thesis_strengthened","add_reason_summary":"","thesis_change":"strengthened","add_pct":3,"max_pct_after_add":10,"emotion_tag":"calm","related_library_ids":[],"tier_acknowledgement":false,"emotion_retrospect":null}`,
	}
	if t, ok := templates[checklistType]; ok {
		return t
	}
	return `{"emotion_retrospect": null}`
}
