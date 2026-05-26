# schema 包说明

Go 结构体与 `migrations/001_initial.up.sql` 列一一对应，**每个字段有中文注释**（业务含义 + 可选值）。

## 用途

| 受众 | 怎么用 |
|---|---|
| 开发者 | 写 sqlstore repository 时对照 struct + `db` tag |
| 你（验收） | IDE 悬停看注释；或配合 [docs/manual/ref-sqlite-decision-tables.md](../../../docs/manual/ref-sqlite-decision-tables.md) |
| AI 协作者 | 改表必须同时改 SQL + 本目录 + manual |

## 文件与表

| 文件 | 表 |
|---|---|
| `journal.go` | `journals` |
| `lot.go` | `lots` |
| `lot_allocation.go` | `lot_allocations` |
| `checklist.go` | `checklist_submissions` |
| `snapshot.go` | `data_snapshots` |
| `risk_exception.go` | `risk_exceptions` |
| `inspection.go` | `inspection_records`, `review_reports` |
| `infra.go` | `id_sequences`, `sync_repairs`, `schema_meta`, `monitor_events` |
| `library.go` | `library_candidates`, `library_items`（assets/links 见 H2） |

## 维护规则

1. 新增列：改 `001` 或 `002_*.up.sql` → 改 struct → 改 `docs/manual/ref-*`。
2. DECIMAL 列在 struct 中用 `string` + 注释「逻辑 DECIMAL」，读写走 `decimal_scan.go`。
3. 不在此 package 写 SQL 查询（保持纯模型）。
