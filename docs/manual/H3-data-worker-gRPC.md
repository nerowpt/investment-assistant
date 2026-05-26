# H3 - Python data-worker gRPC

> 决议：T7 / T12 / T13（04 §二十一）  
> 代码状态：✅  
> 最后验证：2026-05-26

---

## 一句话

Go CLI 通过 gRPC 调用本机 Python **data-worker**，从东财/新浪等源拉取**客观行情事实**；worker 由 supervisor 懒启动，地址写入 `DATA_ROOT/.run/worker.port`。

---

## 使用场景（读这一节再敲命令）

`inv worker` **不是**资料入库命令，也**不会**自动写入 SQLite / YAML。它的定位是 **「外部事实数据管道」**，供你在决策流程里**按需取数**或给后续模块（H5 approve 快照）提供数据源。

| 命令 | 典型场景 | 谁在用 | 数据去向 |
|---|---|---|---|
| `worker health` | 排查 worker / 网络 / 依赖是否正常 | 你（验收、排错） | 仅 stdout，不落库 |
| `worker quote` | 建仓/加仓前**看一眼现价**；核对某标的实时价 | 你（CLI 临时查） | 默认**不落库**；H5 approve 时会自动调 worker 写入 `data_snapshot` |
| `worker valuation` | 填 Checklist 时辅助看 PE/PB（**只呈现数据，不下结论**） | 你（CLI 临时查） | 同上，H5 快照自动引用 |
| `worker announcements` | 预览某标的近期公告/新闻列表 | 你 / 日后 crawl | H2 `library crawl` 会把公告 **candidate 化**（H5 前 stub/半自动） |

### 与 `inv library` 的关系（重要）

| 你想做的事 | 用哪个 | 说明 |
|---|---|---|
| 把一篇研报/笔记/URL **纳入 L1 素材库** | `inv library ingest …` | 写入 `library_candidates` → promote 成 `library_item` |
| 查一下 **600519 现在多少钱** | `inv worker quote 600519` | 只返回行情 JSON/文本，**不会**进 library |
| 建仓时 **冻结决策时点的价格事实** | H5 `inv checklist …` → approve | 系统自动调 worker，写入 `data_snapshots`（无需你手动 quote） |

**结论**：MVP-1 阶段 `inv worker` 主要是 **调试管道 + 人工临时查数**；与 library 入库 **没有自动联动**，需要入库请走 `inv library ingest`。

### 端到端示意（MVP-1 → H5）

```text
【现在 H3 可用】
  你 ── inv worker quote ──► 看一眼行情（stdout）

【H5 approve 后自动】
  buy checklist approve ──► Go 调 worker.FetchQuote/Valuation
                         ──► 写入 data_snapshots（只读冻结）
                         ──► 与 library_item 引用并列，不互相替代

【素材入库始终独立】
  你 ── inv library ingest --url … ──► library_candidates ──► promote ──► library_item
```

---

## 前置条件

- [ ] Go 1.22+，`go build -o bin/inv.exe ./cmd/inv`
- [ ] **Python 3.10+** 已安装
- [ ] 安装 worker 依赖：

```powershell
cd services/data-worker
python -m pip install -e .
```

- [ ] 可选：生成 proto stub → `make proto`

---

## 接口签名

```text
inv worker health [--json]
inv worker restart
inv worker quote <code> [--json]
inv worker valuation <code> [--json]
inv worker announcements --code 600519 [--since ISO8601] [--json]
```

---

## 验证步骤

### 1. 探活（自动拉起 worker）

```powershell
$env:DATA_ROOT = "./data"
.\bin\inv.exe worker health
```

**成功输出示例**

```text
worker OK: version=0.1.0 python=3.12.x providers=[akshare]
port_file=...\data\.run\worker.port
```

### 2. 拉取行情

> **若刚改过 Python 代码**：先执行 `inv worker restart`，再 `inv worker health`。

```powershell
.\bin\inv.exe worker quote 600519
.\bin\inv.exe worker quote 600519 --json
```

**成功输出示例**

```text
600519 贵州茅台  price=1273.38 change_pct=-0.97% tier=A source=eastmoney
```

### 3. 估值

```powershell
.\bin\inv.exe worker valuation 600519
```

**成功输出示例**

```text
600519 PE=14.63 PB=5.89 as_of=2026-05-26 tier=A
```

---

## 架构要点

| 组件 | 路径 |
|---|---|
| Proto | `proto/dataworker/v1/dataworker.proto` |
| Go client + supervisor | `internal/worker/` |
| CoreIngest（Go server） | `internal/coreingest/` @ `127.0.0.1:50051` |
| Python 服务 | `services/data-worker/data_worker/` |
| worker 端口文件 | `{DATA_ROOT}/.run/worker.port` |

**边界**：Python **不得**读写 SQLite / YAML；仅通过 gRPC 返回事实数据。

---

## 失败样例

| 场景 | 现象 | 是否漏步骤 |
|---|---|---|
| 未装 Python 依赖 | `worker health` 就失败 | ✅ 需 `pip install -e .` |
| `health` OK，`quote/valuation` 报 `RemoteDisconnected` | 东财 API 被断连 | ❌ 非安装问题；删 port 重启 worker，或查网络 |
| supervisor 3 次启动失败 | `data-worker unavailable` | 检查 Python 路径 / 依赖 |

**自测 Python 数据源**：

```powershell
cd services/data-worker
python -c "from data_worker.fetch.akshare_quote import fetch_quote, fetch_valuation; print(fetch_quote('600519')); print(fetch_valuation('600519'))"
```

---

## 自动化验证

```powershell
go test ./internal/worker/... ./internal/coreingest/...
go test ./...
```

---

## 关联文档

- [04 §二十一](../04-技术架构.md#二十一round-3protobufpython-worker-与-supervisor)
- [05 H3](../05-MVP-Roadmap.md#h3--python-data-worker-grpc)
- [H2 library 入库](H2-library归纳流水线.md)（素材与 worker 分工）
- [services/data-worker/README.md](../../services/data-worker/README.md)
