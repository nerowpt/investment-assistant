# data-worker (Python)

gRPC 服务，实现 `dataworker.v1.DataWorker`。Go `core-server` 为客户端。

## 目录结构

见各模块中文 docstring；架构见 `docs/04-技术架构.md` §二十一。

## 环境变量

| 变量 | 说明 |
|---|---|
| `IA_WORKER_LISTEN` | Windows：host:port |
| `IA_WORKER_SOCKET` | Unix：socket 路径 |
| `IA_CORE_INGEST_TARGET` | Go CoreIngest 地址 |
| `IA_ACCOUNT_ID` | 仅日志 |

## 开发启动

```bash
cd services/data-worker
uv sync
uv run python -m data_worker
```

## 注释规范

Python 模块/函数须**简体中文** docstring，见 `.cursor/rules/chinese-comments.mdc`。
