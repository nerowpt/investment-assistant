package api

import (
	"context"
	"net/http"
	"strings"
	"time"
)

func (s *Server) handlePool(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	zone := strings.TrimSpace(r.URL.Query().Get("zone"))
	deps := newDeps(ac)
	reader := deps.reader(db)
	data, err := reader.BuildPool(zone)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "read_error", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	reader.EnrichPoolNames(ctx, deps.worker(), data)
	WriteJSON(w, http.StatusOK, data)
}

func (s *Server) handlePoolBuyContext(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	from := strings.TrimSpace(r.URL.Query().Get("from"))
	watchID := strings.TrimSpace(r.URL.Query().Get("watch_id"))
	if code == "" {
		WriteError(w, http.StatusBadRequest, "missing_code", "code 必填")
		return
	}
	deps := newDeps(ac)
	reader := deps.reader(db)
	data, err := reader.BuildBuyContext(code, from, watchID)
	if err != nil {
		if err.Error() == "code 必填" {
			WriteError(w, http.StatusBadRequest, "missing_code", err.Error())
			return
		}
		WriteError(w, http.StatusInternalServerError, "read_error", err.Error())
		return
	}
	if data.Name == "" || data.Name == code {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		data.Name = reader.ResolveStockName(ctx, deps.worker(), code)
	}
	WriteJSON(w, http.StatusOK, data)
}
