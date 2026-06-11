package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	chksvc "github.com/investment-assistant/investment-assistant/internal/core/checklist"
	"github.com/investment-assistant/investment-assistant/internal/core/doctor"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
)

func (s *Server) handlePortfolio(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	deps := newDeps(ac)
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	code := r.URL.Query().Get("code")
	includeClosed := r.URL.Query().Get("include_closed") == "true"
	data, err := deps.reader(db).GetPortfolio(code, includeClosed)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "read_error", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, toPortfolioJSON(data))
}

func (s *Server) handleWatchlist(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	deps := newDeps(ac)
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	data, err := deps.reader(db).GetWatchlist(state, code)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "read_error", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, toWatchlistJSON(data))
}

func (s *Server) handleJournals(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	code := r.URL.Query().Get("code")
	action := r.URL.Query().Get("action_type")
	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	rows, err := sqlstore.SearchJournals(db, code, action, limit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "read_error", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"journals": toJournalList(rows)})
}

func (s *Server) handleJournalByID(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")
	row, err := sqlstore.GetJournal(db, id)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "read_error", err.Error())
		return
	}
	if row == nil {
		WriteError(w, http.StatusNotFound, "not_found", "journal 不存在")
		return
	}
	payload, _ := sqlstore.GetJournalPayload(db, id)
	snap, _ := sqlstore.GetDataSnapshotSummary(db, row.DataSnapshotID)
	var payloadObj any
	if payload != "" {
		_ = json.Unmarshal([]byte(payload), &payloadObj)
	}
	var snapSummary any
	if snap != "" {
		_ = json.Unmarshal([]byte(snap), &snapSummary)
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"journal":          toJournalListItem(*row),
		"payload":          payloadObj,
		"snapshot_summary": snapSummary,
	})
}

func (s *Server) handleChecklists(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	deps := newDeps(ac)
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	rows, err := deps.checklist(db).List(sqlstore.ChecklistListFilter{
		Status: r.URL.Query().Get("status"),
		Type:   r.URL.Query().Get("type"),
		Code:   r.URL.Query().Get("code"),
		Limit:  limit,
	})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "read_error", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"checklists": toChecklistList(rows)})
}

func (s *Server) handleChecklistByID(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	deps := newDeps(ac)
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")
	cs, err := deps.checklist(db).Get(id)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "read_error", err.Error())
		return
	}
	if cs == nil {
		WriteError(w, http.StatusNotFound, "not_found", "checklist 不存在")
		return
	}
	var riskResult any
	if cs.RiskGuardrailResultJSON != "" {
		_ = json.Unmarshal([]byte(cs.RiskGuardrailResultJSON), &riskResult)
	}
	var exception any
	if cs.ExceptionJSON != "" {
		_ = json.Unmarshal([]byte(cs.ExceptionJSON), &exception)
	}
	var payload any
	_ = json.Unmarshal([]byte(cs.PayloadJSON), &payload)
	item := toChecklistListItem(*cs)
	WriteJSON(w, http.StatusOK, map[string]any{
		"checklist":          item,
		"payload":            payload,
		"risk_result":        riskResult,
		"exception":          exception,
		"emotion_self_check": cs.EmotionSelfCheck,
	})
}

func (s *Server) handleChecklistSchema(w http.ResponseWriter, r *http.Request) {
	typ := r.URL.Query().Get("type")
	if typ == "" {
		WriteError(w, http.StatusBadRequest, "missing_type", "须指定 type 查询参数")
		return
	}
	schema, err := chksvc.GetFormSchema(typ)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_type", err.Error())
		return
	}
	defaults := chksvc.DefaultFlatValues(schema)
	template := chksvc.DefaultPayloadTemplate(typ)
	var templateObj any
	_ = json.Unmarshal([]byte(template), &templateObj)
	WriteJSON(w, http.StatusOK, map[string]any{
		"schema":           schema,
		"default_values":   defaults,
		"payload_template": templateObj,
	})
}

func (s *Server) handleDoctor(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "all"
	}
	var issues []DoctorIssueJSON
	ok := true

	runScope := func(name string, fn func() error) {
		if scope != "all" && scope != name {
			return
		}
		if err := fn(); err != nil {
			ok = false
		}
	}

	runScope("library", func() error {
		ver, err := sqlstore.SchemaVersion(db)
		if err != nil {
			issues = append(issues, toDoctorIssueFromText(err.Error()))
			return err
		}
		missing, err := sqlstore.VerifyTables(db)
		if err != nil {
			issues = append(issues, toDoctorIssueFromText(err.Error()))
			return err
		}
		if len(missing) > 0 {
			issues = append(issues, toDoctorIssueFromText("缺少表: "+strings.Join(missing, ", ")))
			return errMissingTables(missing)
		}
		libIssues := doctor.CheckLibrary(db)
		if len(libIssues) > 0 {
			for _, s := range libIssues {
				issues = append(issues, toDoctorIssueFromText(s))
			}
			return errDoctor("library", libIssues)
		}
		_ = ver
		return nil
	})

	runScope("portfolio", func() error {
		p, err := yamlstore.LoadPortfolio(ac.PortfolioPath())
		if err != nil {
			issues = append(issues, toDoctorIssueFromText(err.Error()))
			return err
		}
		portIssues := doctor.CheckPortfolio(db, p)
		if len(portIssues) > 0 {
			for _, iss := range portIssues {
				issues = append(issues, toDoctorIssueJSON(iss))
			}
			return errDoctorFormatted(portIssues)
		}
		return nil
	})

	runScope("watchlist", func() error {
		wl, err := yamlstore.LoadWatchlist(ac.WatchlistPath())
		if err != nil {
			issues = append(issues, toDoctorIssueFromText(err.Error()))
			return err
		}
		p, err := yamlstore.LoadPortfolio(ac.PortfolioPath())
		if err != nil {
			issues = append(issues, toDoctorIssueFromText(err.Error()))
			return err
		}
		wlIssues := doctor.CheckWatchlist(db, wl, p)
		if len(wlIssues) > 0 {
			for _, s := range wlIssues {
				issues = append(issues, toDoctorIssueFromText(s))
			}
			return errDoctor("watchlist", wlIssues)
		}
		return nil
	})

	var repairActions []doctor.RepairAction
	if scope == "all" || scope == "portfolio" {
		if p, loadErr := yamlstore.LoadPortfolio(ac.PortfolioPath()); loadErr == nil {
			repairActions = doctor.BuildPortfolioRepairPlan(db, p)
		}
	}

	status := http.StatusOK
	if !ok {
		log.Printf("[api] doctor failed scope=%s issue_count=%d", scope, len(issues))
	}
	WriteJSON(w, status, map[string]any{
		"ok":             ok,
		"scope":          scope,
		"issues":         issues,
		"repair_actions": repairActions,
	})
}

func errMissingTables(missing []string) error {
	return &doctorErr{msg: "缺少表: " + strings.Join(missing, ", ")}
}

func errDoctor(scope string, items []string) error {
	return &doctorErr{msg: scope + " 校验失败: " + strings.Join(items, "; ")}
}

func errDoctorFormatted(items []doctor.Issue) error {
	return &doctorErr{msg: doctor.FormatIssues(items)}
}

type doctorErr struct{ msg string }

func (e *doctorErr) Error() string { return e.msg }
