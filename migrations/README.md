# SQLite 迁移（golang-migrate）

对齐 [03-数据架构与数据源](../docs/03-数据架构与数据源.md) §九 / §十A 与 [04-技术架构](../docs/04-技术架构.md) §二十三、**§10.4**。

## 方言说明

**本目录下的 `*.sql` 是 SQLite 方言 DDL**，不是 ANSI SQL，也不能原样在 MySQL 上执行。

| 层 | 位置 | 职责 |
|---|---|---|
| 逻辑字段 | `docs/03` §十A | 语义、必填、业务规则 |
| 跨库映射 | `docs/04` §10.4 | 逻辑类型 → SQLite / MySQL / Postgres |
| 物理 DDL | `migrations/*.sql`（本目录） | **仅 SQLite** MVP-1 运行时 |
| 未来 MySQL | `migrations/mysql/`（尚未创建） | 平行版本号，同 `schema_version` |

字段注释 `-- logical: STRING_ID` 等见 `001_initial.up.sql` 文件头。

## 文件

| 文件 | 说明 |
|---|---|
| `001_initial.up.sql` | MVP-1 全表创建 |
| `001_initial.down.sql` | 回滚（开发用） |
| `embed.go` | 嵌入 up 文件供 `inv doctor` 使用 |

## 执行

```bash
# 需安装 migrate CLI：https://github.com/golang-migrate/migrate
make migrate-up ACCOUNT=default

# 或 inv 内置（H0+）
inv doctor --scope library
```

迁移目标路径：`$DATA_ROOT/accounts/{account_id}/db/assistant.sqlite`。

## 维护规则

- **v1**：Round 5 定稿；含 `id_sequences`、`sync_repairs` 及全部 MVP-1 业务表。
- 变更须新增 `002_*.up.sql`，**禁止**修改已发布 up 文件（H1 期间 001 微调已结束并冻结）。

## Go struct 映射

每张表对应 `internal/core/store/sqlstore/schema/*.go`，字段中文注释见 struct。  
用户向说明见 `docs/manual/ref-sqlite-decision-tables.md`。
- ⚠️ **H1 期间例外窗口**（截至 H1 完成、首次写入真实数据前）：
  尚未在任何 account 生产数据上跑过 buy/sell approve，因此 `001_initial.up.sql` **仍允许在
  H1 期间做兼容性微调**（如 docs/06 §D8 追加 lots 复权字段）。
  H5 buy approve 端到端跑通后，001 进入永久冻结，后续一律走 002+。

## DECIMAL / REAL

`lots`、`lot_allocations` 中比例与金额在 SQLite 用 `REAL` 存储；Go `domain`/`store` 须用 `shopspring/decimal` 读写，禁止 `float64` 相等比较。
读取时必须走 `internal/core/store/sqlstore/decimal_scan.go`（`CAST AS TEXT` + `decimal.NewFromString`）。
MySQL 迁移时映射为 `DECIMAL(18,6)`（见 04 §10.4.3、docs/06 §D11）。
