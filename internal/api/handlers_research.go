package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	libsvc "github.com/investment-assistant/investment-assistant/internal/core/library"
	"github.com/investment-assistant/investment-assistant/internal/core/query"
)

type researchFetchRequest struct {
	Pack string `json:"pack"`
}

type researchSaveLibraryRequest struct {
	Title string `json:"title"`
	Text  string `json:"text"`
	Tier  string `json:"tier"`
}

func (s *Server) handleResearchDossier(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	code := strings.TrimSpace(chi.URLParam(r, "code"))
	if code == "" {
		WriteError(w, http.StatusBadRequest, "missing_code", "code 必填")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	data, err := newDeps(ac).reader(db).BuildResearchDossier(ctx, newDeps(ac).worker(), code)
	if err != nil {
		if err.Error() == "code 必填" {
			WriteError(w, http.StatusBadRequest, "missing_code", err.Error())
			return
		}
		WriteError(w, http.StatusInternalServerError, "read_error", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, data)
}

func (s *Server) handleResearchFetch(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	code := strings.TrimSpace(chi.URLParam(r, "code"))
	if code == "" {
		WriteError(w, http.StatusBadRequest, "missing_code", "code 必填")
		return
	}
	var req researchFetchRequest
	pack := strings.TrimSpace(r.URL.Query().Get("pack"))
	if r.Method == http.MethodPost {
		if err := DecodeJSON(r, &req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		if req.Pack != "" {
			pack = req.Pack
		}
	}
	if pack == "" {
		WriteError(w, http.StatusBadRequest, "missing_pack", "须指定 pack")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	result, err := query.FetchResearchPack(ctx, newDeps(ac).worker(), code, pack)
	if err != nil {
		WriteError(w, http.StatusBadGateway, "fetch_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, result)
}

func (s *Server) handleResearchSaveLibrary(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	code := strings.TrimSpace(chi.URLParam(r, "code"))
	if code == "" {
		WriteError(w, http.StatusBadRequest, "missing_code", "code 必填")
		return
	}
	var req researchSaveLibraryRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.Title == "" || req.Text == "" {
		WriteError(w, http.StatusBadRequest, "missing_field", "title、text 必填")
		return
	}
	svc, err := libsvc.NewService(ac, db)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "service_error", err.Error())
		return
	}
	libID, err := svc.QuickAdd(libsvc.QuickAddInput{
		Title: req.Title,
		Text:  req.Text,
		Stock: code,
		Tier:  req.Tier,
	})
	if err != nil {
		WriteError(w, http.StatusBadRequest, "save_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusCreated, map[string]any{
		"library_id": libID,
		"title":      req.Title,
		"code":       code,
	})
}
