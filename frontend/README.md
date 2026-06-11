# 投资助手前端（H8 · uni-app）

一套代码可编译 **H5 / 微信小程序 / APP**。业务逻辑只通过 `src/api/` 访问后端，不直接操作本地文件。

## 前置

1. 启动 Go HTTP API（仓库根目录）：

```powershell
$env:DATA_ROOT = ".\data"
$env:IA_ACCOUNT_ID = "default"
$env:IA_CONFIG_ROOT = ".\config"
go build -o bin/inv.exe ./cmd/inv
.\bin\inv.exe api --addr :8787
```

或直接：

```powershell
go run ./cmd/ia-api --addr :8787
```

2. 安装依赖并启动 H5：

```powershell
cd frontend
npm install
npm run dev:h5
```

浏览器打开控制台提示的地址（默认 `http://localhost:5173`）。Vite 已将 `/api` 代理到 `127.0.0.1:8787`。

## 目录

| 路径 | 说明 |
|---|---|
| `src/api/` | HTTP 封装（`uni.request`），换服务器只改 `config/index.ts` |
| `src/store/wizard.ts` | 向导 session（内部持有 checklist id，用户无感） |
| `src/components/DynamicForm.vue` | 按后端 schema 动态渲染表单 |
| `src/pages/dashboard/` | 首页：持仓 + 「我要买入/加仓/卖出/巡检」 |
| `src/pages/wizard/` | 向导：填表 → M7 → 确认落库 |
| `src/pages/records/` | 决策记录列表 |

## 编译小程序（预留）

```powershell
npm run dev:mp-weixin
```

用微信开发者工具打开 `frontend/dist/dev/mp-weixin`。须将 `src/config/index.ts` 的 `apiBaseURL` 改为可访问的服务器地址。

## 多端配置

`src/config/index.ts`：

- `apiBaseURL`：H5 开发留空（走 proxy）；小程序/APP 填 `http://your-server:8787`
- `accountId`：对应后端 `X-Account-Id` header
