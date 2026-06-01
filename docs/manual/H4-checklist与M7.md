# H4 - Checklist draft / submit + M7

> 决议：D5 / D14（[06 §四](../06-架构Review决议.md)）  
> 代码状态：✅  
> 最后验证：2026-05-26

---

## 一句话

填写 **Checklist**（建仓/加仓等决策表单）→ **submit** 时运行 **M7 风险护栏** → 写入 SQLite；**approve 落库见 H5**。

> 📖 **payload 字段权威**：[02 §十六](../02-核心场景与功能边界.md) · [03 §10A.3](../03-数据架构与数据源.md)  
> 📄 **buy 示例 JSON**：[examples/checklist-buy-600519.json](examples/checklist-buy-600519.json)  
> 📄 **hard_block 例外示例**：[examples/checklist-exception-hard.json](examples/checklist-exception-hard.json)  
> 📄 **risk 配置模板**：[config/risk_rules.yaml.example](../../config/risk_rules.yaml.example) · [config/personal_redlines.yaml.example](../../config/personal_redlines.yaml.example)

---

## 使用场景

| 场景 | 怎么做 | 产物 |
|---|---|---|
| 准备建仓，先填表单 | `checklist draft --type buy …` → `--file` 提交完整 JSON → `submit` | `checklist_submissions` status=submitted |
| 检查 M7 是否拦你 | `submit` 输出 `approve_blocked=true` | `risk_guardrail_result_json` + `risk_exceptions` |
| 引用 L1 素材建仓 | payload 填 `related_library_ids: ["lib_…"]`；C/D tier 须 `tier_acknowledgement: true` | 校验失败则 submit 报错 |
| **只想查现价** | 用 [H3 worker](H3-data-worker-gRPC.md) | 与 checklist 无自动联动 |
| **真正买入落库** | H5 `checklist approve` | journal + lot + portfolio.yaml |

---

## 前置条件

- [x] H0–H3 已完成
- [ ] 已在仓库根目录执行 `go build -o bin/inv.exe ./cmd/inv`
- [ ] 环境变量（PowerShell）：

```powershell
cd C:\Users\qs\Desktop\workspace\investment-assistant
$env:DATA_ROOT = ".\data"
$env:IA_ACCOUNT_ID = "default"
```

- [ ] 以下文件存在（首次任意 `inv` 命令会自动从 `config/*.example` 复制）：
  - `data/accounts/default/state/risk_rules.yaml`
  - `data/accounts/default/state/personal_redlines.yaml`
  - `data/accounts/default/db/assistant.sqlite`（H0 migrate）

---

## 数据文件与 ID 约定

| 项 | 路径 / 格式 |
|---|---|
| **Checklist 行（draft/submit 落库处）** | SQLite `data/accounts/default/db/assistant.sqlite` → 表 `checklist_submissions` |
| Checklist id | `cs_{YYYYMMDD}_{seq}`，如 `cs_20260527_001` |
| Risk exception id | `rx_{YYYYMMDD}_{seq}` |
| payload 工作文件 | 仅 **draft 输入**用；落库后进 SQLite，**不在** `state/` 目录 |
| **M7 阈值 SoT** | `data/accounts/default/state/risk_rules.yaml` |
| **禁区 SoT** | `data/accounts/default/state/personal_redlines.yaml` |

> ⚠️ **常见误解**：`draft` 不会把 JSON 写到 `state/`。`state/` 只放 YAML 配置（portfolio、risk_rules 等）；Checklist 正文在 **SQLite**。

### risk_rules / personal_redlines 从哪来？

**无需单独「录入」命令**。首次执行任意 `inv` 命令时，`EnsureInitialized` 会从仓库模板自动复制：

| 模板 | 复制到 |
|---|---|
| `config/risk_rules.yaml.example` | `data/accounts/default/state/risk_rules.yaml` |
| `config/personal_redlines.yaml.example` | `data/accounts/default/state/personal_redlines.yaml` |

修改阈值或禁用某条 redline：直接编辑上述 YAML（建议改前先备份）。M7 在 **submit** 时读取。

查看 draft 是否落库：

