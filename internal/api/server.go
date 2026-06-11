package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/investment-assistant/investment-assistant/internal/core/account"
)

// Server HTTP API 服务（H8）。
type Server struct {
	baseAccount *account.Context
	addr        string
	dbMu        sync.Mutex
	dbs         map[string]*sql.DB // key: DBPath，进程内复用
}

// Options 启动参数。
type Options struct {
	Addr string // 监听地址，默认 :8787
}

// NewServer 构造 API server。
func NewServer(base *account.Context, opt Options) (*Server, error) {
	if base == nil {
		var err error
		base, err = account.ResolveFromEnv()
		if err != nil {
			return nil, err
		}
	}
	addr := opt.Addr
	if addr == "" {
		addr = ":8787"
	}
	return &Server{baseAccount: base, addr: addr}, nil
}

// Handler 返回 http.Handler。
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Account-Id"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(AuthMiddleware)
	r.Use(WithAccountMiddleware(s.baseAccount))

	r.Get("/api/health", s.handleHealth)
	r.Get("/api/portfolio", s.handlePortfolio)
	r.Get("/api/watchlist", s.handleWatchlist)
	r.Get("/api/pool", s.handlePool)
	r.Get("/api/pool/buy-context", s.handlePoolBuyContext)
	r.Get("/api/review/workbench", s.handleReviewWorkbench)
	r.Get("/api/review/lot-context", s.handleLotReviewContext)
	r.Get("/api/research/{code}", s.handleResearchDossier)
	r.Post("/api/research/{code}/fetch", s.handleResearchFetch)
	r.Post("/api/research/{code}/library", s.handleResearchSaveLibrary)
	r.Get("/api/journals", s.handleJournals)
	r.Get("/api/journals/{id}", s.handleJournalByID)
	r.Get("/api/checklists", s.handleChecklists)
	r.Get("/api/checklists/{id}", s.handleChecklistByID)
	r.Get("/api/checklist/schema", s.handleChecklistSchema)
	r.Get("/api/doctor", s.handleDoctor)
	r.Post("/api/doctor/repair", s.handleDoctorRepair)
	r.Get("/api/risk/rules", s.handleRiskRules)
	r.Post("/api/risk/check", s.handleRiskCheck)
	r.Get("/api/library", s.handleLibraryList)
	r.Get("/api/library/{id}", s.handleLibraryByID)
	r.Post("/api/library/quick-add", s.handleLibraryQuickAdd)

	r.Post("/api/checklist", s.handleChecklistCreate)
	r.Put("/api/checklist/{id}", s.handleChecklistUpdate)
	r.Post("/api/checklist/{id}/preview", s.handleChecklistPreview)
	r.Post("/api/checklist/{id}/submit", s.handleChecklistSubmit)
	r.Post("/api/checklist/{id}/plan", s.handleChecklistPlan)
	r.Post("/api/checklist/{id}/approve", s.handleChecklistApprove)
	r.Post("/api/checklist/{id}/reject", s.handleChecklistReject)

	return r
}

// Run 启动 HTTP 服务并阻塞。
func (s *Server) Run() error {
	fmt.Printf("ia-api listening on %s (account=%s data_root=%s)\n",
		s.addr, s.baseAccount.AccountID, s.baseAccount.DataRoot)
	return http.ListenAndServe(s.addr, s.Handler())
}
