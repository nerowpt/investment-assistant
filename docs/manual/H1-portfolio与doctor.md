# H1 - portfolio.yaml 与 doctor

> 决议：D11 / D12 / D13（[06 §四](../06-架构Review决议.md)）  
> 代码状态：🔄（portfolio + doctor 已实现；watchlist 等待下一批）  
> 最后验证：2026-05-22

---

## 一句话

读取/写入当前持仓 YAML，并检查它与 SQLite 里 lot、journal 引用是否一致——防止「YAML 和数据库各说各话」。

> 📖 **改 portfolio.yaml 前请先读**：[ref-portfolio-yaml-fields.md](ref-portfolio-yaml-fields.md)（每个字段怎么填、枚举值）  
> 📖 **理解 journals 表**：[ref-sqlite-decision-tables.md](ref-sqlite-decision-tables.md)  
> 📄 **带注释模板**：[config/portfolio.yaml.example](../../config/portfolio.yaml.example)

---

## 前置条件

- [x] H0 已通过：`inv doctor --scope library` → OK
- [x] `data/accounts/default/state/portfolio.yaml` 存在（首次 doctor 自动从 example 复制）

---

## 数据文件：`portfolio.yaml`

路径：`{DATA_ROOT}/accounts/{account_id}/state/portfolio.yaml`

SoT 含义：**当前持仓视图**；历史决策在 SQLite `journals` / `lots`。

### 根结构示例（节选）

```yaml
schema_version: 1
meta:
  updated_at: "2026-05-19T10:00:00+08:00"
  total_equity_ref: null
  currency: CNY
positions:
  - code: "002624"
    name: 完美世界
    state: holding
    position_type: swing
    position_pct: 8
    cost_basis: 18.20
    lot_ids:
      - lot_20260518_001
    journal_ids:
      - j_20260518_001
    monitoring:
      last_inspection_id: insp_20260618_001
      last_inspection_at: "2026-06-18T09:00:00+08:00"
      next_inspection_due: 2026-07-18
      classification: wait_for_style_switch
      planned_action: hold
```

完整 example：`config/portfolio.yaml.example`

---

## 接口 1：`inv doctor --scope portfolio`

### 签名

```text
inv doctor --scope portfolio [--account <id>]
```

### 检查项（03 §10B.8 + 06 §D11）

| 检查 | 说明 |
|---|---|
| schema / 基本约束 | `schema_version`、closed 须 `position_pct=0`、禁止 `watching` |
| `journal_ids` | 每条 id 在 SQLite `journals` 存在 |
| `lot_ids` | 每条 id 在 SQLite `lots` 存在且 `code` 匹配 |
| open lot 仓位之和 | `sum(lots.current_pct)` == `position.position_pct`（decimal 精度，禁 float） |

### 成功输出

**场景：空 portfolio（positions: []）且 DB 无引用**

```text
doctor OK (scope=portfolio)
```

> 若 example 模板含虚构 lot/journal id 而 DB 为空，**预期失败**（见下方失败样例）。

### 失败输出示例

**场景 A：example 模板 + 空数据库（当前默认情况）**

```text
Error: portfolio 校验失败:
  - 002624: lot_ids 引用不存在 lot_20260518_001
  - 002624: lot_ids 引用不存在 lot_20260618_001
  - journal_ids 引用不存在: j_20260518_001
  - journal_ids 引用不存在: j_20260618_002
  - 002624: sum(open lots.current_pct)=0 != position_pct=8
```

**场景 B：手改 YAML 把 closed 的 position_pct 改成非 0**

```text
Error: portfolio 校验失败:
  - 600519: closed 标的 position_pct 应为 0
```

### 退出码

| 码 | 场景 |
|---|---|
| 0 | 全部检查通过 |
| 1 | 任一检查失败 |

### 手动验证步骤

#### 步骤 1：确认「模板 + 空库」会失败（证明 doctor 在工作）

```powershell
.\bin\inv.exe doctor --scope library
.\bin\inv.exe doctor --scope portfolio
# 预期：失败，见场景 A
```

#### 步骤 2：清空 portfolio 引用后应通过

编辑 `data/accounts/default/state/portfolio.yaml`：

```yaml
schema_version: 1
meta:
  updated_at: "2026-05-22T12:00:00+08:00"
  currency: CNY
positions: []
```

```powershell
.\bin\inv.exe doctor --scope portfolio
# 预期：doctor OK (scope=portfolio)
```

#### 步骤 3：注入一致数据后应通过（SQLite 手工插入）

> H5 前无 `inv checklist approve`，验收用 SQL 模拟最小一致集。

在 PowerShell 中（需安装 sqlite3 CLI，或用 DB 工具执行）：

