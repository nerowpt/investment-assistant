package checklist

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
)

// RejectInput 作废 checklist 参数。
type RejectInput struct {
	ID     string // cs_*
	Reason string // 作废原因（必填）
}

// RejectResult 作废结果。
type RejectResult struct {
	ID     string // cs_*
	Status string // rejected
	Reason string // 写入 payload._reject_meta.reason
}

// Reject 将 draft/submitted checklist 作废为 rejected（不可逆；修正须新建 draft）。
func (s *Service) Reject(in RejectInput) (*RejectResult, error) {
	cs, err := sqlstore.GetChecklistSubmission(s.db, in.ID)
	if err != nil {
		return nil, err
	}
	if cs == nil {
		return nil, fmt.Errorf("checklist 不存在: %s", in.ID)
	}
	if cs.Status == "approved" {
		return nil, fmt.Errorf("已 approved 不可 reject（须走新 submission 修正）")
	}
	reason := strings.TrimSpace(in.Reason)
	if cs.Status == "rejected" {
		if reason != "" {
			return nil, fmt.Errorf("已 rejected 不可重复作废")
		}
		return &RejectResult{ID: cs.ID, Status: "rejected", Reason: rejectReasonFromPayload(cs.PayloadJSON)}, nil
	}
	if cs.Status != "draft" && cs.Status != "submitted" {
		return nil, fmt.Errorf("仅 draft/submitted 可 reject（当前 status=%s）", cs.Status)
	}
	if reason == "" {
		return nil, fmt.Errorf("须提供作废原因：--reason")
	}
	rejectedAt := nowISO()
	payload, err := mergeRejectMeta(cs.PayloadJSON, reason, rejectedAt)
	if err != nil {
		return nil, err
	}
	if err := sqlstore.UpdateChecklistRejected(s.db, cs.ID, payload); err != nil {
		return nil, err
	}
	return &RejectResult{ID: cs.ID, Status: "rejected", Reason: reason}, nil
}

func mergeRejectMeta(payloadJSON, reason, rejectedAt string) (string, error) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(payloadJSON), &raw); err != nil {
		return "", err
	}
	raw["_reject_meta"] = map[string]any{
		"reason":      reason,
		"rejected_at": rejectedAt,
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func rejectReasonFromPayload(payloadJSON string) string {
	var raw map[string]any
	if json.Unmarshal([]byte(payloadJSON), &raw) != nil {
		return ""
	}
	meta, ok := raw["_reject_meta"].(map[string]any)
	if !ok {
		return ""
	}
	if s, ok := meta["reason"].(string); ok {
		return s
	}
	return ""
}
