package worker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadWorkerAddr(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "worker.port")
	if err := os.WriteFile(path, []byte("127.0.0.1:50052\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	addr, err := readWorkerAddr(path)
	if err != nil || addr != "127.0.0.1:50052" {
		t.Fatalf("got %q err=%v", addr, err)
	}
}

func TestPickFreePort(t *testing.T) {
	port, err := pickFreePort()
	if err != nil || port <= 0 {
		t.Fatalf("port=%d err=%v", port, err)
	}
}

func TestFindRepoRoot(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := findRepoRoot(wd)
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("go.mod not found under %s", root)
	}
}
