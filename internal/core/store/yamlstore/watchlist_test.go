package yamlstore

import (
	"context"
	"path/filepath"
	"testing"
)

func TestValidateWatchlist_promotedRequiresJournal(t *testing.T) {
	w := &Watchlist{
		SchemaVersion: 1,
		Meta:          WatchlistMeta{UpdatedAt: "2026-05-22T10:00:00+08:00"},
		Items: []WatchlistItem{{
			ID:                "w_20260518_001",
			Name:              "测试",
			WatchType:         "stock",
			State:             "removed",
			SourceEntry:       "manual",
			WatchReason:       "r",
			Hypothesis:        "h",
			KeyMetricsToWatch: []string{"m"},
			ExpectedTrigger:   "t",
			InvalidCondition:  "i",
			ReviewDate:        "2026-06-01",
			CreatedAt:         "2026-05-22T10:00:00+08:00",
			RemovedAt:         "2026-05-23T10:00:00+08:00",
			RemovedReason:     "promoted_to_holding",
		}},
	}
	issues := ValidateWatchlist(w)
	if len(issues) == 0 {
		t.Fatal("期望 promoted_to_holding 缺少 promoted_journal_id 报错")
	}
}

func TestLoadSaveWatchlist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "watchlist.yaml")
	w := &Watchlist{
		SchemaVersion: 1,
		Meta:          WatchlistMeta{UpdatedAt: "2026-05-22T10:00:00+08:00"},
		Items:         []WatchlistItem{},
	}
	if err := SaveWatchlist(path, w); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadWatchlist(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.SchemaVersion != 1 {
		t.Fatalf("schema_version=%d", loaded.SchemaVersion)
	}
}

func TestMemoryWatchlistStore_Isolation(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryWatchlistStore()
	path := "/mem/watchlist.yaml"
	w := &Watchlist{
		SchemaVersion: 1,
		Meta:          WatchlistMeta{UpdatedAt: "2026-05-22T10:00:00+08:00"},
		Items:         []WatchlistItem{},
	}
	if err := s.Save(ctx, path, w); err != nil {
		t.Fatal(err)
	}
	loaded, err := s.Load(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	loaded.Items = append(loaded.Items, WatchlistItem{ID: "w_1", Name: "x"})
	again, err := s.Load(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Items) != 0 {
		t.Fatal("Memory store 应隔离外部 mutation")
	}
}