```powershell
sqlite3 data\accounts\default\db\assistant.sqlite "SELECT id, status, checklist_type, code FROM checklist_submissions WHERE id='cs_20260527_001';"
```

---

## 接口 1：`inv checklist draft`

### 签名

```text
inv checklist draft --type buy|watch|add|inspect|sell|review|import
                    [--code <code>] [--name <name>]
                    [--file <payload.json>] [--json]
```

### `--type` 枚举说明

> 完整说明（何时用、前置条件、approve 产物）→ **[ref-checklist-types.md](ref-checklist-types.md)**  
> 端到端流程（先 buy 再什么）→ **[01-操作流程总览.md](01-操作流程总览.md)**

| `--type` | 中文名 | 什么时候用 | approve 后 |
|---|---|---|---|
| `buy` | 建仓 | 首次买入，建立持仓 | portfolio **holding** + 新 lot |
| `add` | 加仓 | 已有 holding，追加买入 | 新 lot，股数/成本累加 |
| `sell` | 卖出 | 减仓或清仓 | lot_allocations + portfolio 减仓（见 H6） |
| `watch` | 观察池 | 感兴趣但尚未买 | watchlist.yaml 新条目 |
| `inspect` | 巡检 | 定期检视持仓逻辑 | inspection_records |
| `review` | 复盘 | 总结教训 | review_reports |
| `import` | 存量导入 | 一次性导入历史仓 | 多条 journal/lot（非日常流） |

**共用流水线**：`draft` → `submit`（M7）→ `approve`（落库）。仅 payload 字段与 approve 产物因 type 而异。

### 入参

| 参数 | 必填 | 说明 |
|---|---|---|
| `--type` | 是 | 上表之一；决定表单模板与 M7 场景 |
| `--code` | buy/add/sell 等建议填 | 标的代码，如 `600519` |
| `--name` | 否 | 标的名称，如 `贵州茅台` |
| `--file` | 否 | 完整 payload JSON 路径；**推荐**直接用示例文件 |
| `--json` | 否 | 输出 `{ "ID": "cs_…", "Status": "draft" }` |

### 成功输出

**场景：用示例文件创建 buy draft**

```powershell
Copy-Item docs\manual\examples\checklist-buy-600519.json .\checklist-buy-600519.json
.\bin\inv.exe checklist draft --type buy --code 600519 --name 贵州茅台 --file checklist-buy-600519.json
```

**预期输出**

```text
draft OK: id=cs_20260526_001 status=draft
```

> `cs_20260526_001` 中日期与序号以你本机当天为准；下文用 `<CS_ID>` 代指实际 id。

**`--json` 输出示例**

```json
{
  "ID": "cs_20260526_001",
  "Status": "draft"
}
```

### 失败输出示例

**场景：payload 缺少 `emotion_retrospect` 预留位**

```text
Error: payload 须含 emotion_retrospect 预留位（可设为 null）
```

---

## 接口 2：`inv checklist submit`

### 签名

```text
inv checklist submit <cs_id>
                   [--emotion-check "<文案>"]
                   [--exception-file <exception.json>]
                   [--json]
```

### 入参

| 参数 | 必填 | 说明 |
|---|---|---|
| `<cs_id>` | 是 | draft id，如 `cs_20260526_001` |
| `--emotion-check` | 条件 | `emotion_tag` 为 fomo/greedy/anxious 时必填 |
| `--exception-file` | 条件 | 触发 M7 hard_block 或 soft 警示时必填（见下方 JSON） |
| `--json` | 否 | 结构化输出 SubmitResult |

### 成功输出

**场景 A：calm 情绪，无 hard_block**

```powershell
.\bin\inv.exe checklist submit cs_20260526_001
```

```text
submit OK: id=cs_20260526_001 status=submitted hard_blocks=0 warnings=0 approve_blocked=false
```

**场景 B：fomo 须二次确认**

将 `checklist-buy-600519.json` 中 `"emotion_tag": "fomo"` 后重新 `draft --file`，再：

```powershell
.\bin\inv.exe checklist submit cs_20260526_001 --emotion-check "我承认有 FOMO 情绪，已对照 reversal_conditions 自检，仍决定小仓位试探"
```

**场景 C：hard_block + 例外文件**

