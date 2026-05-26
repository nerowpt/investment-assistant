package ids

import (
	"path/filepath"
	"testing"

	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
)

func TestNextSequence(t *testing.T) {
	dir := t.TempDir()
	db, err := sqlstore.Open(filepath.Join(dir, "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := sqlstore.MigrateUp(db); err != nil {
		t.Fatal(err)
	}

	id1, err := Next(db, "lc")
	if err != nil {
		t.Fatal(err)
	}
	id2, err := Next(db, "lc")
	if err != nil {
		t.Fatal(err)
	}
	if id1 == id2 {
		t.Fatalf("expected distinct ids: %s %s", id1, id2)
	}
}
