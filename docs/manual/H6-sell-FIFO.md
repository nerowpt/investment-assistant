# H6 - Sell + Lot FIFO（Q4C）

> 决议：Q4C / D3（[03 §10A.7](../03-数据架构与数据源.md) · [06 §D3](../06-架构Review决议.md)）  
> 代码状态：✅  
> 最后验证：2026-05-27

---

## 一句话

**sell** checklist 经 `approve` → 按 **股数 FIFO** 匹配 open lot → 写入 `lot_allocations` + 更新 lot 状态 + 同步 `portfolio.yaml`（减仓或 closed）。

> **记账 SoT**：`sell_shares` + `execution_price`（手动填写）；`sell_pct` 由系统按股数比例推算，供 M7/portfolio 维护。

> 📖 **前置**：[H5-checklist-approve.md](H5-checklist-approve.md)（须已有 holding + open lots）  
> 📄 **流程总览**：[01-操作流程总览.md](01-操作流程总览.md) §5.2  
> 📄 **sell 示例**：[examples/checklist-sell-600519.json](examples/checklist-sell-600519.json)

---

## 使用场景

| 场景 | 命令 | 产物 |
|---|---|---|
| 预览 FIFO 分配 | `checklist plan <cs_id>` | 终端输出 lot → allocated_shares |
| 用户调整分配 | 编辑 JSON → `set-payload --file` | payload.lot_allocation_plan + user_adjusted |
| 卖出落库 | `checklist approve` | journal(sell) + lot_allocations + snapshot |
| 多 lot 减仓 | 自动 FIFO（open_at 升序） | 可能跨多个 lot |
| 清仓 | sell_type=full 或 sell_shares=持仓总股数 | portfolio.state=closed |

---

## 前置条件

- [x] H5 已完成，且标的已在 **holding**（有 open/partial lots）
- [ ] `go build -o bin/inv.exe ./cmd/inv`
- [ ] 环境变量同 H4/H5

---

## 接口 1：`inv checklist plan`（sell）

### 签名

```text
inv checklist plan <cs_id> [--json]
```

### 说明

- 仅 **sell** checklist
- 若 payload 无 `lot_allocation_plan`，按 **股数** FIFO **计算**推荐（不写库）
- 若已有 plan，**校验** sum(allocated_shares)=sell_shares

### 成功 stdout 示例

```text
plan OK: checklist=cs_20260527_003 code=600519 sell_shares=50 execution_price=1750.0000 match=recommended_fifo
  lot=lot_20260527_001 allocated_shares=50 user_adjusted=false
提示: 调整 plan 后使用 checklist set-payload --file 写回，再 approve
```

---

## 接口 2：`inv checklist set-payload`

### 签名

```text
inv checklist set-payload <cs_id> --file <payload.json>
```

更新 **draft** 或 **submitted** 的完整 payload（含用户调整的 `lot_allocation_plan`）。

用户调整示例：

```json
"lot_allocation_plan": [
  { "lot_id": "lot_20260527_002", "allocated_shares": 50, "user_adjusted": true }
]
```

任一 `user_adjusted: true` → approve 后 `match_method=user_adjusted`。

---

## 接口 3：`inv checklist approve`（sell）

与 H5 相同命令；sell 时额外写入 `lot_allocations` 并 UPDATE lots。

### 约束

| 规则 | 说明 |
|---|---|
| sum(allocated_shares) = sell_shares | 否则 approve 失败 |
| execution_price > 0 | 实际卖出成交价，手动填写 |
| 不可匹配 closed lot | |
| plan 为空 | approve 时 **自动股数 FIFO** 并写回 payload |

---

## 手动验证（sell 端到端）

### 步骤 1：确保有持仓

沿用 H5 buy approve，或已有 `600519` holding + open lots。

### 步骤 2：draft → submit

```powershell
.\bin\inv.exe checklist draft --type sell --code 600519 --name 贵州茅台 --file docs\manual\examples\checklist-sell-600519.json
.\bin\inv.exe checklist submit cs_YYYYMMDD_NNN
```

### 步骤 3：plan 预览

```powershell
.\bin\inv.exe checklist plan cs_YYYYMMDD_NNN
```

### 步骤 4：（可选）调整 plan

复制 payload，修改 `lot_allocation_plan`，保存为 `sell-payload.json`：

```powershell
.\bin\inv.exe checklist set-payload cs_YYYYMMDD_NNN --file sell-payload.json
```

### 步骤 5：approve

```powershell
.\bin\inv.exe checklist approve cs_YYYYMMDD_NNN
```

### 步骤 6：SQLite 验证

```powershell
# 替换 j_* 为 approve 输出的 journal id
# lot_allocations
# lots.status 应为 partial 或 closed
# portfolio.yaml position_pct 减少
```

```sql
SELECT id, sell_journal_id, lot_id, allocated_shares, proceeds_amount, realized_pnl_amount, match_method, realized_return_pct
FROM lot_allocations WHERE sell_journal_id='j_...';

SELECT id, shares, current_pct, status, close_at FROM lots WHERE code='600519';
```

### 步骤 7：doctor

```powershell
.\bin\inv.exe doctor --scope portfolio
```

holding 标的须满足：`sum(open lots.current_pct) = position_pct`。

---

## 自动化验证

```powershell
go test ./internal/core/lot/... -v
go test ./internal/core/checklist/... -run Sell -v
```

---

## 故障排查

| 现象 | 处理 |
|---|---|
| `无 open/partial lot` | 先 buy approve |
| `sell_shares 超过 open lot 可卖总和` | 调小 sell_shares 或先 add |
| `sum(allocated_shares) != sell_shares` | 用 plan 核对，set-payload 修正 |
| `execution_price 须 > 0` | payload 填写实际卖出成交价 |
| `database is locked` | 勿在单连接嵌套事务；已修复 la id 预分配 |

---

## 相关文档

- [H5-checklist-approve.md](H5-checklist-approve.md)
- [ref-sqlite-decision-tables.md](ref-sqlite-decision-tables.md) — lot_allocations 表
