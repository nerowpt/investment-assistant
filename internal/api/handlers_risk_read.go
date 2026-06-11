package api

import (
	"net/http"

	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
)

// LimitPairJSON M7 双阈值 JSON。
type LimitPairJSON struct {
	WarningPct   float64 `json:"warning_pct"`
	HardBlockPct float64 `json:"hard_block_pct"`
}

// RiskRulesJSON 风控规则 JSON（H8 前端展示）。
type RiskRulesJSON struct {
	PositionLimits struct {
		SingleStock  LimitPairJSON `json:"single_stock"`
		SingleSector LimitPairJSON `json:"single_sector"`
		TotalEquity  LimitPairJSON `json:"total_equity"`
		SingleThesis LimitPairJSON `json:"single_thesis"`
	} `json:"position_limits"`
	ConfigPath string `json:"config_path"`
}

func toLimitPairJSON(p yamlstore.LimitPair) LimitPairJSON {
	return LimitPairJSON{WarningPct: p.WarningPct, HardBlockPct: p.HardBlockPct}
}

func (s *Server) handleRiskRules(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	rules, err := yamlstore.LoadRiskRules(ac.RiskRulesPath())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "read_error", err.Error())
		return
	}
	out := RiskRulesJSON{ConfigPath: ac.RiskRulesPath()}
	out.PositionLimits.SingleStock = toLimitPairJSON(rules.PositionLimits.SingleStock)
	out.PositionLimits.SingleSector = toLimitPairJSON(rules.PositionLimits.SingleSector)
	out.PositionLimits.TotalEquity = toLimitPairJSON(rules.PositionLimits.TotalEquity)
	out.PositionLimits.SingleThesis = toLimitPairJSON(rules.PositionLimits.SingleThesis)
	WriteJSON(w, http.StatusOK, out)
}
