package library

import (
	"path/filepath"
	"testing"

	"github.com/investment-assistant/investment-assistant/internal/core/account"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
)

func TestIngestPromoteFlow(t *testing.T) {
	t.Setenv("IA_CONFIG_ROOT", filepath.Join("..", "..", "..", "config"))
	root := t.TempDir()
	ac, err := account.WithAccount(root, "default")
	if err != nil {
		t.Fatal(err)
	}
	if err := ac.EnsureInitialized(); err != nil {
		t.Fatal(err)
	}

	db, err := sqlstore.Open(ac.DBPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := sqlstore.MigrateUp(db); err != nil {
		t.Fatal(err)
	}

	svc, err := NewService(ac, db)
	if err != nil {
		t.Fatal(err)
	}

	res, err := svc.Ingest(IngestInput{
		Text:  "测试笔记正文",
		Title: "调研备忘",
		Tier:  "B",
		Stocks: []string{"600519"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "pending" {
		t.Fatalf("status=%s", res.Status)
	}

	libID, err := svc.Promote(PromoteInput{
		CandidateID: res.CandidateID,
		ContentType: "note",
		MediaType:   "text",
		Tier:        "B",
		Tags:        []string{"event_earnings"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if libID == "" {
		t.Fatal("empty lib id")
	}

	item, err := sqlstore.GetLibraryItem(db, libID)
	if err != nil {
		t.Fatal(err)
	}
	if item.Status != "active" {
		t.Fatalf("item status=%s", item.Status)
	}
	n, err := sqlstore.CountPrimaryAssets(db, libID)
	if err != nil || n != 1 {
		t.Fatalf("primary assets=%d err=%v", n, err)
	}
}

func TestExactDedupAutoDismiss(t *testing.T) {
	t.Setenv("IA_CONFIG_ROOT", filepath.Join("..", "..", "..", "config"))
	root := t.TempDir()
	ac, err := account.WithAccount(root, "default")
	if err != nil {
		t.Fatal(err)
	}
	if err := ac.EnsureInitialized(); err != nil {
		t.Fatal(err)
	}
	db, err := sqlstore.Open(ac.DBPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := sqlstore.MigrateUp(db); err != nil {
		t.Fatal(err)
	}
	svc, err := NewService(ac, db)
	if err != nil {
		t.Fatal(err)
	}

	url := "https://example.com/report"
	r1, err := svc.Ingest(IngestInput{URL: url, Title: "报告", AutoDismiss: true})
	if err != nil {
		t.Fatal(err)
	}
	libID, err := svc.Promote(PromoteInput{
		CandidateID: r1.CandidateID,
		ContentType: "report",
		MediaType:   "html",
		Tier:        "A",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = libID

	r2, err := svc.Ingest(IngestInput{URL: url, Title: "报告", AutoDismiss: true})
	if err != nil {
		t.Fatal(err)
	}
	if r2.Status != "dismissed" {
		t.Fatalf("expected dismissed, got %s", r2.Status)
	}
}

func TestTagsValidate(t *testing.T) {
	tags := &yamlstore.ControlledTags{
		Rules: yamlstore.ControlledRules{MaxTagsPerItem: 12},
		System: []yamlstore.ControlledTag{
			{ID: "event_earnings", Label: "财报", Dimension: "event"},
		},
	}
	if err := tags.ValidateTagIDs([]string{"event_earnings"}); err != nil {
		t.Fatal(err)
	}
	if err := tags.ValidateTagIDs([]string{"not_exists"}); err == nil {
		t.Fatal("expected error")
	}
}