```sql
-- 文件：data/accounts/default/db/assistant.sqlite

INSERT INTO journals (id, action_type, code, payload_json, created_at)
VALUES ('j_20260518_001', 'buy', '002624', '{}', '2026-05-18T10:00:00+08:00');

INSERT INTO lots (id, code, journal_id, action_type, position_type, open_at,
  initial_pct, current_pct, cost_basis, status, created_at)
VALUES ('lot_20260518_001', '002624', 'j_20260518_001', 'buy', 'swing', '2026-05-18',
  8, 8, 18.2, 'open', '2026-05-18T10:00:00+08:00');
```

恢复 portfolio 单条 position（`lot_ids` / `journal_ids` / `position_pct: 8` 与上表一致），再跑：

```powershell
.\bin\inv.exe doctor --scope portfolio
# 预期：doctor OK
```

#### 步骤 4：组合检查

```powershell
.\bin\inv.exe doctor --scope all
# 预期：library OK + portfolio OK
```

### 自动化验证

```powershell
go test ./internal/core/store/yamlstore/... -run "Monitoring|RoundTrip|Validate"
go test ./internal/core/store/sqlstore/... -run TestSumDecimalColumn_PrecisionBoundary
```

覆盖：

- `TestPortfolio_MonitoringRoundTrip` — monitoring 子结构不丢失（D12）
- `TestSumDecimalColumn_PrecisionBoundary` — 0.1+0.2 精度（D11）
- `TestFilePortfolioStore_RoundTrip` — 原子读写（D13）

---

## 接口 2：`PortfolioStore`（编程接口，供 H4 测试 mock）

> 非 CLI；给开发者 / 未来 MCP 适配层使用。

### 签名

```go
type PortfolioStore interface {
    Load(ctx context.Context, path string) (*Portfolio, error)
    Save(ctx context.Context, path string, p *Portfolio) error
}

store := yamlstore.NewFilePortfolioStore()
// 或测试：yamlstore.NewMemoryPortfolioStore()
```

### 入参

| 参数 | 类型 | 说明 |
|---|---|---|
| `ctx` | context.Context | 预留取消/超时 |
| `path` | string | portfolio.yaml **绝对或相对路径** |
| `p` | *Portfolio | Save 时的完整结构 |

### 出参 / 错误

| 结果 | 说明 |
|---|---|
| `*Portfolio` | Load 成功 |
| `yamlstore.ErrNotFound` | 文件不存在（Memory/File 统一语义） |
| 其他 error | YAML 解析失败、IO 失败 |

### 使用示例

```go
ctx := context.Background()
path := filepath.Join(ac.StateDir, "portfolio.yaml")
store := yamlstore.NewFilePortfolioStore()

p, err := store.Load(ctx, path)
if errors.Is(err, yamlstore.ErrNotFound) {
    // 首次使用
}
p.Meta.UpdatedAt = time.Now().Format(time.RFC3339)
err = store.Save(ctx, path, p)
```

---

## 边界与限制

| 项 | 说明 |
|---|---|
| `--json` | H1 未实现；H4 起统一 |
| `--fix` | 03 规划；MVP-1 未实现 |
| 手改 `lot_ids` | doctor 会报错；正确路径是 Checklist approve（H5） |
| REAL 列 | SQLite 仍用 REAL 存储；**读取**走 `CAST AS TEXT` + decimal（D11） |

---

## 接口 3：`inv doctor --scope watchlist`

> 📖 填 watchlist 前读：[ref-watchlist-yaml-fields.md](ref-watchlist-yaml-fields.md)

### 签名

```text
inv doctor --scope watchlist [--account <id>]
```

### 检查项

| 检查 | 说明 |
|---|---|
| schema / 必填字段 | `ValidateWatchlist` |
| `promoted_to_holding` | 必有 `promoted_journal_id`，且 journal 在 SQLite 存在 |
| Q2A 交叉 | 同一 code 不能同时在 watchlist(`watching`) 与 portfolio(`holding`) |

### 成功输出

```text
doctor OK (scope=watchlist)
```

### 手动验证

```powershell
.\bin\inv.exe doctor --scope watchlist
.\bin\inv.exe doctor --scope all   # library + portfolio + watchlist
```

### 自动化验证

```powershell
go test ./internal/core/store/yamlstore/... -run Watchlist
go test ./internal/core/doctor/... -run Watchlist
```

---

## H1 剩余验收

| 项 | 状态 |
|---|---|
| watchlist + doctor | ✅ |
| emotion_retrospect 文档 | H4 checklist 手册 |

---

## 关联文档

- [03 §10B.2](../03-数据架构与数据源.md) portfolio 字段
- [03 §10B.8](../03-数据架构与数据源.md) doctor 规则
- [06 §D11–D13](../06-架构Review决议.md)
