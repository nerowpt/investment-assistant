# portfolio.yaml 字段填写说明（用户向）

> 权威设计文档：[03 §10B.2](../03-数据架构与数据源.md)  
> 示例文件：[config/portfolio.yaml.example](../../config/portfolio.yaml.example)  
> Go 类型定义：[internal/core/store/yamlstore/portfolio.go](../../internal/core/store/yamlstore/portfolio.go)

---

## 这个文件是干什么的？

`portfolio.yaml` 记录 **「我现在持有什么」**（当前持仓视图）。  
**不要**在这里写历史故事——每笔买卖的完整记录在 SQLite 的 `journals` / `lots` 表。

路径：`{DATA_ROOT}/accounts/{你的account_id}/state/portfolio.yaml`

---

## 根级字段

| 字段 | 必填 | 类型 | 怎么填 |
|---|---|---|---|
| `schema_version` | 是 | 整数 | 固定填 `1`（以后升级格式会变） |
| `meta.updated_at` | 是 | 文本 | 最后一次合法修改时间，ISO8601，如 `2026-05-22T10:00:00+08:00` |
| `meta.total_equity_ref` | 否 | 数字或 null | 可选：你的总资产参考值（元），用于从股数推算仓位比例 |
| `meta.currency` | 否 | 文本 | 默认 `CNY` |
| `positions` | 是 | 列表 | 持仓列表；没有持仓时写 `[]` |

---

## positions[] 每个标的

### 身份与状态

| 字段 | 必填 | 可选值 | 说明 |
|---|---|---|---|
| `code` | 是 | 如 `"002624"` | 股票代码；**同一文件内唯一**（一只票一条） |
| `name` | 是 | 任意中文名 | 标的名称，便于人读 |
| `state` | 是 | 见下表 | 当前状态 |
| `position_type` | 是 | 见下表 | 仓位风格 |

**`state` 怎么选**

| 值 | 何时用 |
|---|---|
| `holding` | 当前仍持有（有仓位） |
| `closed` | 已清仓；**必须** `position_pct: 0` 且填 `closed_at` |

> ⚠️ **不要**写 `watching`——观察池在 `watchlist.yaml`，不在 portfolio（Q2A）。

**`position_type` 怎么选**

| 值 | 含义 | 典型场景 |
|---|---|---|
| `core` | 主仓 / 长线核心 | 3 年逻辑、低换手 |
| `swing` | 波段 / 热点副仓 | 博弈板块、事件驱动 |

### 仓位与成本（数字）

| 字段 | 必填 | 类型 | 说明 |
|---|---|---|---|
| `position_pct` | 是 | 数字 | **当前**占总资产百分比，如 `8` 表示 8%。= 所有未关闭 lot 的 `current_pct` 之和（系统 approve 后自动维护；手改会被 doctor 报错） |
| `cost_basis` | 是 | 数字 | 加权平均成本价（元/股） |
| `shares` | 否 | 数字 | 持股数量；不维护可省略 |
| `entry_date` | 是 | 日期 | 首次买入日，`YYYY-MM-DD` |
| `closed_at` | 条件 | 日期 | `state=closed` 时必填，清仓日 |

### 投资逻辑（用户必填，AI 不写）

| 字段 | 必填 | 说明 |
|---|---|---|
| `thesis_version` | 是 | 从 `1` 起；你更新持有逻辑时 +1 |
| `investment_thesis` | 是 | **当前**持有理由（多行文本）；不是历史快照 |
| `target_price` | 是 | 目标价：单个数字 **或** `{lower: 32, upper: 36}` 区间 |
| `stop_loss` | 是 | 止损/减仓线（元/股） |
| `reversal_conditions` | 是 | 逻辑反转条件，**至少 1 条**字符串列表 |
| `opportunity_cost_benchmark` | 是 | 见下表 |
| `confidence` | 否 | `low` / `medium` / `high` |

**`opportunity_cost_benchmark` 可选值**

| 值 | 含义 |
|---|---|
| `HS300` | 沪深 300 |
| `CSI_TECH` | 中证科技 / 科技类指数 |
| `sector_index` | 行业指数（需在 notes 说明具体指数） |
| `custom` | 自定义（notes 说明） |

### 关联 ID（通常由系统写入，勿手改）

| 字段 | 说明 |
|---|---|
| `lot_ids` | 关联的 lot 记录，格式 `lot_{日期}_{序号}` |
| `journal_ids` | 关联的 journal，格式 `j_{日期}_{序号}` |
| `related_library_ids` | 支撑当前逻辑的 L1 素材 id |
| `watchlist_origin_id` | 若从观察池升级，原 `w_*` id |
| `legacy_flags` | 存量导入标记，如 `legacy_over_limit` |

> 手改 `lot_ids` / `journal_ids` 而不产生对应 SQLite 流水时，`inv doctor --scope portfolio` **会报错**。

**`legacy_flags` 常见值**

| 值 | 含义 |
|---|---|
| `legacy_over_limit` | 导入时整体仓位已超 M7 上限 |
| `legacy_over_limit_sector` | 板块集中度超限（存量） |
| `legacy_over_limit_thesis` | 同一 thesis 暴露超限（存量） |

### monitoring（可选，巡检后缓存）

| 字段 | 说明 |
|---|---|
| `last_inspection_id` | 最近巡检 id，`insp_*` |
| `last_inspection_at` | 最近巡检时间 ISO8601 |
| `next_inspection_due` | 下次建议巡检日 `YYYY-MM-DD` |
| `classification` | 四象限分类（**用户填**，见下表） |
| `planned_action` | 计划动作（**用户填**，见下表） |

**`classification` 四象限（用户判断，非 AI）**

| 值 | 中文含义 |
|---|---|
| `low_valuation_logic_intact` | 低估值且逻辑还在 |
| `wait_for_style_switch` | 等待风格切换 |
| `cycle_bottom_or_reversal` | 周期低谷 / 逻辑反转中 |
| `value_trap_or_broken` | 价值陷阱 / 逻辑已破坏 |

**`planned_action` 计划动作（用户计划，非系统建议）**

| 值 | 含义 | 接下来通常做什么 |
|---|---|---|
| `hold` | 继续持有 | 更新下次巡检日 |
| `add` | 准备加仓 | 走加仓 Checklist |
| `reduce` | 准备减仓 | 走卖出 Checklist |
| `exit` | 准备清仓 | 走卖出 Checklist |
| `keep_watching` | 继续观察 | 更新监控重点 |

### 其他

| 字段 | 说明 |
|---|---|
| `sector_id` | 板块标签 id，对应 `controlled_tags.yaml`，如 `sector_gaming` |
| `thesis_id` | 同一投资逻辑分组，多标的可相同 |
| `notes` | 自由备注 |

---

## 填写示例（最小 holding）

```yaml
schema_version: 1
meta:
  updated_at: "2026-05-22T12:00:00+08:00"
  currency: CNY
positions:
  - code: "600519"
    name: 贵州茅台
    state: holding
    position_type: core
    position_pct: 10
    cost_basis: 1680.0
    entry_date: 2024-01-15
    thesis_version: 1
    investment_thesis: |
      高端白酒龙头，长期品牌护城河。
    target_price: 2200
    stop_loss: 1400
    reversal_conditions:
      - 批价体系崩溃且批价持续低于出厂价
    opportunity_cost_benchmark: HS300
    confidence: medium
    lot_ids: []      # approve 后系统自动填
    journal_ids: []  # approve 后系统自动填
```

---

## 验证

```powershell
.\bin\inv.exe doctor --scope portfolio
```

见 [H1-portfolio与doctor.md](H1-portfolio与doctor.md)。
