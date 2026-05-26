package library

import "testing"

func TestComputeDedupKeyURL(t *testing.T) {
	key := ComputeDedupKey(DedupInput{CanonicalURL: "HTTPS://Example.com/path/"})
	if key != "url:https://example.com/path" {
		t.Fatalf("got %q", key)
	}
}

func TestComputeDedupKeyFile(t *testing.T) {
	key := ComputeDedupKey(DedupInput{FileSHA256: "ABC"})
	if key != "sha256:abc" {
		t.Fatalf("got %q", key)
	}
}

func TestComputeDedupKeyMeta(t *testing.T) {
	a := ComputeDedupKey(DedupInput{Source: "x", Title: "t", Timestamp: "2026", PrimaryStock: "600519"})
	b := ComputeDedupKey(DedupInput{Source: "x", Title: "t", Timestamp: "2026", PrimaryStock: "600519"})
	if a != b || !hasPrefix(a, "meta:") {
		t.Fatalf("meta key unstable: %q %q", a, b)
	}
}

func hasPrefix(s, p string) bool {
	return len(s) >= len(p) && s[:len(p)] == p
}
