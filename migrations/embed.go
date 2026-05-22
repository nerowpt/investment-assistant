// Package migrations 嵌入 SQL 迁移文件，供 inv 在无 migrate CLI 时应用 schema。
package migrations

import "embed"

// UpFS 包含所有 *.up.sql 迁移（当前仅 001_initial）。
//
//go:embed *.up.sql
var UpFS embed.FS
