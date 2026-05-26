# data-worker (Python)

gRPC 服务，实现 `dataworker.v1.DataWorker`。Go `inv` 通过 `internal/worker` 作为客户端调用。

## 环境要求

- Python **3.10+**
- 网络（akshare 拉行情）

## 安装

```powershell
cd services/data-worker
python -m pip install -e .
```

## 环境变量

| 变量 | 说明 |
|---|---|
| `IA_WORKER_LISTEN` | 监听地址，如 `127.0.0.1:50052`（supervisor 注入） |
| `IA_WORKER_PORT_FILE` | supervisor 注入；worker 绑定后写入实际地址 |
| `IA_PYTHON` | 指定 Python 可执行文件（Windows 若 `python` 指向商店占位程序时必设） |
| `IA_CORE_INGEST_TARGET` | Go CoreIngest 地址，默认 `127.0.0.1:50051` |
| `IA_ACCOUNT_ID` | 仅日志，**禁止**用于访问 DATA_ROOT |

## 开发启动（手动）

```powershell
$env:IA_WORKER_LISTEN = "127.0.0.1:50052"
python -m data_worker
```

## 用户路径（推荐）

无需手动起 worker；由 Go supervisor 懒启动：

```powershell
.\bin\inv.exe worker health
.\bin\inv.exe worker quote 600519
```

## Proto 生成

```powershell
make proto-py
```

生成物：`data_worker/pb/`（勿手写）。

## 边界

- **禁止** import sqlite3 访问业务库、禁止读写 portfolio/watchlist YAML
- 只返回带来源 tier 的**客观事实**；不下投资结论

## 注释规范

Python 模块/函数须**简体中文** docstring，见 `.cursor/rules/chinese-comments.mdc`。
