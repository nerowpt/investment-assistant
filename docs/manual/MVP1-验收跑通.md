# MVP-1 验收跑通

> 权威标准：[05 §三 MVP-1 成功标准](../05-MVP-Roadmap.md) · [06 §D2 定量门槛](../06-架构Review决议.md)  
> 代码状态：H0–H7 ✅  
> 最后验证：2026-05-27

---

## 一句话

按本文 **路径 A** 可在干净 account 上 **2–3 小时** 跑通 E1–E10 技术验收；**路径 B** 用真实持仓 dogfood **≥4 周** 后签收 Q1–Q5，才视为 MVP-1 **可交付**。

> 📖 **环境**：[00-环境与快速开始.md](00-环境与快速开始.md)  
> 📖 **日常操作**：[01-操作流程总览.md](01-操作流程总览.md)  
> 📖 **单功能细节**：H0–H7 各手册

---

## 两种验收模式

| 模式 | 用时 | 通过标准 | 适用 |
|---|---|---|---|
| **A · 技术冒烟** | 2–3 h | E1–E10 均可 CLI/MCP 跑通；`doctor --scope all` OK | H7 完成后首次整体验收 |
| **B · 正式 DoD** | ≥4 周 | A 全部 + Q1–Q5 定量门槛 + dogfood 书面签收 | 启动 MVP-2 前 |

**建议顺序**：先跑完 A 并修 bug → 再在 `default` account 上跑 B。

---

## 路径 A：技术冒烟（逐步复制）

### A.0 一次性准备

```powershell
cd C:\Users\qs\Desktop\workspace\investment-assistant
go build -o bin/inv.exe ./cmd/inv

$env:DATA_ROOT = ".\data"
$env:IA_CONFIG_ROOT = ".\config"
# 使用独立 account，避免污染日常 default 数据
$env:IA_ACCOUNT_ID = "mvp1-smoke"
```

验证：

```powershell
.\bin\inv.exe version
go test ./...
```

**预期**：`go test` 全绿；version 输出 `account_id: mvp1-smoke`。

> 若你已在 `default` 上有脏数据，仍可用 `mvp1-smoke` 隔离验收；日常数据修复见 [H7-backup-MCP.md](H7-backup-MCP.md) restore 节。

---

### A.1 自动化门禁（必须通过）

| # | 命令 | 预期 |
|---|---|---|
| 1 | `go test ./...` | exit 0 |
| 2 | `go build -o bin/inv.exe ./cmd/inv` | 无报错 |
| 3 | `.\bin\inv.exe doctor --scope library` | `doctor OK (scope=library, …)` |
| 4 | `go test ./internal/mcp/... -run ReadOnlyOnly -v` | 9 tool、无写 tool |

---

### A.2 E9 · 备份（建议最先做）

对应 **E9**；后续步骤出错可 restore。

```powershell
.\bin\inv.exe backup create
.\bin\inv.exe backup list
```

**预期**：输出 `backup OK: id=YYYYMMDD_HHMMSS`；目录含 `manifest.json`、`state/`、`db/`。

```powershell
.\bin\inv.exe backup show <BACKUP_ID>
.\bin\inv.exe backup restore --from <BACKUP_ID> --dry-run
```

**预期**：dry-run 列出将覆盖的 `state`、`db`，**不**改数据。

详见 [H7-backup-MCP.md](H7-backup-MCP.md)。

---

### A.3 E2 · 素材入库 → 可被引用

对应 **E2**（S1）。

```powershell
.\bin\inv.exe library ingest --text "茅台 Q1 批价企稳，关注渠道改革" --title "茅台调研备忘" --stock 600519 --tier B --no-review
# 记下输出的 lc_*
.\bin\inv.exe library promote lc_YYYYMMDD_NNN --content-type note --media-type text --tier B --tags event_earnings --yes
# 记下 lib_*
.\bin\inv.exe doctor --scope library
```

**预期**：`promote OK: lib_*`；doctor library OK。

后续 buy payload 可填 `"related_library_ids": ["lib_…"]`（可选）。

详见 [H2-library归纳流水线.md](H2-library归纳流水线.md)。

---

### A.4 E3 · 观察池

对应 **E3**（另一标的，避免与后续 buy 冲突）。

保存 `watch-000858.json`：

```json
{
  "watch_reason": "五粮液渠道改革进展值得跟踪，尚未到买入价位",
  "hypothesis": "若批价与茅台价差收窄，存在相对收益机会",
  "review_date": "2026-07-31",
  "priority": "medium",
  "related_library_ids": [],
  "emotion_retrospect": null
}
```

```powershell
.\bin\inv.exe checklist draft --type watch --code 000858 --name 五粮液 --file watch-000858.json
.\bin\inv.exe checklist submit <CS_WATCH>
.\bin\inv.exe checklist approve <CS_WATCH>
.\bin\inv.exe doctor --scope watchlist
```

