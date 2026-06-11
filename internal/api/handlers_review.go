package api

import (
	"context"
	"net/http"
	"strings"
	"time"
)

func (s *Server) handleReviewWorkbench(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		WriteError(w, http.StatusBadRequest, "missing_code", "code 必填")
		return
	}
	deps := newDeps(ac)
	reader := deps.reader(db)
	data, err := reader.BuildReviewWorkbench(code)
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

func (s *Server) handleLotReviewContext(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	lotID := strings.TrimSpace(r.URL.Query().Get("lot_id"))
	if code == "" || lotID == "" {
		WriteError(w, http.StatusBadRequest, "missing_param", "code 与 lot_id 必填")
		return
	}
	deps := newDeps(ac)
	reader := deps.reader(db)
	data, err := reader.BuildLotReviewContext(code, lotID)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "不存在") || strings.Contains(msg, "不匹配") || strings.Contains(msg, "仅已关闭") {
			WriteError(w, http.StatusBadRequest, "invalid_lot", msg)
			return
		}
		WriteError(w, http.StatusInternalServerError, "read_error", msg)
		return
	}
	if data.Name == "" || data.Name == code {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		data.Name = reader.ResolveStockName(ctx, deps.worker(), code)
	}
	WriteJSON(w, http.StatusOK, data)
}
