# watchlist.yaml 字段填写说明（用户向）

> 权威设计：[03 §10B.3](../03-数据架构与数据源.md) · [02 §16.1](../02-核心场景与功能边界.md)  
> 示例：[config/watchlist.yaml.example](../../config/watchlist.yaml.example)  
> Go 类型：[internal/core/store/yamlstore/watchlist.go](../../internal/core/store/yamlstore/watchlist.go)

---

## 这个文件是干什么的？

`watchlist.yaml` 是 **观察池**（「值得跟踪、尚未建仓」的清单）。  
**`watching` 状态不进 `portfolio.yaml`**（Q2A）。

路径：`{DATA_ROOT}/accounts/{account_id}/state/watchlist.yaml`

---

## 根级字段

| 字段 | 必填 | 说明 |
|---|---|---|
| `schema_version` | 是 | 固定 `1` |
| `meta.updated_at` | 是 | ISO8601 |
| `items` | 是 | 观察项列表；无观察时 `[]` |

---

## items[] 每个观察项

### 身份

| 字段 | 必填 | 可选值 | 说明 |
|---|---|---|---|
| `id` | 是 | `w_{YYYYMMDD}_{seq}` | 系统生成，勿重复 |
| `code` | 条件 | 如 `"688xxx"` | 股票代码；**主题级**观察可空 |
| `name` | 是 | 任意 | 名称 |
| `watch_type` | 是 | 见下表 | 观察对象类型 |
| `state` | 是 | `watching` / `removed` | 当前状态 |

**`watch_type`**

| 值 | 含义 |
|---|---|
| `stock` | 单只股票 |
| `sector` | 板块/行业 |
| `theme` | 主题（如「光模块」） |
| `person` | 关注人物/大 V |
| `event` | 事件驱动 |

### 来源

| 字段 | 必填 | 可选值 | 说明 |
|---|---|---|---|
| `source_entry` | 是 | 见下表 | 如何进入观察池 |
| `source_candidate_id` | 条件 | `cand_*` | `from_candidate` 时必填 |

**`source_entry`**

| 值 | 含义 |
|---|---|
| `manual` | 手动加入 |
| `passive_discovery` | 系统被动发现 |
| `from_candidate` | 从 candidates.yaml 升级 |
| `from_inspection` | 巡检后转入观察 |

### 观察逻辑（用户填）

| 字段 | 必填 | 说明 |
|---|---|---|
| `watch_reason` | 是 | 为什么值得观察 |
| `hypothesis` | 是 | 待验证的核心假设 |
| `key_metrics_to_watch` | 是 | 跟踪指标列表，**≥1 项** |
| `expected_trigger` | 是 | 什么发生后考虑进入建仓分析 |
| `invalid_condition` | 是 | 什么发生后移出观察池 |
| `review_date` | 是 | 下次复查 `YYYY-MM-DD`（默认约 30 天后） |
| `priority` | 否 | `low` / `medium` / `high` |

### 关联

| 字段 | 说明 |
|---|---|
| `related_library_ids` | L1 素材 id 列表 |
| `related_positions` | 相关**已有持仓**的 code（可与 watching 并存） |
| `notes` | 自由备注 |

### removed 专用（state=removed 时）

| 字段 | 必填 | 说明 |
|---|---|---|
| `removed_at` | 是 | 移出时间 ISO8601 |
| `removed_reason` | 是 | 见下表 |
| `removed_detail` | 条件 | `removed_reason=other` 时必填 |
| `promoted_journal_id` | 条件 | `promoted_to_holding` 时必填 `j_*` |

**`removed_reason`**

| 值 | 含义 | 后续 |
|---|---|---|
| `invalidated` | 假设被证伪 | 仅保留记录 |
| `no_longer_relevant` | 不再相关 | 仅保留记录 |
| `promoted_to_holding` | 升级建仓 | **必须**填 `promoted_journal_id`，且 journal 须在 SQLite 存在 |
| `merged` | 合并到其他观察项 | — |
| `other` | 其他 | 须 `removed_detail` |

---

## 与 portfolio 的关系

| 规则 | 说明 |
|---|---|
| Q2A | `watching` **只在** watchlist，**不在** portfolio |
| 升级建仓 | watchlist → `removed` + `promoted_to_holding` + `promoted_journal_id`；portfolio 新增 `holding` |
| doctor 交叉 | 同一 `code` **不能**同时在 watchlist(`watching`) 与 portfolio(`holding`) |

---

## 最小 watching 示例

```yaml
schema_version: 1
meta:
  updated_at: "2026-05-22T12:00:00+08:00"
items:
  - id: w_20260522_001
    code: "600519"
    name: 贵州茅台
    watch_type: stock
    state: watching
    source_entry: manual
    watch_reason: 估值回到历史低位，需验证批价体系
    hypothesis: 批价企稳 + 渠道库存下降则逻辑成立
    key_metrics_to_watch:
      - 批价环比
    expected_trigger: 连续两季批价同比为正
    invalid_condition: 批价体系崩溃
    review_date: 2026-06-22
    created_at: "2026-05-22T12:00:00+08:00"
    related_library_ids: []
    related_positions: []
```

---

## 验证

```powershell
.\bin\inv.exe doctor --scope watchlist
.\bin\inv.exe doctor --scope all
```

见 [H1-portfolio与doctor.md](H1-portfolio与doctor.md) §watchlist。