```powershell
Copy-Item docs\manual\examples\checklist-exception-hard.json .\checklist-exception-hard.json
# 将 library_item_ids 改为你库中真实的 lib_* id
.\bin\inv.exe checklist submit cs_20260526_001 --exception-file checklist-exception-hard.json
```

```text
submit OK: id=cs_20260526_001 status=submitted hard_blocks=1 warnings=0 approve_blocked=true
注意: 触发 hard_block，H5 approve 将被门禁（须已填 exception_json）
```

### 失败输出示例

**场景：缺 `initial_pct`**

```text
Error: buy payload 缺少 position_size_plan.initial_pct
```

**场景：C/D tier 未确认**

```text
Error: 主要依据素材最高 tier 为 C/D 时须 tier_acknowledgement=true
```

**场景：未提供 --exception-file 但触发 hard_block**

```text
Error: 触发 hard_block，须提供 --exception-file
hard_block:
  - [m7_single_stock/risk_rules] 单标的仓位 拟达 17.00%，超过 hard_block 15%
要求：exception_reason≥80 字、expected_compensation、review_date、confirm_text、library_item_ids（S/A tier）
```

**场景：exception_reason 不足 80 字**

```text
Error: exception_reason 不足 80 字（当前 12 字）
hard_block:
  - [r004/personal_redlines] ...
```

### 退出码

| 码 | 场景 |
|---|---|
| 0 | submit 成功 |
| 1 | 参数 / 校验 / M7 错误 |

---

## 接口 2b：`inv checklist reject`

### 签名

```text
inv checklist reject <cs_id> --reason "<作废原因>" [--json]
```

### 说明

- 将 **draft** 或 **submitted** checklist 作废为 `rejected`（**不可逆**）
- **不能**把 buy 改成 add；type 在 draft 时已固定。误建 buy 时 reject 后 **新建** `--type add`
- `approved` 不可 reject；已落库修正须走新 submission

### 入参

| 参数 | 必填 | 说明 |
|---|---|---|
| `<cs_id>` | 是 | 要作废的 checklist id |
| `--reason` | 是 | 作废原因（写入 payload `_reject_meta`） |
| `--json` | 否 | 结构化输出 |

### 成功输出

**场景：误 submit 了 buy，标的已在 holding**

```powershell
.\bin\inv.exe checklist reject cs_20260529_001 --reason "误建 buy，600519 已在 holding，改走 add"
```

```text
reject OK: id=cs_20260529_001 status=rejected reason=误建 buy，600519 已在 holding，改走 add
提示: rejected 不可 approve；修正须新建 draft（如误建 buy 改走 add）
```

### 失败输出

```text
Error: 已 approved 不可 reject（须走新 submission 修正）
```

---

## 接口 3：`inv checklist show`

### 签名

```text
inv checklist show <cs_id> [--json]
```

### 成功输出（节选）

```powershell
.\bin\inv.exe checklist show cs_20260526_001
```

```text
id=cs_20260526_001 type=buy code=600519 name=贵州茅台 status=submitted
created=2026-05-26T10:00:00+08:00 submitted=2026-05-26T10:05:00+08:00
m7={"scenario":"buy","hard_blocks":[...],"approve_blocked":false,...}
--- payload ---
{ ... 完整 payload_json ... }
```

---

## 接口 4：`inv checklist list`

```powershell
.\bin\inv.exe checklist list --status submitted --type buy --code 600519
```

```text
cs_20260526_001	buy	600519	submitted	2026-05-26T10:00:00+08:00	2026-05-26T10:05:00+08:00
```

---

## payload 示例：`checklist-buy-600519.json`

完整文件见 [examples/checklist-buy-600519.json](examples/checklist-buy-600519.json)。核心字段节选：

```json
{
  "position_size_plan": { "initial_pct": 5, "max_pct": 10 },
  "execution_price": 1680,
  "shares": 100,
  "emotion_tag": "calm",
  "related_library_ids": [],
  "no_library_reason": "基于公开财报与个人渠道调研，暂无合适 L1 素材归档",
  "tier_acknowledgement": false,
  "emotion_retrospect": null
}
```

