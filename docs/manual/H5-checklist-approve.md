# H5 - Checklist approve 流水线

> 决议：D5 / D14 / D19（[06 §四](../06-架构Review决议.md)）  
> 代码状态：✅  
> 最后验证：2026-05-27

---

## 一句话

**submitted** 的 Checklist 经 `inv checklist approve` → 写入 SQLite（journal / lot / snapshot）+ 同步 Layer A YAML（portfolio / watchlist）；sell 的 approve 在 **H6**。

> 📖 **架构权威**：[04 §二十](../04-技术架构.md) ApproveChecklist 流水线  
> 📄 **前置**：[H4-checklist与M7.md](H4-checklist与M7.md)（须先 draft → submit）  
> 📄 **流程总览**：[01-操作流程总览.md](01-操作流程总览.md)  
> 📄 **buy 示例**：[examples/checklist-buy-600519.json](examples/checklist-buy-600519.json)

---

## 记账字段（手动填写）

本系统**不联动券商**，无法自动获取实际成交价。approve 时：

| 类型 | 必填字段 | 用途 |
|---|---|---|
| buy | `execution_price`（元/股）、`shares`（股数） | 写入 `lots.cost_basis`、`lots.shares` |
| add | 同上 | 新 lot + portfolio 加权成本 |
| sell | `execution_price`、`sell_shares` | FIFO 分配 + `lot_allocations` 盈亏金额 |

`position_size_plan.initial_pct` / `add_pct` 仍用于 **M7 仓位风控**；`data_snapshots` 中的 worker 行情仅为决策时点**参考**，**不参与** cost_basis 计算。

---

## 使用场景

| 场景 | 怎么做 | 产物 |
|---|---|---|
| 建仓决策落库 | buy submit 后 `checklist approve` | `journals` + `lots` + `data_snapshots` + `portfolio.yaml` 新 holding |
| 加仓 | add submit → approve | 新 lot + portfolio.position_pct 累加 |
| 观察池 | watch submit → approve | `watchlist.yaml` 新 item（`w_*`） |
| 巡检 / 复盘 | inspect / review submit → approve | `inspection_records` / `review_reports` |
| 存量导入 | import submit → approve | 多条 journal/lot + portfolio positions（可带 `legacy_flags`） |
| 冻结决策时点行情 | approve 时自动调 worker | `data_snapshots.snapshot_json` 含 quote/valuation + tier |
| YAML 写失败 | SQL **不回滚** | `sync_repairs` 待修复队列 + doctor 报错 |

---

## 前置条件

- [x] H0–H4 已完成
- [ ] 已编译 CLI：`go build -o bin/inv.exe ./cmd/inv`
- [ ] 环境变量（PowerShell）：

```powershell
cd C:\Users\qs\Desktop\workspace\investment-assistant
$env:DATA_ROOT = ".\data"
$env:IA_ACCOUNT_ID = "default"
$env:IA_CONFIG_ROOT = ".\config"
```

- [ ] 至少一条 **status=submitted** 的 checklist（见 H4 步骤）
- [ ] approve buy/add 时须已在 payload 填写 **execution_price + shares**（见上节）；worker 可选，仅写入 snapshot 参考行情

```powershell
.\bin\inv.exe worker health
```

---

## 数据文件与 ID 约定

| 项 | 路径 / 格式 |
|---|---|
| Checklist 状态 | SQLite `checklist_submissions.status` → `approved` |
| Journal id | `j_{YYYYMMDD}_{seq}` |
| Lot id | `lot_{YYYYMMDD}_{seq}` |
| Snapshot id | `snap_{YYYYMMDD}_{seq}` |
| Inspection id | `insp_{YYYYMMDD}_{seq}` |
| Review id | `rev_{YYYYMMDD}_{seq}` |
| Watch item id | `w_{YYYYMMDD}_{seq}` |
| Sync repair id | `sr_{YYYYMMDD}_{seq}`（YAML 写失败时） |
| portfolio SoT | `data/accounts/default/state/portfolio.yaml` |
| watchlist SoT | `data/accounts/default/state/watchlist.yaml` |

**原则**：approve 后 checklist payload **不可改**；修正须新建 submission。

---

## 接口 1：`inv checklist approve`

### 签名

```text
inv checklist approve <cs_id> [--json]
```

### 入参

| 参数 | 必填 | 说明 |
|---|---|---|
| `cs_id` | 是 | 如 `cs_20260527_001` |
| `--json` | 否 | 机器可读输出 |

### 状态机要求

| 条件 | 说明 |
|---|---|
| `status=submitted` | draft 不能直接 approve |
| `submitted_by=user` | 须为用户提交 |
| M7 `approve_blocked=true` | 须 submit 时已填完整 `exception_json`，否则拒绝 |

### 成功 stdout（buy 示例）

```text
approve OK: checklist=cs_20260527_001 status=approved
  journal=j_20260527_001 lot=lot_20260527_001 snapshot=snap_20260527_001
  yaml: portfolio/watchlist 已同步
```

### `--json` 输出样例

```json
{
  "ChecklistID": "cs_20260527_001",
  "JournalID": "j_20260527_001",
  "LotID": "lot_20260527_001",
  "SnapshotID": "snap_20260527_001",
  "InspectionID": "",
  "ReviewID": "",
  "WatchID": "",
  "YAMLSynced": true,
  "SyncRepairID": ""
}
```

### 失败样例

**非 submitted：**

```text
Error: 仅 submitted 可 approve（当前 status=draft）
```

