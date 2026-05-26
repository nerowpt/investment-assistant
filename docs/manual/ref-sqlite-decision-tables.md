# SQLite 决策流水表字段说明

> DDL 文件：[migrations/001_initial.up.sql](../../migrations/001_initial.up.sql)  
> Go 结构体（字段注释权威副本）：[internal/core/store/sqlstore/schema/](../../internal/core/store/sqlstore/schema/)  
> 设计文档：[03 §十A](../../docs/03-数据架构与数据源.md)

---

## 为什么有两份说明？

| 位置 | 给谁看 | 内容 |
|---|---|---|
| **本页 + manual** | 你（验收 / 手填 / 读库） | 中文含义、可选值、示例 |
| **`schema/*.go`** | 开发者 / IDE | 与表列一一对应的 struct + 注释 |
| **`001_initial.up.sql`** | 数据库 | SQLite 方言 DDL |

**推荐**：查字段含义优先看本页或 Go struct 注释；改表结构时 **同时** 改 SQL + Go struct + 本页。

---

## 表索引（MVP-1）

| 表 | 用途 | Go 类型 |
|---|---|---|
| `journals` | 决策日志（买/卖/导入…） | `schema.Journal` |
| `lots` | 仓位 lot 归因 | `schema.Lot` |
| `lot_allocations` | 卖出时 lot 匹配 | `schema.LotAllocation` |
| `checklist_submissions` | Checklist 提交记录 | `schema.ChecklistSubmission` |
| `data_snapshots` | 决策时市场快照（冻结） | `schema.DataSnapshot` |
| `risk_exceptions` | 风险护栏例外 | `schema.RiskException` |
| `inspection_records` | 巡检产物 | `schema.InspectionRecord` |
| `review_reports` | 复盘报告 | `schema.ReviewReport` |
| `id_sequences` | 日序 ID | `schema.IDSequence` |
| `sync_repairs` | YAML 写失败修复队列 | `schema.SyncRepair` |

L1 四表见 [ref-sqlite-library-tables.md](ref-sqlite-library-tables.md)（H2 补全）。

---

## `journals` 表

**一句话**：每笔通过 Checklist 的决策动作都会生成一条 **不可变** 日志。

数据库路径：`{DATA_ROOT}/accounts/{account_id}/db/assistant.sqlite`

### 字段说明

| 列名 | 必填 | 逻辑类型 | 怎么填 / 含义 |
|---|---|---|---|
| `id` | 是 | STRING_ID | 主键，格式 `j_{YYYYMMDD}_{seq}`，如 `j_20260519_001`（系统生成，勿自编规则外的 id） |
| `action_type` | 是 | STRING | 动作类型，见下表 |
| `code` | 条件 | STRING | 股票代码；`import` 批次级可空 |
| `name` | 否 | STRING | 标的名称 |
| `checklist_submission_id` | 条件 | STRING_ID | 来源 Checklist id，`cs_*`；`rule_change` 等可来自 review |
| `data_snapshot_id` | 条件 | STRING_ID | 冻结快照 id；buy/add/sell 应有；import 可简版 |
| `payload_json` | 是 | JSON | approve 时刻从 checklist **完整复制**的 JSON（权威字段） |
| `summary` | 否 | STRING | 一行摘要，列表展示用 |
| `lot_id` | 条件 | STRING_ID | buy/add/import **新建 lot** 时指向 `lot_*` |
| `created_at` | 是 | TIMESTAMP | 创建时间 ISO8601，**写入后不可改** |

### `action_type` 可选值

| 值 | 含义 | 是否新建 lot | 是否写 lot_allocations |
|---|---|---|---|
| `buy` | 建仓 | ✅ 新建 | — |
| `add` | 加仓 | ✅ 新建 | — |
| `sell` | 卖出 | — | ✅ |
| `import` | 存量导入 | ✅（每只标的） | — |
| `rule_change` | 规则变更（复盘改 redlines 等） | — | — |

### 手工插入示例（仅 doctor 测试用，生产走 Checklist）

```sql
INSERT INTO journals (
  id, action_type, code, name, payload_json, created_at
) VALUES (
  'j_20260522_001',
  'buy',
  '600519',
  '贵州茅台',
  '{"note":"doctor 测试"}',
  '2026-05-22T10:00:00+08:00'
);
```

### 与 portfolio.yaml 的关系

- `portfolio.yaml` 的 `journal_ids[]` **必须**能在这里查到对应 `id`
- 正常流程：**不要**手 INSERT；H5 起用 `inv checklist approve`

---

## `lots` 表

**一句话**：每次 buy/add/import 产生一个 **lot**，卖出时按 lot 做 FIFO 归因。

Go 定义：`schema.Lot`（IDE 打开 `internal/core/store/sqlstore/schema/lot.go` 可看每列注释）

### 字段说明

| 列名 | 必填 | 可选值 / 格式 | 含义 |
|---|---|---|---|
| `id` | 是 | `lot_{YYYYMMDD}_{seq}` | 主键 |
| `code` | 是 | 股票代码 | 与 portfolio.position.code 一致 |
| `journal_id` | 是 | `j_*` | 开启本 lot 的 journal |
| `action_type` | 是 | `buy` / `add` / `import` | 如何产生 |
| `position_type` | 是 | `core` / `swing` | 与 portfolio 一致 |
| `status` | 是 | `open` / `partial` / `closed` | 是否卖完 |
| `initial_pct` | 是 | 数字 | 产生时占总资产 % |
| `current_pct` | 是 | 数字 | 当前剩余 %（卖出递减） |
| `cost_basis` | 是 | 数字 | 该 lot 成本价 |
| `dividends_received` | 否 | 数字，默认 0 | 已收分红累计（D8 预留） |
| `adjusted_cost_basis` | 否 | 数字 | 前复权成本（D8 预留） |
| `corporate_actions_json` | 否 | JSON 数组 | 送转/拆股事件（D8 预留） |

### 与 portfolio 的 doctor 规则

- `portfolio.lot_ids` 每条必须在此表存在且 `code` 匹配
- `holding` 标的：`sum(open|partial 的 current_pct) == portfolio.position_pct`

---

## 常用查询（验收）

```sql
-- 列出最近 journal
SELECT id, action_type, code, created_at FROM journals ORDER BY created_at DESC LIMIT 10;

-- 检查 portfolio 引用的 journal 是否存在
SELECT id FROM journals WHERE id = 'j_20260518_001';
```

可用 DB 工具打开 `assistant.sqlite`，或安装 `sqlite3` CLI。

---

## 关联手册

- [ref-portfolio-yaml-fields.md](ref-portfolio-yaml-fields.md)
- [H1-portfolio与doctor.md](H1-portfolio与doctor.md)
