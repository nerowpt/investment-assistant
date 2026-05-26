# 用户使用与验证手册（manual）

本目录是 **「怎么调用、怎么验收」** 的实操文档，与 `docs/01–06` 的设计文档互补。

- **设计文档**：Why / What / How（架构与字段字典）
- **本目录**：When you run what command, what you should see

总纲与索引：[../07-接口与验证手册.md](../07-接口与验证手册.md)

## 字段字典（怎么填）

| 文档 | 内容 |
|---|---|
| [ref-portfolio-yaml-fields.md](ref-portfolio-yaml-fields.md) | **portfolio.yaml 每个字段** + 枚举可选值 + 示例 |
| [ref-watchlist-yaml-fields.md](ref-watchlist-yaml-fields.md) | **watchlist.yaml 每个字段** + 枚举可选值 |
| [ref-sqlite-decision-tables.md](ref-sqlite-decision-tables.md) | **journals / lots 等 SQLite 表** + SQL 示例 |
| [config/portfolio.yaml.example](../../config/portfolio.yaml.example) | 带行内中文注释的模板 |
| [internal/.../schema/README.md](../../internal/core/store/sqlstore/schema/README.md) | Go struct 与表映射（IDE 悬停） |

## 阅读顺序（新手上手）

1. [00-环境与快速开始.md](00-环境与快速开始.md)
2. [ref-portfolio-yaml-fields.md](ref-portfolio-yaml-fields.md) — **改 portfolio 前必读**
3. [H0-骨架与迁移.md](H0-骨架与迁移.md)
4. [H1-portfolio与doctor.md](H1-portfolio与doctor.md)
5. [ref-sqlite-decision-tables.md](ref-sqlite-decision-tables.md) — 理解 journals/lots
6. 后续按 [07 §五](../07-接口与验证手册.md#五功能项索引与状态) 里程碑追加

## 给协作者

每合并一个功能 PR，必须：

1. 新建或更新对应 `H*-*.md`
2. 更新 [07 §五](../07-接口与验证手册.md#五功能项索引与状态) 状态表
3. 手册中的命令必须在你本机复制粘贴可跑通
