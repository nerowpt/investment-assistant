# ref - Checklist 类型（`--type` 枚举）

> 权威来源：`internal/core/checklist/validate.go` · `approve.go`  
> 关联：[H4-checklist与M7.md](H4-checklist与M7.md) · [01-操作流程总览.md](01-操作流程总览.md)

---

## 一句话

`inv checklist draft --type <类型>` 中的 **type** 决定：填什么表单、跑哪套 M7 规则、approve 后写入哪些表/YAML。

**所有类型共用同一流水线**：`draft` → `submit`（M7）→ `approve`（落库）。差异在 payload 字段与 approve 产物。

---

## 类型总表

| `--type` | 中文名 | 什么时候用 | 前置条件 | approve 后产物 | 示例 payload |
|---|---|---|---|---|---|
| `buy` | 建仓 | 首次买入某标的，建立持仓 | portfolio 中该 code **无** holding（或为空库） | `journals` + `lots` + `portfolio.yaml` 新 holding | [checklist-buy-600519.json](examples/checklist-buy-600519.json) |
| `add` | 加仓 | 已有持仓，追加买入 | 该 code 已在 **holding**；payload 填 `linked_buy_journal_id` | 新 `lot` + portfolio 加权成本/股数 | [checklist-add-600519.json](examples/checklist-add-600519.json) |
| `sell` | 卖出/减仓 | 部分或全部卖出 | 有 **open/partial lots** | `lot_allocations` + lot 状态更新 + portfolio 减仓 | [checklist-sell-600519.json](examples/checklist-sell-600519.json) |
| `watch` | 加入观察池 | 感兴趣但尚未买入 | 无 | `watchlist.yaml` 新条目 `w_*` | H4 默认模板 |
| `inspect` | 持仓巡检 | 定期检视持仓逻辑是否仍成立 | 该 code 在 **holding** | `inspection_records` + 可选报告文件 | H4 默认模板 |
| `review` | 复盘 | 卖出后或阶段性总结教训 | 通常针对已 closed 或 holding | `review_reports` | H4 默认模板 |
| `import` | 存量导入 | 一次性导入历史持仓（非 Checklist 决策流） | 按 import payload 定义 | 多条 journal/lot + portfolio | 见 02 §十六 import |

---

## 各类型说明

### buy（建仓）

**用户意图**：「我决定买入，要把决策理由和实际成交价记下来。」

| 项 | 说明 |
|---|---|
| 必填 payload | `execution_price`（实际成交价）、`shares`（股数）、`position_size_plan.initial_pct` 等（见示例 JSON） |
| M7 场景 | `scenario=buy`，检查单标的/板块/总仓位 |
| submit 后 | `status=submitted`；若 `approve_blocked=true` 须带 `--exception-file` |
| approve 后 | 新建 lot（cost_basis=execution_price）、portfolio 进入 holding |
| 下一步常见分支 | 持有 → `inspect`；继续买 → `add`；卖出 → `sell` |

> 查现价用 `inv worker quote`（**仅参考**）；成交价必须手动填入 payload。

---

### add（加仓）

**用户意图**：「已有仓位，追加买入，形成新 lot。」

| 项 | 说明 |
|---|---|
| 必填 payload | `linked_buy_journal_id`（首次建仓 journal）、`add_pct`、`execution_price`、`shares` |
| M7 场景 | `scenario=add`，在现有仓位上累加检查 |
| approve 后 | 新 lot；portfolio 股数累加、成本加权 |

---

### sell（卖出/减仓）

**用户意图**：「按实际卖出价和股数减仓，FIFO 匹配 lot。」

| 项 | 说明 |
|---|---|
| 必填 payload | `sell_shares`、`execution_price`、`sell_type`、`lesson` 等 |
| submit 后可选 | `inv checklist plan <cs_id>` 预览 FIFO 分配 |
| 调整分配 | 编辑 `lot_allocation_plan` → `set-payload --file` |
| approve 后 | `lot_allocations`（含盈亏金额）、lots 变 partial/closed |
| 详细手册 | [H6-sell-FIFO.md](H6-sell-FIFO.md) |

---

### watch（观察池）

**用户意图**：「先跟踪，不买。」

| 项 | 说明 |
|---|---|
| 必填 payload | `watch_reason`、`hypothesis`、`review_date` 等 |
| approve 后 | `watchlist.yaml` 新增条目；**不**写 lot/portfolio holding |
| 下一步 | 决定买入 → 新建 `buy` checklist |

---

### inspect（巡检）

**用户意图**：「持仓还在，定期检查 thesis 是否仍有效。」

| 项 | 说明 |
|---|---|
| 必填 payload | `inspection_type`、`linked_buy_journal_id`、`classification`、`planned_action` |
| approve 后 | SQLite `inspection_records`；可能生成 Markdown 报告 |
| 下一步 | 若结论为减仓 → `sell`；若加强 → `add` |

---

### review（复盘）

**用户意图**：「交易结束或阶段结束，写教训总结。」

| 项 | 说明 |
|---|---|
| approve 后 | `review_reports` |
| 与 sell 关系 | 可先 `sell` approve，再 `review`；也可独立复盘 |

---

### import（存量导入）

**用户意图**：「系统上线前已有持仓，批量导入而非走 buy 决策流。」

| 项 | 说明 |
|---|---|
| 使用频率 | 低；一次性迁移 |
| 注意 | payload 结构不同于 buy；见架构文档 02 §十六 |

---

## checklist 状态（与 type 正交）

| status | 含义 | 可执行命令 |
|---|---|---|
| `draft` | 刚创建，payload 可改 | `submit`、`reject`、`set-payload`、`show` |
| `submitted` | 已过 M7，待落库 | `approve`、`reject`、`plan`（仅 sell）、`set-payload`、`show` |
| `approved` | 已落库 | `show`；重复 `approve` 幂等返回原结果 |
| `rejected` | 已作废（不可逆） | `show`；修正须 **新建** draft |

---

## 相关命令速查

| 命令 | 适用 status | 作用 |
|---|---|---|
| `checklist draft --type …` | — | 创建 draft |
| `checklist submit <id>` | draft | M7 + → submitted |
| `checklist reject <id> --reason` | draft/submitted | 作废 → rejected |
| `checklist plan <id>` | submitted（sell） | 预览 FIFO |
| `checklist set-payload <id> --file` | draft/submitted | 更新 payload |
| `checklist approve <id>` | submitted | 落库 |
| `checklist show/list` | 任意 | 查看 |