**预期**：approve 成功；`watchlist.yaml` 出现 `000858`；doctor watchlist OK。

---

### A.5 E4 · 建仓 buy

对应 **E4**（S2）。

```powershell
Copy-Item docs\manual\examples\checklist-buy-600519.json .\smoke-buy.json
# 编辑 smoke-buy.json：execution_price、shares 改为你的测试值（须 > 0）
.\bin\inv.exe checklist draft --type buy --code 600519 --name 贵州茅台 --file smoke-buy.json
.\bin\inv.exe checklist submit <CS_BUY>
.\bin\inv.exe checklist approve <CS_BUY>
```

**预期 stdout（示例）**：

```text
approve OK: id=cs_... status=approved
  journal=j_... lot=lot_... snapshot=snap_...
  yaml_sync=portfolio
```

```powershell
.\bin\inv.exe doctor --scope portfolio
```

**记下** `<J_BUY>` = approve 输出的 `journal=` id，add/inspect 要用。

> 若 submit 报 `hard_block`：用 [checklist-exception-hard.json](examples/checklist-exception-hard.json) → `submit --exception-file …`（Q4 验收例外字数 ≥80，见 §B.3）。

详见 [H4-checklist与M7.md](H4-checklist与M7.md) · [H5-checklist-approve.md](H5-checklist-approve.md)。

---

### A.6 E5 · 加仓 add

对应 **E5**（S2a）。

```powershell
Copy-Item docs\manual\examples\checklist-add-600519.json .\smoke-add.json
# 编辑：linked_buy_journal_id → <J_BUY>；execution_price、shares
.\bin\inv.exe checklist draft --type add --code 600519 --name 贵州茅台 --file smoke-add.json
.\bin\inv.exe checklist submit <CS_ADD>
.\bin\inv.exe checklist approve <CS_ADD>
.\bin\inv.exe doctor --scope portfolio
```

**预期**：新 `lot_*`；portfolio 股数/成本更新；open lots 的 `current_pct` 之和 = `position_pct`。

---

### A.7 E6 · 巡检 inspect

对应 **E6**（S4）。

保存 `inspect-600519.json`：

```json
{
  "inspection_type": "scheduled",
  "linked_buy_journal_id": "j_YYYYMMDD_NNN",
  "classification": "thesis_intact",
  "planned_action": "hold",
  "thesis_still_valid": true,
  "key_observations": ["批价仍稳", "渠道改革按计划推进"],
  "emotion_retrospect": null
}
```

将 `linked_buy_journal_id` 改为 `<J_BUY>`。

```powershell
.\bin\inv.exe checklist draft --type inspect --code 600519 --name 贵州茅台 --file inspect-600519.json
.\bin\inv.exe checklist submit <CS_INSP>
.\bin\inv.exe checklist approve <CS_INSP>
```

**预期**：`inspection=insp_*`；SQLite `inspection_records` 有新行。

---

### A.8 E7 · 卖出 sell + FIFO

对应 **E7**（S5）。

```powershell
Copy-Item docs\manual\examples\checklist-sell-600519.json .\smoke-sell.json
# 编辑：sell_shares ≤ 当前可卖股数；execution_price > 0
.\bin\inv.exe checklist draft --type sell --code 600519 --name 贵州茅台 --file smoke-sell.json
.\bin\inv.exe checklist submit <CS_SELL>
.\bin\inv.exe checklist plan <CS_SELL>
.\bin\inv.exe checklist approve <CS_SELL>
.\bin\inv.exe doctor --scope portfolio
```

**预期**：`lot_allocations` 有行；lot 状态 `partial` 或 `closed`；portfolio 股数减少。

详见 [H6-sell-FIFO.md](H6-sell-FIFO.md)。

---

### A.9 E8 · 复盘 review

对应 **E8**（S6）。

保存 `review-q1.json`：

```json
{
  "review_type": "quarterly",
  "period_start": "2026-01-01",
  "period_end": "2026-03-31",
  "confirmed_patterns": [
    "按计划分批止盈有效，避免 FOMO 持有",
    "C/D tier 素材仅作辅助，未作为主要依据"
  ],
  "missed_patterns": [],
  "tier_cd_usage_pct": 10,
  "notes": "MVP-1 smoke 复盘占位",
  "emotion_retrospect": null
}
```

```powershell
.\bin\inv.exe checklist draft --type review --file review-q1.json
.\bin\inv.exe checklist submit <CS_REV>
.\bin\inv.exe checklist approve <CS_REV>
```

**预期**：`review=rev_*`；`confirmed_patterns` 非空（Q2 最低要求）。

---

