# H7 - Backup + MCP 只读

> 决议：O5 备份策略 / T6 九 tool（[03 §10D](../03-数据架构与数据源.md) · [04 §二十二](../04-技术架构.md)）  
> 代码状态：✅  
> 最后验证：2026-05-27

---

## 一句话

**backup** 在 `approve` 等写操作前保护 account 数据；**MCP** 让 Cursor 通过 **9 个只读 tool** 读取 portfolio / journal / library，**不能**代替用户 approve 或改 yaml。

> 📖 **前置**：[H1-portfolio与doctor.md](H1-portfolio与doctor.md)（doctor 验收）  
> 📄 **流程总览**：[01-操作流程总览.md](01-操作流程总览.md)  
> 📄 **MCP schema 详表**：[04 §二十二](../04-技术架构.md)

---

## 使用场景

| 场景 | 命令 / 入口 | 产物 |
|---|---|---|
| approve 前快照 | `inv backup create` | `{BACKUP_ROOT}/{account}/{timestamp}/` + manifest.json |
| 误操作回滚 | `inv backup restore --from … --yes` | 覆盖 state/db；自动 pre_restore 快照 |
| 清理旧备份 | `inv backup prune --keep 8` | 删除超出保留份数的目录 |
| Cursor 读持仓 | MCP `get_portfolio` | portfolio.yaml JSON |
| Cursor 查决策记录 | MCP `search_journal` / `get_journal` | journals + snapshot 摘要 |
| 填表前拿模板 | MCP `get_checklist_template` | 默认 payload 结构 |

---

## 前置条件

- [x] H1–H6 已完成，`inv doctor --scope all` 可运行
- [ ] `go build -o bin/inv.exe ./cmd/inv`
- [ ] 环境变量（与 H4 相同）：

```powershell
$env:DATA_ROOT = ".\data"
$env:IA_ACCOUNT_ID = "default"
$env:IA_CONFIG_ROOT = ".\config"
```

备份默认写入 `{DATA_ROOT}/_backups/{account_id}/`（与 `accounts/` 平级，见 03 §10D）。

---

## 接口 1：`inv backup`

### 子命令签名

```text
inv backup create [--account default] [--mode lite|full] [--destination DIR] [--json]
inv backup list   [--account default] [--json]
inv backup show   <backup_id>
inv backup restore --from <backup_id> [--dry-run] [--yes]
inv backup prune  [--keep 8]
```

### 参数表

| 参数 | 必填 | 默认 | 说明 |
|---|---|---|---|
| `--mode` | 否 | `lite` | `lite` = state + db；`full` = 再加 library + reports |
| `--destination` | 否 | 自动 | 自定义输出目录；默认 `{BACKUP_ROOT}/{account}/{YYYYMMDD_HHMMSS}` |
| `--from` | restore 必填 | — | 备份 id（目录名时间戳） |
| `--dry-run` | 否 | false | 仅打印将覆盖的目录 |
| `--yes` | restore 必填 | false | 确认覆盖当前 account |
| `--keep` | 否 | 8 | prune 保留最新 N 份 |

### lite 备份内容

```text
{backup_id}/
├── manifest.json
├── state/          # portfolio.yaml、watchlist.yaml、risk_rules 等
└── db/             # account.sqlite（create 前 WAL checkpoint）
```

### 成功 stdout 示例

```text
backup OK: id=20260527_143022 mode=lite path=./data/_backups/default/20260527_143022
```

```text
restore OK: from=20260527_143022 pre_restore=pre_restore_20260527_150001
请运行: inv doctor --scope all
```

### 失败 stdout 示例

```text
Error: restore 将覆盖当前 account 数据；须加 --yes 确认（建议先 inv backup create）
```

### restore 行为

1. 无 `--yes` → **拒绝**（防误覆盖）
2. 有 `--yes` → 先自动 `pre_restore_*` lite 快照，再覆盖 `state/`、`db/`（full 模式含 library/reports）
3. 完成后**必须**跑 `inv doctor --scope all`

### 手动验证步骤

1. 创建备份：

```powershell
.\bin\inv.exe backup create
.\bin\inv.exe backup list
.\bin\inv.exe backup show 20260527_143022
```

2. 确认 manifest：

```powershell
Get-Content .\data\_backups\default\20260527_143022\manifest.json
```

3. 预览 restore：

```powershell
.\bin\inv.exe backup restore --from 20260527_143022 --dry-run
```

4. 真 restore（仅在测试 account 或已确认时）：

```powershell
.\bin\inv.exe backup restore --from 20260527_143022 --yes
.\bin\inv.exe doctor --scope all
```

---

## 接口 2：`inv mcp` / `inv serve`

### 签名

```text
inv mcp   [--account default]
inv serve [--account default]    # MVP-1 等价于 inv mcp（stdio）
```

启动 **MCP stdio server**，供 Cursor / Claude Desktop 等通过 JSON-RPC 调用 tool。进程阻塞直到 IDE 关闭连接。

### Cursor 配置示例

项目或用户级 `.cursor/mcp.json`（路径按本机调整）：

