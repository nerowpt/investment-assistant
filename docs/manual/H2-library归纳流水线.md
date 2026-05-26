# H2 - L1 归纳流水线

> 决议：—  
> 代码状态：✅  
> 最后验证：2026-05-19

---

## 一句话

把 URL / 文件 / 笔记录入为 **candidate**，经 promote / supplement / dismiss 归纳成正式 **library_item**，供后续 Checklist 引用。

---

## 使用场景

| 场景 | 怎么做 | 产物 |
|---|---|---|
| 读完一篇研报，想纳入 L1 | `library ingest --url …` → `promote` | `library_item`（可被 Checklist `related_library_ids` 引用） |
| 快速记一条调研笔记 | `library ingest --text "…" --title "…"` | candidate → promote |
| 发现与已有素材重复 | review 时选 **supplement** 或 **dismiss** | 合并到已有 item 或丢弃 |
| **只想查现价** | 用 [H3 `worker quote`](H3-data-worker-gRPC.md)，**不是** library | 不落库 |

**与 worker 分工**：library 管「研究素材长期归档」；worker 管「当时的市场事实快照」。两者不自动联动。

---

## 前置条件

- [x] H1 已完成（account 路径、doctor 基础）
- [ ] 已在仓库根目录执行 `go build -o bin/inv.exe ./cmd/inv`
- [ ] `DATA_ROOT` 指向可写目录（默认 `./data`）

---

## 接口签名

```text
inv library ingest|list|show|search|promote|supplement|dismiss|review|prune|archive|candidates …
inv tags list|add|disable
inv doctor --scope library   # 扩展：primary asset / dedup_key
```

---

## 典型流程

### 1. 录入文本笔记（非交互）

```powershell
.\bin\inv.exe library ingest --text "茅台 Q1 业绩超预期，关注批价" --title "茅台调研备忘" --stock 600519 --tier B --no-review
```

**成功输出示例**

```text
ingest OK: candidate=lc_20260519_001 status=pending match_tier=none dedup_key=meta:...
```

### 2. promote → library_item

```powershell
.\bin\inv.exe library promote lc_20260519_001 --content-type note --media-type text --tier B --tags event_earnings --yes
```

**成功输出**

```text
promote OK: lib_20260519_001
```

### 3. 精确重复 → 自动 dismiss

再次 ingest 相同 URL / 相同 dedup_key：

```text
ingest OK: candidate=lc_20260519_001 status=dismissed match_tier=exact dedup_key=url:...
auto: auto_dismiss_exact
```

### 4. doctor 扩展

```powershell
.\bin\inv.exe doctor --scope library
```

```text
doctor OK (scope=library, schema_version=1)
```

---

## 子命令速查

| 命令 | 作用 |
|---|---|
| `library ingest` | 主动录入 → candidate |
| `library candidates list` | 候选列表 |
| `library review --id lc_*` | 交互式归纳（1/2/3/s/q） |
| `library promote` | 非交互 promote |
| `library supplement --into lib_*` | 合并补充 + tags 并集 |
| `library dismiss` | 丢弃 |
| `library list` / `show` / `search` | 正式素材 CRUD 读 |
| `library prune` | TTL 180d 过期 |
| `library merge` / `link-cluster` | **MVP-2 stub** |
| `tags list` / `add` / `disable` | 受控标签 |
| `tags suggest/confirm/reject` | **MVP-1 stub** |

---

## 退出码

| 码 | 含义 |
|---|---|
| 0 | 成功 |
| 1 | 参数 / 运行错误（当前 cobra 默认） |

> H2 起规划 exit 2=业务校验失败；当前尚未统一。

---

## 自动化验证

```powershell
go test ./...
```

重点包：`internal/core/library`、`internal/core/ids`、`internal/core/doctor`

---

## 关联文档

- [03 §十C](../03-数据架构与数据源.md#十cround-4cli-命令面规格l1-归纳--受控标签--doctor)
- [05 H2 验收](../05-MVP-Roadmap.md#h2--l1-归纳流水线)
- [controlled_tags 模板](../../config/controlled_tags.yaml.example)
