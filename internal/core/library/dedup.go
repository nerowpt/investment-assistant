// Package library 实现 L1 归纳流水线（03 §8.10–§9.9、§十C）。
package library

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
)

// DedupInput 计算 dedup_key 所需字段（03 §9.7）。
type DedupInput struct {
	FileSHA256   string // 文件 SHA256（不含前缀）
	CanonicalURL string // 规范化前的 URL
	Source       string
	Title        string
	Timestamp    string // ISO8601
	PrimaryStock string // related_stocks 首个标的
}

// ComputeDedupKey 按优先级生成 dedup_key。
func ComputeDedupKey(in DedupInput) string {
	if in.FileSHA256 != "" {
		return "sha256:" + strings.ToLower(in.FileSHA256)
	}
	if u := normalizeURL(in.CanonicalURL); u != "" {
		return "url:" + u
	}
	meta := strings.Join([]string{
		strings.TrimSpace(in.Source),
		strings.TrimSpace(in.Title),
		strings.TrimSpace(in.Timestamp),
		strings.TrimSpace(in.PrimaryStock),
	}, "|")
	sum := sha256.Sum256([]byte(meta))
	return "meta:" + hex.EncodeToString(sum[:16])
}

// normalizeURL 规范化 URL 用于去重。
func normalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return strings.ToLower(raw)
	}
	u.Fragment = ""
	u.RawFragment = ""
	host := strings.ToLower(u.Hostname())
	path := strings.TrimSuffix(u.EscapedPath(), "/")
	if path == "" {
		path = "/"
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme == "" {
		scheme = "https"
	}
	out := fmt.Sprintf("%s://%s%s", scheme, host, path)
	if u.RawQuery != "" {
		out += "?" + u.RawQuery
	}
	return out
}
