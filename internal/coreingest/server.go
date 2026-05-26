// Package coreingest 实现 Go 侧 CoreIngest gRPC 服务（04 §21.4）。
package coreingest

import (
	"context"
	"fmt"
	"net"
	"sync"

	coreingestv1 "github.com/investment-assistant/investment-assistant/gen/go/coreingest/v1"
	"github.com/investment-assistant/investment-assistant/internal/core/account"
	"github.com/investment-assistant/investment-assistant/internal/core/library"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"google.golang.org/grpc"
)

// Server CoreIngest gRPC 服务。
type Server struct {
	dataRoot string
	grpc     *grpc.Server
	lis      net.Listener
	mu       sync.Mutex
	running  bool
}

// NewServer 构造 CoreIngest 服务。
func NewServer(dataRoot string) *Server {
	return &Server{dataRoot: dataRoot}
}

// Start 监听 addr 并 serve。
func (s *Server) Start(addr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return nil
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("CoreIngest listen %s: %w", addr, err)
	}
	srv := grpc.NewServer()
	coreingestv1.RegisterCoreIngestServer(srv, &handler{dataRoot: s.dataRoot})
	s.grpc = srv
	s.lis = lis
	s.running = true
	go func() {
		_ = srv.Serve(lis)
	}()
	return nil
}

// Running 是否已启动。
func (s *Server) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// Stop 停止 gRPC 服务。
func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.grpc != nil {
		s.grpc.GracefulStop()
	}
	s.running = false
}

type handler struct {
	coreingestv1.UnimplementedCoreIngestServer
	dataRoot string
}

func (h *handler) StageCandidate(ctx context.Context, req *coreingestv1.StageCandidateRequest) (*coreingestv1.StageCandidateResponse, error) {
	return h.stageOne(ctx, req.GetAccountId(), req.GetDraft())
}

func (h *handler) StageCandidateBatch(ctx context.Context, req *coreingestv1.StageCandidateBatchRequest) (*coreingestv1.StageCandidateBatchResponse, error) {
	var results []*coreingestv1.StageCandidateBatchResult
	for i, draft := range req.GetDrafts() {
		res, err := h.stageOne(ctx, req.GetAccountId(), draft)
		if err != nil {
			res = &coreingestv1.StageCandidateResponse{ErrorMessage: err.Error()}
		}
		results = append(results, &coreingestv1.StageCandidateBatchResult{
			Index:  int32(i),
			Result: res,
		})
	}
	return &coreingestv1.StageCandidateBatchResponse{Results: results}, nil
}

func (h *handler) stageOne(ctx context.Context, accountID string, draft *coreingestv1.CandidateDraft) (*coreingestv1.StageCandidateResponse, error) {
	_ = ctx
	if draft == nil {
		return nil, fmt.Errorf("draft 不能为空")
	}
	if accountID == "" {
		accountID = "default"
	}
	ac, err := account.WithAccount(h.dataRoot, accountID)
	if err != nil {
		return nil, err
	}
	if err := ac.EnsureInitialized(); err != nil {
		return nil, err
	}
	db, err := sqlstore.Open(ac.DBPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := sqlstore.MigrateUp(db); err != nil {
		return nil, err
	}
	svc, err := library.NewService(ac, db)
	if err != nil {
		return nil, err
	}

	tier := "B"
	if p := draft.GetProvenance(); p != nil && p.GetTier() != "" {
		tier = p.GetTier()
	}
	source := "crawl"
	if draft.GetSourceEntry() != "" {
		source = draft.GetSourceEntry()
	}
	if p := draft.GetProvenance(); p != nil && p.GetSource() != "" {
		source = p.GetSource()
	}

	res, err := svc.StageFromDraft(library.DraftInput{
		SourceEntry:  draft.GetSourceEntry(),
		Title:        draft.GetTitle(),
		ContentType:  draft.GetContentType(),
		MediaType:    draft.GetMediaType(),
		Stocks:       draft.GetRelatedStocksJson(),
		CanonicalURL: draft.GetCanonicalUrl(),
		SummaryDraft: draft.GetSummaryDraft(),
		ExtractJSON:  draft.GetExtractJson(),
		Tier:         tier,
		Source:       source,
		AutoDismiss:  true,
	})
	if err != nil {
		return &coreingestv1.StageCandidateResponse{ErrorMessage: err.Error()}, nil
	}
	return &coreingestv1.StageCandidateResponse{
		CandidateId:        res.CandidateID,
		MatchTier:          res.MatchTier,
		DedupKey:           res.DedupKey,
		DuplicateDismissed: res.Status == "dismissed" && res.AutoAction == "auto_dismiss_exact",
	}, nil
}