### A.10 E1 · 存量导入 import

对应 **E1**（S0.5）；在 smoke account 上导入**第二只**标的。

保存 `import-one.json`（摘自 [H5 §import](H5-checklist-approve.md)）：

```json
{
  "import_reason": "MVP-1 smoke 存量迁移",
  "positions": [
    {
      "code": "000001",
      "name": "平安银行",
      "position_type": "swing",
      "position_pct": 5,
      "cost_basis": 10.5,
      "import_thesis_summary": "存量导入占位 thesis",
      "legacy_flags": ["legacy_over_limit"]
    }
  ],
  "emotion_retrospect": null
}
```

```powershell
.\bin\inv.exe checklist draft --type import --file import-one.json
.\bin\inv.exe checklist submit <CS_IMP>
.\bin\inv.exe checklist approve <CS_IMP>
```

**预期**：portfolio 出现 `000001`；`legacy_flags` 含 `legacy_over_limit`。

---

### A.11 E10 · MCP 只读（Cursor）

对应 **E10**。

1. 按 [H7-backup-MCP.md](H7-backup-MCP.md) 配置 `.cursor/mcp.json`（`IA_ACCOUNT_ID=mvp1-smoke` 或 default）
2. 重启 Cursor，确认 MCP 连接成功
3. 在聊天中验证至少 2 个 tool：
   - `get_portfolio` → 含 600519 / 000001
   - `search_journal` code=600519 → 含 buy/add/sell journal

**预期**：返回 JSON；**无**写 tool（`approve_checklist` 等不存在）。

---

### A.12 E3 · worker 参考价（非 E 路径，建议顺带测）

```powershell
.\bin\inv.exe worker health
.\bin\inv.exe worker quote 600519
```

**预期**：health OK；quote 返回结构化 JSON（worker 不可用时 approve 仍可完成，但 snapshot 可能缺行情）。

详见 [H3-data-worker-gRPC.md](H3-data-worker-gRPC.md)。

---

### A.13 终局检查

```powershell
.\bin\inv.exe doctor --scope all
.\bin\inv.exe checklist list --status approved
.\bin\inv.exe backup create
```

**预期**：

```text
doctor OK (scope=library, schema_version=...)
doctor OK (scope=portfolio)
doctor OK (scope=watchlist)
```

| 检查项 | 命令 / SQL | 预期 |
|---|---|---|
| 无未修复 sync | 见 §附录 A | 0 行或未 resolved 为空 |
| checklist 类型覆盖 | `checklist list --status approved` | 含 buy/add/sell/watch/inspect/review/import |
| 备份可列 | `backup list` | ≥2 条（含步骤 A.2 与 A.13） |

---

### A.14 路径 A 签收表

复制下表，逐项打勾：

| ID | 路径 | 步骤 | ☐ |
|---|---|---|---|
| E1 | 存量导入 | A.10 | |
| E2 | 素材 → library | A.3 | |
| E3 | 观察池 | A.4 | |
| E4 | 建仓 buy | A.5 | |
| E5 | 加仓 add | A.6 | |
| E6 | 巡检 inspect | A.7 | |
| E7 | 卖出 FIFO | A.8 | |
| E8 | 复盘 review | A.9 | |
| E9 | backup / restore | A.2 + dry-run | |
| E10 | MCP 9 tool | A.11 | |
| NF | doctor --scope all | A.13 | |
| NF | go test 全绿 | A.1 | |

**路径 A 通过**：上表全部 ☐ → ☑。

---

## 路径 B：正式 DoD（dogfood）

> 门禁：[05 §3.1.2](../05-MVP-Roadmap.md) · 持仓基线：[docs/_baseline/02-当前持仓与个人逻辑.md](../_baseline/02-当前持仓与个人逻辑.md)

### B.1 数据与环境

```powershell
$env:IA_ACCOUNT_ID = "default"
.\bin\inv.exe backup create
```

1. 阅读 `_baseline/02` 中各标的 thesis / 止损 / 巡检结论
2. 用 **import** checklist 导入真实持仓（或逐只 **buy** 重建，耗时更长）
3. 按真实节奏完成至少一轮：**素材入库 → 巡检 → 复盘**

### B.2 非功能（05 §3.2）

| 项 | 验收 |
|---|---|
| 纯 CLI 可完成 E1–E9 | 关闭 Cursor，仅用 `inv` 完成一次 inspect + approve |
| 决策快照不可变 | approve 后改 library 内容，`get_journal` 中 payload 不变 |
| M7 默认启用 | submit 故意超限 → hard_block；例外须 `--exception-file` |
| Windows 本机 | Go + Python worker 均在本机跑通 |

### B.3 定量门槛 Q1–Q5

