package query

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore/schema"
)

// ClosedLotSummary 已关闭 lot 卡片摘要（复盘工作台）。
type ClosedLotSummary struct {
	LotID              string `json:"lot_id"`
	ActionType         string `json:"action_type"`
	PositionType       string `json:"position_type"`
	OpenAt             string `json:"open_at"`
	CloseAt            string `json:"close_at"`
	InitialPct         string `json:"initial_pct"`
	CostBasis          string `json:"cost_basis"`
	RealizedReturnPct  string `json:"realized_return_pct,omitempty"`
	RealizedPnLAmount  string `json:"realized_pnl_amount,omitempty"`
	HoldingDays        int    `json:"holding_days"`
	SellJournalID      string `json:"sell_journal_id,omitempty"`
	OpenJournalID      string `json:"open_journal_id"`
	Reviewed           bool   `json:"reviewed"`
	ReviewReportID     string `json:"review_report_id,omitempty"`
}

// SellReviewContext 卖出 journal 中供复盘对照的字段。
type SellReviewContext struct {
	JournalID            string `json:"journal_id"`
	Lesson               string `json:"lesson,omitempty"`
	ThesisResult         string `json:"thesis_result,omitempty"`
	ThesisResultExplain  string `json:"thesis_result_explanation,omitempty"`
	EmotionTag           string `json:"emotion_tag,omitempty"`
	SellReason           string `json:"sell_reason,omitempty"`
	SellReasonDetail     string `json:"sell_reason_detail,omitempty"`
}

// LotReviewPrefill 单笔 lot 复盘向导预填（扁平字段）。
type LotReviewPrefill struct {
	ReviewType       string `json:"review_type"`
	TargetLotID      string `json:"target_lot_id"`
	TargetCode       string `json:"target_code"`
	PeriodStart      string `json:"period_start"`
	PeriodEnd        string `json:"period_end"`
	ResultCategory   string `json:"attribution.result_category,omitempty"`
	ConfirmedPatterns []string `json:"confirmed_patterns"`
	Notes            string `json:"notes,omitempty"`
}

// ReviewWorkbenchResponse 复盘工作台 API 响应。
type ReviewWorkbenchResponse struct {
	Code       string             `json:"code"`
	Name       string             `json:"name"`
	ClosedLots []ClosedLotSummary `json:"closed_lots"`
}

// LotReviewContextResponse 单笔 lot 复盘上下文（向导预填 + 只读对照）。
type LotReviewContextResponse struct {
	Code         string             `json:"code"`
	Name         string             `json:"name"`
	Lot          ClosedLotSummary   `json:"lot"`
	OpenSummary  string             `json:"open_summary,omitempty"`
	SellContext  *SellReviewContext `json:"sell_context,omitempty"`
	Prefill      LotReviewPrefill   `json:"prefill"`
}

// BuildReviewWorkbench 聚合标的已关闭 lot 列表（H8.3 复盘工作台）。
func (r *Reader) BuildReviewWorkbench(code string) (*ReviewWorkbenchResponse, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, fmt.Errorf("code 必填")
	}
	lots, err := sqlstore.ListClosedLotsByCode(r.db, code)
	if err != nil {
		return nil, err
	}
	name := code
	if len(lots) > 0 && lots[0].Name != "" {
		name = lots[0].Name
	}
	out := &ReviewWorkbenchResponse{Code: code, Name: name}
	for _, lot := range lots {
		summary, err := r.buildClosedLotSummary(&lot)
		if err != nil {
			return nil, err
		}
		out.ClosedLots = append(out.ClosedLots, *summary)
	}
	return out, nil
}

