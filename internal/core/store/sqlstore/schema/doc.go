// Package schema 定义 SQLite 表结构与 Go 字段映射（03 §九 / §十A）。
//
// 维护约定（docs/06 Review 后强制执行）：
//   - 每张表一个 Go struct，字段注释为中文业务含义 + 可选值；
//   - 与 migrations/*.up.sql 列名通过 db tag 对齐；
//   - 用户向字段说明见 docs/manual/ref-sqlite-*.md；
//   - 改表须同时改：SQL migration、本 package、manual 字段页。
//
// 本 package 不含 CRUD 逻辑；读写见 sqlstore 各 repository（H4+）。
package schema