> 说明：`inv journal list`、`inv risk exceptions stats` 在路线图中规划，**MVP-1 尚未实现 CLI**；暂用 SQLite / MCP 替代。

#### Q1 · Journal ≥ 20 笔

```powershell
sqlite3 data\accounts\default\db\assistant.sqlite "SELECT action_type, COUNT(*) FROM journals GROUP BY action_type;"
sqlite3 data\accounts\default\db\assistant.sqlite "SELECT COUNT(*) FROM journals;"
```

| 指标 | 目标 |
|---|---|
| 总 journal 数 | ≥ 20 |
| buy / add / sell | 各 ≥ 1 |
| inspect | ≥ 1 条 `inspection_records`（inspect 不写 journals 表） |

```powershell
sqlite3 data\accounts\default\db\assistant.sqlite "SELECT COUNT(*) FROM inspection_records;"
```

#### Q2 · 完整 quarter 复盘 ≥ 1

```powershell
sqlite3 data\accounts\default\db\assistant.sqlite "SELECT id, review_type, period_start, period_end FROM review_reports;"
```

人工打开 `payload_json`，确认 `confirmed_patterns` 为用户手填、非空。

#### Q3 · tier C/D 主要依据占比 < 30%

在复盘 payload 或你的复盘 Markdown 中统计「主要依据」里 C/D tier 素材占比 < 30%。  
（MVP-1 无自动统计 CLI，**人工签收**。）

#### Q4 · hard block 例外说明 ≥ 80 字

```powershell
sqlite3 data\accounts\default\db\assistant.sqlite "SELECT id, checklist_submission_id, exception_json FROM risk_exceptions;"
```

检查 `exception_json` 内 `exception_reason` 字段长度；或回忆 submit 时 `--exception-file` 内容。  
目标：hard block 例外平均 ≥ 80 字。

#### Q5 · dogfood ≥ 4 周

| 项 | 记录 |
|---|---|
| 开始日期 | ______ |
| 结束日期 | ______ |
| 是否推翻 MVP-1 架构 | 是 / 否 |
| 用户书面签收 | ______ |

**失败处置**（05 §3.1.2）：若流程过繁 / 价值未感知 → **暂停 MVP-2**，回到 MVP-1 简化主路径（优先 inspect / library add）。

---

### B.4 路径 B 签收表

| ID | 门槛 | ☐ |
|---|---|---|
| A 全部 | 路径 A 已通过 | |
| Q1 | journal ≥ 20 + 各类型覆盖 | |
| Q2 | quarter 复盘 + confirmed_patterns | |
| Q3 | C/D 占比 < 30% | |
| Q4 | 例外说明 ≥ 80 字 | |
| Q5 | ≥4 周 dogfood 签收 | |
| D4 | 真实持仓 import → 巡检 → 复盘 一轮 | |

**MVP-1 可交付**：上表全部 ☑。

---

## 附录

### A · sync_repairs 检查

```powershell
sqlite3 data\accounts\default\db\assistant.sqlite "SELECT id, repair_type, status FROM sync_repairs WHERE status != 'resolved';"
```

**预期**：空结果，或已全部 resolved。

### B · 常用 checklist 命令

```powershell
.\bin\inv.exe checklist list --status approved --type buy
.\bin\inv.exe checklist show <CS_ID>
.\bin\inv.exe checklist reject <CS_ID> --reason "误建 buy，改走 add"
```

> `reject` 仅适用于 draft/submitted；**approved 不可 reject**，须 backup restore。

### C · 误操作恢复

| 情况 | 处理 |
|---|---|
| 误建 buy（已在 holding） | `reject` → 新建 `draft --type add` |
| 误 approve | `backup restore --from <id> --yes` → `doctor --scope all` |
| doctor P001/P002/P004 | [H1-portfolio与doctor.md](H1-portfolio与doctor.md) 错误码表 |

### D · 关联手册索引

| 里程碑 | 文档 |
|---|---|
| H0 | [H0-骨架与迁移.md](H0-骨架与迁移.md) |
| H1 | [H1-portfolio与doctor.md](H1-portfolio与doctor.md) |
| H2 | [H2-library归纳流水线.md](H2-library归纳流水线.md) |
| H3 | [H3-data-worker-gRPC.md](H3-data-worker-gRPC.md) |
| H4 | [H4-checklist与M7.md](H4-checklist与M7.md) |
| H5 | [H5-checklist-approve.md](H5-checklist-approve.md) |
| H6 | [H6-sell-FIFO.md](H6-sell-FIFO.md) |
| H7 | [H7-backup-MCP.md](H7-backup-MCP.md) |

---

## 变更日志

| 日期 | 变更 |
|---|---|
| 2026-05-27 | H7 完成后首版：路径 A 逐步冒烟 + 路径 B Q1–Q5 + 签收表 |