```json
{
  "mcpServers": {
    "investment-assistant": {
      "command": "C:\\Users\\qs\\Desktop\\workspace\\investment-assistant\\bin\\inv.exe",
      "args": ["mcp"],
      "env": {
        "DATA_ROOT": "C:\\Users\\qs\\Desktop\\workspace\\investment-assistant\\data",
        "IA_ACCOUNT_ID": "default",
        "IA_CONFIG_ROOT": "C:\\Users\\qs\\Desktop\\workspace\\investment-assistant\\config"
      }
    }
  }
}
```

保存后重启 Cursor → Settings → MCP → 确认 `investment-assistant` 为绿色 Connected。

### MVP-1 注册的 9 个只读 tool

| # | tool | 简要说明 | 关键 input |
|---|---|---|---|
| 1 | `get_portfolio` | 读 portfolio.yaml | `code?`, `include_closed?` |
| 2 | `get_watchlist` | 读 watchlist.yaml | `state?` watching/removed/all, `code?` |
| 3 | `search_library` | L1 素材检索 | `query?`, `code?`, `limit?` |
| 4 | `get_library_item` | 单条素材 | **`lib_id`** 必填 |
| 5 | `search_journal` | 决策 journal 列表 | `code?`, `action_type?`, `limit?` |
| 6 | `get_journal` | 单条 journal + snapshot 摘要 | **`journal_id`** 必填 |
| 7 | `get_checklist_template` | checklist 默认 payload | **`checklist_type`** 必填 |
| 8 | `get_risk_rules` | M7 规则 + 启用 redlines | （无） |
| 9 | `check_position_against_rules` | 模拟 M7，不下结论 | `scenario`, `code`, `planned_position_pct_after` |

每个 tool 的 `description` 末尾含 **【边界】** 声明：仅返回客观数据，不构成投资建议，不得代替用户填写结论性字段。

### 禁止注册的写 tool（MVP-1）

`approve_checklist`、`promote_library_item`、`supplement_library`、`write_tags`、`update_portfolio` — 代码与测试均保证**未注册**。

### MCP 调用示例（IDE 内自然语言）

- 「用 get_portfolio 列出当前 holding」
- 「search_journal code=600519 action_type=buy limit=5」
- 「get_checklist_template checklist_type=add」

### 错误返回

- 参数缺失：tool result 为 error 文本，如 `lib_id 必填`
- 记录不存在：`get_library_item` / `get_journal` 返回 JSON `{"error":{"code":"not_found",...}}`（HTTP 200 式 MCP success + 业务 not_found）

### 手动验证步骤

1. 编译并确保 account 已初始化
2. 配置 `.cursor/mcp.json` 如上
3. Cursor 聊天：「调用 get_portfolio，include_closed=false」
4. 确认返回 `schema_version`、`positions` 与 `data/accounts/default/state/portfolio.yaml` 一致
5. 「search_journal code=600519 limit=3」→ 应含已 approve 的 journal 摘要

---

## 自动化验证

```powershell
go test ./internal/core/backup/... -v
go test ./internal/mcp/... -v
go test ./...
go build -o bin/inv.exe ./cmd/inv
```

关键测试：

- `TestCreateAndListLite` — backup create + manifest
- `TestRegistry_ReadOnlyOnly` — 恰好 9 tool、无写 tool
- `TestForbiddenWriteToolsNotEmpty` — 禁止列表非空

---

## 边界与已知限制

| 项 | 说明 |
|---|---|
| MCP 传输 | MVP-1 仅 **stdio**；无 HTTP/SSE 常驻（`inv serve` = `inv mcp`） |
| account 切换 | 通过环境变量 `IA_ACCOUNT_ID`；tool input 里的 `account_id` 字段**尚未**实现多 account 路由 |
| search_library | 未实现 04 schema 中的 `tag` / `content_type` 过滤（MVP-2） |
| search_journal | 未实现 `since` / `until` 日期过滤 |
| daily crawl | **未在本里程碑交付**；library crawl → candidate 属 MVP-2 / H8+ |
| 距上次 backup >7 天 warning | 03 O5B 计划在 submit 前提醒，**尚未**接入 checklist CLI |

---

## 故障排查

| 现象 | 含义 | 处理 |
|---|---|---|
| restore 拒绝 | 缺少 `--yes` | 先 `backup create`，再 `restore --yes` |
| restore 后 doctor 报错 | 备份本身脏或版本不匹配 | 换更早的 backup；或按 [H1](H1-portfolio与doctor.md) P 码修 |
| MCP 未连接 | inv.exe 路径或 env 错误 | 检查 mcp.json 绝对路径；终端手动 `.\bin\inv.exe mcp` 看 stderr |
| MCP tool 超时 | 单 tool >10s | 缩小 limit；检查 db 是否被其他进程锁 |
| approved checklist 无法 undo | 无 reject | 用 **restore** 回到 approve 前备份，或手修 journal/lot |

---

## 关联文档

- [03 §10D 备份策略](../03-数据架构与数据源.md)
- [04 §二十二 MCP tool schema](../04-技术架构.md)
- [05 H7 验收](../05-MVP-Roadmap.md)
- [07 接口索引](../07-接口与验证手册.md) §六
