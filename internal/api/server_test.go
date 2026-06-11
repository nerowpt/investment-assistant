package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/investment-assistant/investment-assistant/internal/core/account"
)

func TestHealthAndSchema(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DATA_ROOT", root)
	t.Setenv("IA_CONFIG_ROOT", filepath.Join("..", "..", "config"))
	t.Setenv("IA_ACCOUNT_ID", "api-test")

	ac, err := account.WithAccount(root, "api-test")
	if err != nil {
		t.Fatal(err)
	}
	if err := ac.EnsureInitialized(); err != nil {
		t.Fatal(err)
	}
	srv, err := NewServer(ac, Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { srv.closeDBs() })
	h := srv.Handler()

	t.Run("health", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("schema_buy", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/checklist/schema?type=buy", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var env Envelope
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatal(err)
		}
		if !env.Success || env.Data == nil {
			t.Fatalf("expected success envelope: %+v", env)
		}
	})

	t.Run("portfolio", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("concurrent_reads", func(t *testing.T) {
		paths := []string{"/api/portfolio", "/api/watchlist", "/api/doctor?scope=all"}
		done := make(chan error, len(paths))
		for _, p := range paths {
			go func(path string) {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				w := httptest.NewRecorder()
				h.ServeHTTP(w, req)
				if w.Code != http.StatusOK {
					done <- fmt.Errorf("%s status=%d body=%s", path, w.Code, w.Body.String())
					return
				}
				done <- nil
			}(p)
		}
		for range paths {
			if err := <-done; err != nil {
				t.Fatal(err)
			}
		}
	})
}
