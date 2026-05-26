package coreingest

import (
	"context"
	"path/filepath"
	"testing"

	coreingestv1 "github.com/investment-assistant/investment-assistant/gen/go/coreingest/v1"
	commonv1 "github.com/investment-assistant/investment-assistant/gen/go/common/v1"
)

func TestStageCandidate(t *testing.T) {
	t.Setenv("IA_CONFIG_ROOT", filepath.Join("..", "..", "config"))
	root := t.TempDir()
	srv := NewServer(root)
	if err := srv.Start("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	// 使用 handler 直接测（不经网络）
	h := &handler{dataRoot: root}
	res, err := h.stageOne(context.Background(), "default", &coreingestv1.CandidateDraft{
		Provenance:   &commonv1.Provenance{Source: "crawl", Tier: "B"},
		SourceEntry:  "crawl",
		Title:        "测试公告",
		ContentType:  "announcement",
		MediaType:    "text",
		SummaryDraft: "事实摘要",
		CanonicalUrl: "https://example.com/a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.GetCandidateId() == "" {
		t.Fatal("expected candidate id")
	}
	if res.GetErrorMessage() != "" {
		t.Fatalf("error: %s", res.GetErrorMessage())
	}
}
