package api

import (
	"net/http"

	"github.com/investment-assistant/investment-assistant/internal/core/doctor"
	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
)

type doctorRepairRequest struct {
	Actions []doctor.RepairApply `json:"actions"`
}

// handleDoctorRepair 应用用户确认的 portfolio 修复并写回 YAML。
func (s *Server) handleDoctorRepair(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	var req doctorRepairRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if len(req.Actions) == 0 {
		WriteError(w, http.StatusBadRequest, "missing_actions", "须至少选择一项修复")
		return
	}

	p, err := yamlstore.LoadPortfolio(ac.PortfolioPath())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "read_error", err.Error())
		return
	}
	plan := doctor.BuildPortfolioRepairPlan(db, p)
	fixed, err := doctor.ApplyPortfolioRepairs(p, plan, req.Actions)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "apply_failed", err.Error())
		return
	}
	if err := yamlstore.SavePortfolio(ac.PortfolioPath(), fixed); err != nil {
		WriteError(w, http.StatusInternalServerError, "save_error", err.Error())
		return
	}

	issues := doctor.CheckPortfolio(db, fixed)
	repairActions := doctor.BuildPortfolioRepairPlan(db, fixed)
	resp := map[string]any{
		"ok":             len(issues) == 0,
		"issues":         toDoctorIssuesJSON(issues),
		"repair_actions": repairActions,
		"message":        "portfolio.yaml 已更新",
	}
	if len(issues) > 0 {
		resp["message"] = "部分问题已修复，仍有未处理项"
	}
	WriteJSON(w, http.StatusOK, resp)
}

func toDoctorIssuesJSON(issues []doctor.Issue) []DoctorIssueJSON {
	out := make([]DoctorIssueJSON, 0, len(issues))
	for _, iss := range issues {
		out = append(out, toDoctorIssueJSON(iss))
	}
	return out
}