**M7 门禁（无 exception）：**

```text
Error: approve 被 M7 门禁拦截：须先在 submit 时提供 exception_json
hard_block:
  - [r004] personal_redlines: ...
```

**重复建仓：**

```text
Error: 600519 已在 holding，应走 add checklist
```

> approve **失败时** checklist 保持 `submitted`，不会写入 journal/lot（业务校验在 SQL commit 之前）。

---

## approve 各类型行为

| checklist_type | SQLite | YAML |
|---|---|---|
| `buy` | journal + lot + data_snapshot | portfolio 新增 holding |
| `add` | journal + lot + data_snapshot | portfolio.position_pct += add_pct |
| `watch` | checklist 状态更新 | watchlist 新增 item |
| `inspect` | inspection_records | 无 |
| `review` | review_reports | 无 |
| `import` | 多 journal/lot | portfolio 批量 positions |
| `sell` | journal + allocations + snapshot | portfolio ↓pct 或 closed（**H6**） |

---

## 手动验证步骤（buy 端到端）

### 步骤 1：draft → submit（沿用 H4）

```powershell
.\bin\inv.exe checklist draft --type buy --code 600519 --name 贵州茅台 --file docs\manual\examples\checklist-buy-600519.json
# 记下返回的 cs_id，例如 cs_20260527_001

.\bin\inv.exe checklist submit cs_20260527_001
# 预期：approve_blocked=false
```

### 步骤 2：approve

```powershell
.\bin\inv.exe worker health
.\bin\inv.exe checklist approve cs_20260527_001
```

### 步骤 3：SQLite 交叉验证

```powershell
sqlite3 data\accounts\default\db\assistant.sqlite "SELECT id, status, generated_journal_id FROM checklist_submissions WHERE id='cs_20260527_001';"
```

```text
cs_20260527_001|approved|j_20260527_001
```

```powershell
sqlite3 data\accounts\default\db\assistant.sqlite "SELECT id, action_type, code, lot_id FROM journals WHERE checklist_submission_id='cs_20260527_001';"
sqlite3 data\accounts\default\db\assistant.sqlite "SELECT id, code, initial_pct, status FROM lots WHERE journal_id='j_20260527_001';"
sqlite3 data\accounts\default\db\assistant.sqlite "SELECT id, substr(snapshot_json,1,120) FROM data_snapshots WHERE journal_id='j_20260527_001';"
```

snapshot_json 应含 `"quote"` / `"valuation"` 块（worker 可用时）及 `"tier"` 字段。

### 步骤 4：portfolio.yaml 验证

```powershell
Select-String -Path data\accounts\default\state\portfolio.yaml -Pattern "600519"
```

应看到 `state: holding`、`lot_ids` / `journal_ids` 与步骤 3 一致。

### 步骤 5：doctor

```powershell
.\bin\inv.exe doctor --scope portfolio
```

> 若 portfolio 仍含 H1 示例持仓 `002624`，doctor 可能因 lot_ids 与 SQLite 不一致而报错——验收 buy approve 时可临时清空 positions 或只验证新增 600519 行字段。

### 步骤 6：幂等

```powershell
.\bin\inv.exe checklist approve cs_20260527_001
# 预期：成功返回同一 journal_id，不重复 INSERT
```

---

## import + legacy_over_limit（S0.5 路径）

import payload 示例（保存为 `import-positions.json`）：

```json
{
  "import_reason": "存量迁移",
  "positions": [
    {
      "code": "000001",
      "name": "平安银行",
      "position_type": "swing",
      "position_pct": 12,
      "cost_basis": 10.5,
      "import_thesis_summary": "存量导入占位 thesis",
      "legacy_flags": ["legacy_over_limit"]
    }
  ],
  "emotion_retrospect": null
}
```

```powershell
.\bin\inv.exe checklist draft --type import --file import-positions.json
.\bin\inv.exe checklist submit cs_YYYYMMDD_NNN
.\bin\inv.exe checklist approve cs_YYYYMMDD_NNN
```

验证 portfolio 中 `legacy_flags` 含 `legacy_over_limit`。

---

## 自动化验证

```powershell
go test ./internal/core/checklist/... -v
```

覆盖：

- `TestApproveBuyCreatesJournalLotPortfolio` — buy approve → journal/lot/portfolio 一致 + 幂等
- `TestApproveBlockedWithoutException` — M7 门禁

---

## 故障排查

| 现象 | 处理 |
|---|---|
| `worker 未就绪` | 先 `inv worker health`；snapshot 仍可写入但缺 quote |
| `已在 holding，应走 add` | reject 误建的 buy → 新建 `draft --type add`（见 [checklist-add-600519.json](examples/checklist-add-600519.json)） |
| `sync_repair=sr_*` | SQL 已提交；检查 portfolio 路径权限，修复 YAML 后重跑 doctor |
| approve 后 doctor lot_ids 不一致 | 勿手改 portfolio.lot_ids；以 approve 输出为准 |
| `sell approve 在 H6 实现` | 正常，卖出流水线下一里程碑 |

---

## 相关文档

- [H4-checklist与M7.md](H4-checklist与M7.md) — draft/submit/M7
- [H3-data-worker-gRPC.md](H3-data-worker-gRPC.md) — snapshot 行情来源
- [ref-sqlite-decision-tables.md](ref-sqlite-decision-tables.md) — journals / lots / snapshots 表
- [ref-portfolio-yaml-fields.md](ref-portfolio-yaml-fields.md) — approve 后 portfolio 字段
