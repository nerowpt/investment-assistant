# H8 - Web API + uni-app 友好前端

> 决议：MVP-2 起步 · 本地优先、为托管预留（[05 §六 H8](../05-MVP-Roadmap.md)）  
> 代码状态：✅  
> 最后验证：2026-05-27

---

## 一句话

用 **HTTP API + uni-app 向导界面** 替代「记命令、记 ID」的 CLI 体验：用户点「我要买入」→ 填表 → 风险检查 → 确认落库，全程无需接触 `cs_` / `j_` 等内部编号。

> 📖 **CLI 仍保留**：高级用户与自动化仍用 `inv checklist …`  
> 📖 **前端 README**：[frontend/README.md](../../frontend/README.md)

---

## 架构

```text
uni-app (H5/小程序/APP)
    ↓ HTTP JSON
cmd/ia-api / inv api  (:8787)
    ↓ 复用
internal/core/checklist|query|doctor
    ↓
SQLite + YAML（与 CLI 相同 SoT）
```

---

## 启动（本地开发）

### 1. 后端 API

```powershell
cd C:\Users\qs\Desktop\workspace\investment-assistant
$env:DATA_ROOT = ".\data"
$env:IA_ACCOUNT_ID = "default"
$env:IA_CONFIG_ROOT = ".\config"

go build -o bin/inv.exe ./cmd/inv
.\bin\inv.exe api --addr :8787
```

或：

```powershell
go run ./cmd/ia-api --addr :8787
```

**预期**：

```text
ia-api listening on :8787 (account=default data_root=...)
```

### 2. 前端 H5

```powershell
cd frontend
npm install
npm run dev:h5
```

浏览器打开 `http://localhost:5173`。Vite 将 `/api` 代理到 `127.0.0.1:8787`。

---

## HTTP API 索引

统一响应（04 §19.5）：`{ "code": 0, "success": true, "data": ... }`；失败时 `success: false` + `message`，服务端 `log.Printf` 记录错误详情。  
请求头：`X-Account-Id`（可选，默认 default）

### 读接口

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/health` | 探活 |
| GET | `/api/portfolio` | 持仓列表 |
| GET | `/api/watchlist` | 观察池 |
| GET | `/api/pool` | 选股看板五分区聚合（H8.1） |
| GET | `/api/pool/buy-context` | 从观察区/研究区进入买入的预填上下文 |
| GET | `/api/review/workbench` | 复盘工作台：已关闭 lot 列表（H8.3） |
| GET | `/api/review/lot-context` | 单笔 lot 复盘上下文与预填（H8.3） |
| GET | `/api/research/{code}` | 研究档案（H8.2） |
| POST | `/api/research/{code}/fetch` | 按需拉取数据包 |
| POST | `/api/research/{code}/library` | 拉取结果纳入 L1 |
| GET | `/api/journals` | 决策 journal 列表 |
| GET | `/api/journals/{id}` | 单条 journal |
| GET | `/api/checklists` | checklist 列表 |
| GET | `/api/checklists/{id}` | checklist 详情 + M7 结果 |
| GET | `/api/checklist/schema?type=buy` | 动态表单 schema |
| GET | `/api/doctor?scope=all` | 数据体检 |

### 写接口（向导用）

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/checklist` | 创建 draft |
| POST | `/api/checklist/{id}/preview` | M7 预览（不改 status） |
| POST | `/api/checklist/{id}/submit` | 提交 + M7 落库 |
| POST | `/api/checklist/{id}/plan` | sell FIFO 预览 |
| POST | `/api/checklist/{id}/approve` | 正式落库 |
| POST | `/api/checklist/{id}/reject` | 作废 |
| POST | `/api/risk/check` | M7 模拟 |

---

## 前端页面

| 页面 | 用户看到什么 |
|---|---|
| **首页** | 持仓卡片 + 「选股看板 / 我要买入」+ 加仓/卖出/巡检 |
| **选股看板** | 观察 / 研究 / 已建仓 / 已卖出 / 波段 五 Tab |
| **研究档案** | 输入代码 → 拉取估值/新闻 → 纳入 L1 |
| **复盘工作台** | 已卖出 lot 列表 → 写单笔归因复盘 |
| **决策向导** | 分步：填表 → 风险检查（含例外说明）→ 确认落库 → 完成 |
| **决策记录** | 交易记录 / 决策表单列表（无需记 ID） |

### 傻瓜式设计要点

1. **功能命名自解释**：按钮文案即意图，无 CLI 术语。
2. **智能模板**：进入向导即加载 schema 默认值 + 字段 tip 释义。
3. **ID 无感**：`checklistId` 存在 Pinia store，用户不见 `cs_*`。
4. **自动串联**：填表 → preview → submit → approve 一键流程。

---

## 手动验证（GUI 版路径 A）

1. 启动 API + H5
2. 首页点 **我要买入** → 填 `execution_price`、`shares` → 下一步
3. 风险页点 **确认并提交** → **确认执行**
4. 首页应出现新持仓
5. 对同一标的：**加仓** → **巡检** → **卖出**（全程不输入任何 id）
6. **决策记录** 页可见 journal / checklist 列表

---

## 多端与托管预留

| 项 | MVP-2 首版 | 后续 |
|---|---|---|
| 鉴权 | 中间件占位，默认放行 | JWT + 多用户 |
| API 地址 | H5 走 vite proxy | 改 `frontend/src/config/index.ts` |
| 小程序 | 架构可编译，未打包验收 | 配置合法 request 域名 |
| 写 API | 已暴露 draft/submit/approve | H15 可迁 Kratos |

---

## 自动化验证

```powershell
go test ./internal/api/... -v
go test ./internal/core/checklist/... -run FormSchema -v
go build ./cmd/ia-api
```

---

## 关联文档

- [H8.1-选股看板.md](H8.1-选股看板.md) — 看板线框与 API 字段清单
- [H8.2-研究档案.md](H8.2-研究档案.md) — 研究页与 data-worker 拉取
- [H8.3-复盘工作台.md](H8.3-复盘工作台.md) — 单笔 lot 归因复盘
- [MVP1-验收跑通.md](MVP1-验收跑通.md) — CLI 版验收；GUI 等价见上文
- [ref-checklist-types.md](ref-checklist-types.md) — 字段语义（schema 来源）
- [04 §十九](../04-技术架构.md) — 未来 Kratos ia-api
