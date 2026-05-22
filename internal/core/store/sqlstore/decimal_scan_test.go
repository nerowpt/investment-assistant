package sqlstore

import (
	"path/filepath"
	"testing"

	"github.com/shopspring/decimal"
)

// TestSumDecimalColumn_PrecisionBoundary 验证 0.1 + 0.2 边界（D11 要求）。
// 经典浮点用例：float64 下 0.1+0.2=0.30000000000000004；decimal 必须等于 0.3。
func TestSumDecimalColumn_PrecisionBoundary(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.sqlite")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE t (pct REAL NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO t (pct) VALUES (0.1), (0.2)`); err != nil {
		t.Fatal(err)
	}

	got, err := SumDecimalColumn(db, `SELECT CAST(pct AS TEXT) FROM t`)
	if err != nil {
		t.Fatal(err)
	}
	want := decimal.NewFromFloat(0.3)
	if !got.Equal(want) {
		t.Fatalf("0.1+0.2 期望 %s 实际 %s", want, got)
	}
}

// TestSumDecimalColumn_NullSkipped 验证 NULL 行视作 0。
func TestSumDecimalColumn_NullSkipped(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.sqlite")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE t (pct REAL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO t (pct) VALUES (1.5), (NULL), (2.5)`); err != nil {
		t.Fatal(err)
	}

	got, err := SumDecimalColumn(db, `SELECT CAST(pct AS TEXT) FROM t`)
	if err != nil {
		t.Fatal(err)
	}
	want := decimal.NewFromFloat(4.0)
	if !got.Equal(want) {
		t.Fatalf("期望 %s 实际 %s", want, got)
	}
}

// TestScanDecimalRow_Empty 验证空结果集（无行）。
func TestScanDecimalRow_Empty(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.sqlite")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE t (pct REAL)`); err != nil {
		t.Fatal(err)
	}

	got, err := SumDecimalColumn(db, `SELECT CAST(pct AS TEXT) FROM t WHERE 1=0`)
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsZero() {
		t.Fatalf("空结果集期望 0，实际 %s", got)
	}
}
