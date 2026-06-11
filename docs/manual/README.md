# 用户使用与验证手册（manual）

本目录是 **「怎么调用、怎么验收」** 的实操文档，与 `docs/01–06` 的设计文档互补。

- **设计文档**：Why / What / How（架构与字段字典）
- **本目录**：When you run what command, what you should see

总纲与索引：[../07-接口与验证手册.md](../07-接口与验证手册.md)

## 字段字典（怎么填）

| 文档 | 内容 |
|---|---|
| [ref-checklist-types.md](ref-checklist-types.md) | **`--type` 七种 checklist** + 状态机 + 命令速查 |
| [ref-portfolio-yaml-fields.md](ref-portfolio-yaml-fields.md) | **portfolio.yaml 每个字段** + 枚举可选值 + 示例 |
| [ref-watchlist-yaml-fields.md](ref-watchlist-yaml-fields.md) | **watchlist.yaml 每个字段** + 枚举可选值 |
| [ref-sqlite-decision-tables.md](ref-sqlite-decision-tables.md) | **journals / lots 等 SQLite 表** + SQL 示例 |
| [examples/checklist-buy-600519.json](examples/checklist-buy-600519.json) | **buy checklist payload 完整示例** |
| [examples/checklist-add-600519.json](examples/checklist-add-600519.json) | **add checklist payload 完整示例** |
| [examples/checklist-sell-600519.json](examples/checklist-sell-600519.json) | **sell checklist payload 完整示例** |
| [examples/checklist-exception-hard.json](examples/checklist-exception-hard.json) | **hard_block 例外 JSON 示例** |
| [config/portfolio.yaml.example](../../config/portfolio.yaml.example) | 带行内中文注释的模板 |
| [internal/.../schema/README.md](../../internal/core/store/sqlstore/schema/README.md) | Go struct 与表映射（IDE 悬停） |

## 阅读顺序（新手上手）

1. [00-环境与快速开始.md](00-环境与快速开始.md)
2. **[01-操作流程总览.md](01-操作流程总览.md)** — **先做什么、再做什么（含 buy→add/sell 分叉）**
3. [ref-checklist-types.md](ref-checklist-types.md) — draft `--type` 怎么选
4. [ref-portfolio-yaml-fields.md](ref-portfolio-yaml-fields.md) — **改 portfolio 前必读**
5. [H0-骨架与迁移.md](H0-骨架与迁移.md)
6. [H1-portfolio与doctor.md](H1-portfolio与doctor.md)
7. [ref-sqlite-decision-tables.md](ref-sqlite-decision-tables.md) — 理解 journals/lots
8. [H2-library归纳流水线.md](H2-library归纳流水线.md) · [H3-data-worker-gRPC.md](H3-data-worker-gRPC.md)
9. [H4-checklist与M7.md](H4-checklist与M7.md)
10. [H5-checklist-approve.md](H5-checklist-approve.md)
11. [H6-sell-FIFO.md](H6-sell-FIFO.md)
12. [H7-backup-MCP.md](H7-backup-MCP.md) · [H8-web-api与前端.md](H8-web-api与前端.md) · [H8.1-选股看板.md](H8.1-选股看板.md) · [H8.2-研究档案.md](H8.2-研究档案.md) · [H8.3-复盘工作台.md](H8.3-复盘工作台.md)
13. [MVP1-验收跑通.md](MVP1-验收跑通.md)
14. 后续按 [07 §六](../07-接口与验证手册.md#六功能项索引与状态) 里程碑追加

## 给协作者

每合并一个功能 PR，必须：

1. 新建或更新对应 `H*-*.md`
2. 若有新枚举/选项 → 更新或新建 `ref-*.md`（见 `.cursor/rules/manual-docs.mdc`）
3. 若改变用户主路径 → 更新 [01-操作流程总览.md](01-操作流程总览.md)
4. 更新 [07 §六](../07-接口与验证手册.md#六功能项索引与状态) 状态表
5. 手册中的命令必须在你本机复制粘贴可跑通