> **注意**：`execution_price`、`shares` 为实际成交价与股数，**须手动填写**（系统不联动券商）。`draft` 只校验 JSON 合法 + `emotion_retrospect` 存在；**完整必填字段在 `submit` 时校验**。

---

## exception 示例：`checklist-exception-hard.json`

完整文件见 [examples/checklist-exception-hard.json](examples/checklist-exception-hard.json)。`exception_reason` 须 **≥80 字**（中文按 rune 计）。

soft 警示最小示例（写入 `exception-soft.json`）：

```json
{ "acked": true, "ack_note": "我已阅读 r003 警示，确认不仅凭低估值建仓" }
```

---

## M7 检查项（MVP-1）

| 规则 id | 来源 | 说明 |
|---|---|---|
| `m7_single_stock` | risk_rules | 单标的 warning 10% / hard 15% |
| `m7_single_sector` | risk_rules | 单板块 30% / 40% |
| `m7_total_equity` | risk_rules | 总股票仓位 90% / 95% |
| `m7_single_thesis` | risk_rules | 单一 thesis 20% / 30% |
| `r003` | personal_redlines | 仅 valuation_repair 作理由 → soft |
| `r004` | personal_redlines | 无 L1 素材且**未填** `no_library_reason` 时 → hard；填了 `no_library_reason` 的纯个人判断建仓 **不触发** |

---

## 手动验证步骤

### 步骤 1：空 portfolio 下 calm buy 应 submit 成功

```powershell
# portfolio positions: []（见 H1 步骤 2）
.\bin\inv.exe checklist draft --type buy --code 600519 --name 贵州茅台 --file checklist-buy-600519.json
.\bin\inv.exe checklist submit cs_20260526_001
# 预期：approve_blocked=false（initial_pct=5，远低于 15%）
```

### 步骤 2：缺 initial_pct 应失败

编辑 JSON 删除 `position_size_plan.initial_pct`，重新 `draft --file` 后 `submit`：

```text
Error: buy payload 缺少 position_size_plan.initial_pct
```

### 步骤 3：SQLite 确认落库

```powershell
sqlite3 data\accounts\default\db\assistant.sqlite "SELECT id, checklist_type, status, length(payload_json) FROM checklist_submissions WHERE id='cs_20260526_001';"
```

```text
cs_20260526_001|buy|submitted|1234
```

```powershell
sqlite3 data\accounts\default\db\assistant.sqlite "SELECT id, severity, rule_id FROM risk_exceptions WHERE checklist_submission_id='cs_20260526_001';"
```

### 步骤 4：H5 approve

`approve_blocked=true` 时 **仍可 submitted**，但 `inv checklist approve` 会拒绝（除非 submit 时已填完整 `exception_json`）。正常建仓见 [H5-checklist-approve.md](H5-checklist-approve.md)。

---

## 自动化验证

```powershell
go test ./internal/core/checklist/... -v
go test ./internal/core/risk/... -v
```

覆盖：

- `TestValidateBuyMissingInitialPct` — 缺 initial_pct
- `TestEmotionNeedsSelfCheck` — fomo/greedy/anxious
- `TestM7SingleStockHardBlock` — 单标的 15% hard_block

---

## 故障排查

| 现象 | 处理 |
|---|---|
| `读取 risk_rules` 失败 | 确认 `data/accounts/default/state/risk_rules.yaml` 存在 |
| 引用 `lc_*` 失败 | 须先 H2 promote 为 `lib_*` |
| `approve_blocked=true` | 正常；用 `show` 查看 M7 JSON；approve 须 exception 或改 payload |
| draft 默认模板无法直接 submit | 须 `--file` 提供完整 payload（见示例 JSON） |

---

## 相关文档

- [01-操作流程总览.md](01-操作流程总览.md) — **先做什么、再做什么、分叉怎么走**
- [ref-checklist-types.md](ref-checklist-types.md) — `--type` 枚举权威说明
- [ref-sqlite-decision-tables.md](ref-sqlite-decision-tables.md) — `checklist_submissions` / `risk_exceptions` 表
- [H2-library归纳流水线.md](H2-library归纳流水线.md) — 获取 `lib_*` 供 `related_library_ids`