// BuildLotReviewContext 单笔 lot 复盘上下文（H8.3 向导预填）。
func (r *Reader) BuildLotReviewContext(code, lotID string) (*LotReviewContextResponse, error) {
	code = strings.TrimSpace(code)
	lotID = strings.TrimSpace(lotID)
	if code == "" || lotID == "" {
		return nil, fmt.Errorf("code 与 lot_id 必填")
	}
	lot, err := sqlstore.GetLotByID(r.db, lotID)
	if err != nil {
		return nil, err
	}
	if lot == nil || lot.Code != code {
		return nil, fmt.Errorf("lot 不存在或与 code 不匹配")
	}
	if lot.Status != schema.LotStatusClosed {
		return nil, fmt.Errorf("仅已关闭 lot 可复盘")
	}
	summary, err := r.buildClosedLotSummary(lot)
	if err != nil {
		return nil, err
	}
	name := lot.Name
	if name == "" {
		name = code
	}
	resp := &LotReviewContextResponse{
		Code: code,
		Name: name,
		Lot:  *summary,
		Prefill: LotReviewPrefill{
			ReviewType:        "lot_attribution",
			TargetLotID:       lotID,
			TargetCode:        code,
			PeriodStart:       dateOnly(lot.OpenAt),
			PeriodEnd:         dateOnly(lot.CloseAt),
			ConfirmedPatterns: []string{},
			Notes:             "",
		},
	}
	if j, err := sqlstore.GetJournal(r.db, lot.JournalID); err == nil && j != nil {
		resp.OpenSummary = j.Summary
	}
	if summary.SellJournalID != "" {
		resp.SellContext, _ = r.loadSellReviewContext(summary.SellJournalID)
	}
	return resp, nil
}

func (r *Reader) buildClosedLotSummary(lot *schema.Lot) (*ClosedLotSummary, error) {
	allocs, err := sqlstore.ListAllocationsByLotID(r.db, lot.ID)
	if err != nil {
		return nil, err
	}
	summary := &ClosedLotSummary{
		LotID:         lot.ID,
		ActionType:    lot.ActionType,
		PositionType:  lot.PositionType,
		OpenAt:        lot.OpenAt,
		CloseAt:       lot.CloseAt,
		InitialPct:    lot.InitialPct,
		CostBasis:     lot.CostBasis,
		HoldingDays:   calcHoldingDays(lot.OpenAt, lot.CloseAt),
		OpenJournalID: lot.JournalID,
	}
	if len(allocs) > 0 {
		summary.SellJournalID = allocs[0].SellJournalID
		summary.RealizedReturnPct = allocs[0].RealizedReturnPct
		summary.RealizedPnLAmount = allocs[0].RealizedPnLAmount
	}
	revID, err := sqlstore.FindLotReviewReportID(r.db, lot.ID)
	if err != nil {
		return nil, err
	}
	if revID != "" {
		summary.Reviewed = true
		summary.ReviewReportID = revID
	}
	return summary, nil
}

func (r *Reader) loadSellReviewContext(journalID string) (*SellReviewContext, error) {
	payloadJSON, err := sqlstore.GetJournalPayload(r.db, journalID)
	if err != nil || payloadJSON == "" {
		return nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(payloadJSON), &raw); err != nil {
		return nil, err
	}
	return &SellReviewContext{
		JournalID:           journalID,
		Lesson:              strFromMap(raw, "lesson"),
		ThesisResult:        strFromMap(raw, "thesis_result"),
		ThesisResultExplain: strFromMap(raw, "thesis_result_explanation"),
		EmotionTag:          strFromMap(raw, "emotion_tag"),
		SellReason:          strFromMap(raw, "sell_reason"),
		SellReasonDetail:    strFromMap(raw, "sell_reason_detail"),
	}, nil
}

func strFromMap(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func dateOnly(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

func calcHoldingDays(openAt, closeAt string) int {
	start, err1 := time.Parse("2006-01-02", dateOnly(openAt))
	end, err2 := time.Parse("2006-01-02", dateOnly(closeAt))
	if err1 != nil || err2 != nil {
		return 0
	}
	d := int(end.Sub(start).Hours() / 24)
	if d < 0 {
		return 0
	}
	return d
}
