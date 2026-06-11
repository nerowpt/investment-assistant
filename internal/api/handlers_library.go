package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	libsvc "github.com/investment-assistant/investment-assistant/internal/core/library"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
)

// LibraryListItem L1 素材列表项（H8 前端选择器）。
type LibraryListItem struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Tier      string   `json:"tier"`
	Source    string   `json:"source"`
	Summary   string   `json:"summary,omitempty"`
	Stocks    []string `json:"stocks,omitempty"`
	CreatedAt string   `json:"created_at"`
}

type libraryQuickAddRequest struct {
	Title string `json:"title"`
	Text  string `json:"text"`
	Stock string `json:"stock"`
	Tier  string `json:"tier"`
}

func (s *Server) handleLibraryList(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	stock := r.URL.Query().Get("stock")
	if stock == "" {
		stock = r.URL.Query().Get("code")
	}
	rows, err := sqlstore.ListLibraryItems(db, "active", stock, "")
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "read_error", err.Error())
		return
	}
	items := make([]LibraryListItem, 0, len(rows))
	for _, row := range rows {
		var stocks []string
		_ = json.Unmarshal([]byte(row.RelatedStocksJSON), &stocks)
		items = append(items, LibraryListItem{
			ID:        row.ID,
			Title:     row.Title,
			Tier:      row.Tier,
			Source:    row.Source,
			Summary:   row.SummaryByUser,
			Stocks:    stocks,
			CreatedAt: row.CreatedAt,
		})
	}
	WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

// handleLibraryByID 按 id 返回单个 L1 素材（记录详情点击展示）。
func (s *Server) handleLibraryByID(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		WriteError(w, http.StatusBadRequest, "missing_id", "缺少素材 id")
		return
	}
	item, err := sqlstore.GetLibraryItem(db, id)
	if err != nil {
		if strings.Contains(err.Error(), "不存在") {
			WriteError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		WriteError(w, http.StatusInternalServerError, "read_error", err.Error())
		return
	}
	var stocks []string
	_ = json.Unmarshal([]byte(item.RelatedStocksJSON), &stocks)
	WriteJSON(w, http.StatusOK, map[string]any{
		"item": LibraryListItem{
			ID:        item.ID,
			Title:     item.Title,
			Tier:      item.Tier,
			Source:    item.Source,
			Summary:   item.SummaryByUser,
			Stocks:    stocks,
			CreatedAt: item.CreatedAt,
		},
	})
}

func (s *Server) handleLibraryQuickAdd(w http.ResponseWriter, r *http.Request) {
	ac := AccountFromContext(r.Context())
	db, err := s.getDB(ac)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	var req libraryQuickAddRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.Title == "" || req.Text == "" || req.Stock == "" {
		WriteError(w, http.StatusBadRequest, "missing_field", "title、text、stock 必填")
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
		Stock: req.Stock,
		Tier:  req.Tier,
	})
	if err != nil {
		WriteError(w, http.StatusBadRequest, "quick_add_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusCreated, map[string]any{
		"library_id": libID,
		"title":      req.Title,
	})
}
