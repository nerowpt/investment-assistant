package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	chksvc "github.com/investment-assistant/investment-assistant/internal/core/checklist"
	"github.com/investment-assistant/investment-assistant/internal/core/query"
)

type draftRequest struct {
	ChecklistType string         `json:"checklist_type"`
	Code          string         `json:"code"`
	Name          string         `json:"name"`
	Payload       map[string]any `json:"payload"`
	Values        map[string]any `json:"values"` // 扁平表单值，与 payload 二选一
}

type submitRequest struct {
	EmotionSelfCheck string         `json:"emotion_self_check"`
	Exception        map[string]any `json:"exception"`
}

type rejectRequest struct {
	Reason string `json:"reason"`
}

type riskCheckRequest struct {
	Scenario                string  `json:"scenario"`
	Code                    string  `json:"code"`
	PlannedPositionPctAfter float64 `json:"planned_position_pct_after"`
	SectorID                string  `json:"sector_id"`
	ThesisID                string  `json:"thesis_id"`
}

func (s *Server) handleChecklistCreate(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	deps := newDeps(ac)
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	var req draftRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.ChecklistType == "" {
		WriteError(w, http.StatusBadRequest, "missing_type", "checklist_type 必填")
		return
	}

	payloadJSON := ""
	if req.Payload != nil {
		b, err := json.Marshal(req.Payload)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_payload", err.Error())
			return
		}
		payloadJSON = string(b)
	} else if len(req.Values) > 0 {
		built := chksvc.BuildPayloadFromFlat(req.Values)
		b, err := json.Marshal(built)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_values", err.Error())
			return
		}
		payloadJSON = string(b)
	}

	res, err := deps.checklist(db).CreateDraft(chksvc.DraftInput{
		ChecklistType: req.ChecklistType,
		Code:          req.Code,
		Name:          req.Name,
		PayloadJSON:   payloadJSON,
	})
	if err != nil {
		WriteError(w, http.StatusBadRequest, "draft_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusCreated, toDraftResultJSON(res))
}

func (s *Server) handleChecklistUpdate(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	deps := newDeps(ac)
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")
	var req draftRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	payloadJSON := ""
	if req.Payload != nil {
		b, err := json.Marshal(req.Payload)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_payload", err.Error())
			return
		}
		payloadJSON = string(b)
	}

	res, err := deps.checklist(db).UpdateDraft(chksvc.UpdateDraftInput{
		ID:          id,
		Code:        req.Code,
		Name:        req.Name,
		PayloadJSON: payloadJSON,
		Values:      req.Values,
	})
	if err != nil {
		WriteError(w, http.StatusBadRequest, "update_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, toDraftResultJSON(res))
}

func (s *Server) handleChecklistPreview(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	deps := newDeps(ac)
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")
	res, err := deps.checklist(db).PreviewSubmit(id)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "preview_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, res)
}

func (s *Server) handleChecklistSubmit(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	deps := newDeps(ac)
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")
	var req submitRequest
	_ = DecodeJSON(r, &req) // 允许空 body

	exceptionJSON := ""
	if req.Exception != nil {
		b, _ := json.Marshal(req.Exception)
		exceptionJSON = string(b)
	}

	res, err := deps.checklist(db).Submit(chksvc.SubmitInput{
		ID:               id,
		EmotionSelfCheck: req.EmotionSelfCheck,
		ExceptionJSON:    exceptionJSON,
	})
	if err != nil {
		WriteError(w, http.StatusBadRequest, "submit_failed", err.Error())
		return
	}

	cs, _ := deps.checklist(db).Get(id)
	var riskResult any
	if cs != nil && cs.RiskGuardrailResultJSON != "" {
		_ = json.Unmarshal([]byte(cs.RiskGuardrailResultJSON), &riskResult)
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"submit":      toSubmitResultJSON(res),
		"risk_result": riskResult,
	})
}

func (s *Server) handleChecklistPlan(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	deps := newDeps(ac)
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")
	res, err := deps.checklist(db).PlanSell(id)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "plan_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, res)
}

func (s *Server) handleChecklistApprove(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	deps := newDeps(ac)
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")
	res, err := deps.checklistWithMarket(r.Context(), db).Approve(context.Background(), id)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "approve_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, toApproveResultJSON(res))
}

func (s *Server) handleChecklistReject(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	deps := newDeps(ac)
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")
	var req rejectRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	res, err := deps.checklist(db).Reject(chksvc.RejectInput{ID: id, Reason: req.Reason})
	if err != nil {
		WriteError(w, http.StatusBadRequest, "reject_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, res)
}

func (s *Server) handleRiskCheck(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	deps := newDeps(ac)
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	var req riskCheckRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.Scenario == "" || req.Code == "" {
		WriteError(w, http.StatusBadRequest, "missing_field", "scenario 与 code 必填")
		return
	}
	data, err := deps.reader(db).CheckPositionAgainstRules(r.Context(), query.CheckPositionInput{
		Scenario:                req.Scenario,
		Code:                    req.Code,
		PlannedPositionPctAfter: req.PlannedPositionPctAfter,
		SectorID:                req.SectorID,
		ThesisID:                req.ThesisID,
	})
	if err != nil {
		WriteError(w, http.StatusBadRequest, "risk_check_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, data)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "ia-api"})
}
