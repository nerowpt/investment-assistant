# 06 - 架构 Review 决议（2026-05-22）

> 文档版本：v0.1（H1 进行中第一次集中 Review 决议）
> 最后修订：2026-05-22
> 关联文档：[00-讨论历史与决策日志](00-讨论历史与决策日志.md) · [01-产品定位与思路](01-产品定位与思路.md) · [02-核心场景与功能边界](02-核心场景与功能边界.md) · [03-数据架构与数据源](03-数据架构与数据源.md) · [04-技术架构](04-技术架构.md) · [05-MVP-Roadmap](05-MVP-Roadmap.md)

---

## 一、本次 Review 的目的与定位

本文档落档 2026-05-22 一轮"资深 PM + 资深架构师 + 资深投资人员"三视角集中 Review 的结论，**所有条目已经过用户当面确认采纳**。

本文档不替代既有 01–05 文档的章节权威性，而是：

1. **决议清单**：明确每条改动的最终结论与优先级（P0/P1/P2）。
2. **回填索引**：列出"本次 Review 影响 01–05 哪些章节"，便于追溯。
3. **代码迁移台账**：列出 H1–H7 期间必须落地的代码改动与验收方式。
4. **文档纪律约束**：明确 H1 起的"文档冻结 + 不超前 2H"规则，避免文档继续膨胀。

> ⚠️ 后续如对本次决议条目反悔或调整，必须在 `00-讨论历史与决策日志.md` 追加新决策并显式更新本文档版本号，**不得**直接改本文档原决议文字。

---

## 二、Review 总体结论

| 维度 | 评分 | 一句话评价 |
|---|---|---|
| 产品定位与思路 | ⭐⭐⭐⭐⭐ | 方向（散户飞行日志 + 个人投资知识资产）选得对，对 AI 边界的硬约束远超同类项目 |
| 产品设计落地性 | ⭐⭐⭐⭐ | 主骨架可生产，但总工期 7–9 月对一人项目偏激进，必须靠 dogfood 校准 |
| 设计拓展性 | ⭐⭐⭐⭐ | 分期合理；事件层、模块包结构、L1 媒体类型范围有微调空间 |
| 数据架构 | ⭐⭐⭐⭐⭐ | 本项目最强项：SoT、tier、快照冻结、account 隔离、决策粒度均到位 |
| 技术架构 | ⭐⭐⭐⭐ | 分层与 Kratos 留白做得好；Python worker 与 cron 长驻引入复杂度需控制 |
| 当前代码骨架 | ⭐⭐⭐ | H0/H1 对齐文档，但已出现 3 处早期漂移，须在 H1 末前修正 |

**最大风险**：不是技术，而是文档完整度走在了实现完整度的 7 倍之前；7–9 月业余工期与 M5 复盘价值兑现节奏不匹配。**唯一缓解办法是强制 dogfood + 文档冻结**。

---

## 三、产品 / 投资视角决议

### D1 — 用户演进路径（P1，影响 docs/01 §二）

| 项 | 决议 |
|---|---|
| 短期 | **仅一人使用**（与 Q1 一致） |
| 中期（MVP-1 dogfood ≥ 4 周稳定后） | 向 **家人 / 朋友** 开放（2–3 个 account，本机/NAS） |
| 长期（MVP-2 完成后） | 演进为 **SaaS 产品**（多租户、Web、订阅） |

**影响**：
- `docs/01` §二用户画像追加"演进路径"小节。
- `docs/03` §二.9 多 account 隔离设计**保留**（已是正确选择，无需调整）。
- Kratos / `cmd/ia-api` 引入时机不变（MVP-2 H8）；但 H8 之前应明确 SaaS 化所需的最小变更点（auth/多用户/计费基线），写入 `docs/05` §六 H8+ 预研项。

### D2 — MVP-1 定量成功标准（P1，影响 docs/05 §3.1）

在 `docs/05` §3.1 现有 E1–E10 之外，**追加定量门槛**：

| # | 指标 | 目标值 | 验收方式 |
|---|---|---|---|
| Q1 | Journal 累计条数 | **≥ 20 笔**（buy/add/sell/inspect 至少各 1 笔） | `inv journal list` 计数 |
| Q2 | 完整 quarter 复盘报告 | **≥ 1 份**，含用户手填 `confirmed_patterns` | `inv review` 输出 + 人工签收 |
| Q3 | tier C/D 在"主要依据"占比 | **从初始未控降到 < 30%** | 复盘报告统计 |
| Q4 | 风险护栏例外说明字数 | hard block 例外平均 ≥ 80 字（防"形同虚设"） | `inv risk exceptions stats` |
| Q5 | 一次完整 dogfood 周期 | **≥ 4 周**真实使用且未推翻 MVP-1 架构 | 用户书面确认 |

只要 Q1–Q5 任一未达成，**不得**启动 MVP-2 任何里程碑。

### D3 — emotion_tag 反作弊机制（P1，影响 docs/01 §三、docs/02 §16、docs/05 H5/H6）

**问题**：用户在 `excited / FOMO` 时极可能不诚实自报情绪，使 emotion_tag 数据失真。

**决议**（MVP-1 实现，MVP-2 强化）：

1. **事后回溯字段**（MVP-1 H6 起）：sell checklist 在卖出后 30/90 天，调度器自动比对涨跌：
   - 若卖出后 30 天涨幅 ≥ 20%：触发"事后情绪回溯提示"，要求用户重选一次"当时实际是 X"。
   - 比对结果记入 `emotion_retrospect` 字段（与原 `emotion_tag` 并列，**不覆盖**）。
2. **复盘统计**（M5 / MVP-2A）：`emotion_tag` vs `emotion_retrospect` 的偏差率，作为"自我认知偏差画像"指标。
3. **MVP-1 不做**：实时弹窗、生理特征采集、对话情绪分析（后两者是隐私 / 工程黑洞）。
4. **schema 预留**：H1 在 `checklist_submissions.payload_json` 中预留 `emotion_retrospect: null` 字段位，避免 H6 再迁移。

### D4 — Dogfood 强制门禁（P1，影响 docs/05）

进入 MVP-2 任何里程碑前，**必须**完成一轮"真实历史交易 dogfood"：

1. **数据来源**：`docs/_baseline/02-当前持仓与个人逻辑.md` 中的真实持仓。
2. **路径**：完整跑通 import (S0.5) → S0 素材入库 → S4 巡检 → S6 复盘 至少一轮。
3. **失败标准**：用户走完后明确表达"不想再用 CLI 操作 / 流程过繁"。
4. **失败处置**：暂停 MVP-2，回到 MVP-1 简化主路径而非加新功能。

写入 `docs/05` §七 风险章节作为"软退路 + 硬门禁"。

### D5 — M7 hard / soft block 拦截差异显式化（P1，影响 docs/02 §18、personal_redlines.yaml.example、internal/core/domain/risk）

**问题**：当前 hard / soft 仅在 severity 字段分级，**拦截行为差异不明确**，存在两类极端：
- 拦截太松 → 例外说明形同虚设；
- 拦截太严 → 用户直接绕过系统去券商 App。

**决议**（写入 `docs/02` §18.7 与 H4 实现）：

| severity | 拦截行为 | exception_json 字段要求 |
|---|---|---|
| `hard` | **强拦截**：必须填写完整例外说明才能 Submit | `triggered_rule_id` + `exception_reason`（≥ 80 字）+ `expected_compensation` + `review_date` + **至少 1 条 S/A tier `library_item_id` 作为依据** + `confirm_text="我已知悉本次例外永久记入 Journal"` |
| `soft` | **软警示**：仅显示警告，提交时记录"warning ack"，不强制例外说明 | `acked: true` + `ack_note`（可选 ≥ 20 字） |

**违规绕过路径检测**（MVP-2 H14 复盘阶段）：扫描"hard block 后 7 天内的 Journal 缺口"，对照券商对账单提示用户"本月可能有未记录交易"。

### D6 — 叙事信号包 90 天 stale 软退路（P1 / MVP-2A，影响 docs/01 §十、docs/05 §6.2 H10）

**问题**：信号包月度自动生成，6 个月前的旧包会卡住今天的建仓，违反"产品减少注意力消耗"原则。

**决议**：

1. 信号包默认 **有效期 90 天**（`narrative_signal_packs.stale_at = generated_at + 90d`）。
2. 到期自动 `status = stale`，但**不删除**（仍可被 M5 错过模式扫描引用）。
3. 建仓时如果标的的信号包状态为 `stale`：
   - **不强制重新生成**信号包；
   - **不强制重新填阶段判断**；
   - 仅要求用户在 Checklist 中勾选 `stale_acknowledgement = true`，并写一句"我已确认 N 天前的旧信号已过时"。
4. 用户可主动 `inv narrative refresh <code>` 强制刷新，刷新后回到强拦截路径。

### D7 — `swap` 换股一等动作（P1，影响 docs/02 §16、docs/03 §10A）

**问题**：散户最常见的"卖 A 买 B"在心理上是一个决策，但当前 sell + buy 是两条独立 Journal，复盘时无法统计"换股对的胜率"。

**决议**：

1. 新增 `checklist_type = swap`，**逻辑上 = sell + buy 原子组合**：
   - 物理实现仍写 2 条 Journal（sell + buy），通过 `swap_pair_id` 关联。
   - swap checklist 在 Approve 阶段在单 SQL Tx 内创建两条 Journal。
2. swap 复盘指标：sell 端 vs buy 端的**相对收益**（30/90/365 天），写入 M5 报告。
3. **MVP-1 不实现** swap：只在 `docs/02` §16、`docs/03` §10A.1 表中**预留**类型位与字段位，避免 MVP-2 实现时迁移。
4. MVP-1 用户暂用"先 sell 再 buy 两次 Journal"+ 在 sell.payload 备注 `swap_target_code` 作为弱关联。

### D8 — 股息 / 分红 / 复权字段位（P1，影响 docs/03 §10A.6 lots）

**问题**：中长线持仓没有 dividend / bonus_share / split 处理，年化收益与 vs 沪深300 对比会失真。

**决议**：

1. **H1 yamlstore + sqlstore 设计期**就在 `lots` 表预留：
   - `dividends_received_decimal`（已收现金分红累计，单位元）
   - `adjusted_cost_basis_decimal`（前复权调整后成本价）
   - `corporate_actions_json`（送转 / 拆股 / 配股事件流水，JSON 数组）
2. **MVP-1 不计算**：字段位预留，数值由 H1 默认 `0 / null`；用户可手填。
3. **MVP-2 H13** 增强：调度器自动从 akshare 拉分红事件，触发 doctor 提示用户更新。
4. **金融精度强制**：所有金额字段在 Go 层用 `decimal`，永不走 float（参考 §四 D11）。

### D9 — L1 media_type 子集 doctor（P2，影响 docs/03 §9.5）

**问题**：当前 schema 字段已经覆盖 image/audio/video/structured，但 MVP-1 只启用 text/pdf/html/structured，存在误用风险。

**决议**：

- `docs/03` §9.5 表头加注："MVP-1 启用集 = `{text, pdf, html, structured}`；image / audio / video 字段已预留，MVP-1 写入将被 doctor 警告。"
- `inv doctor --scope library` 增加检查项：`media_type ∈ {audio, video, image}` 的 active 条目数 = 0，否则输出 warning（不 fatal）。

### D10 — FTS5 评估前置（P1，影响 docs/03 §9.10、docs/05 §6.2）

**问题**：FTS5 推到 MVP-2B（H12）会导致 library_items > 500 时检索体验崩塌，进而**让用户停止使用 L1**。L1 的复利价值随之归零。

**决议**：

1. **MVP-1 H2 末期**做一次"FTS5 接入难度评估"：
   - 标准：modernc.org/sqlite 是否原生支持 FTS5 + 接入工作量是否 ≤ 1 周。
2. 评估通过 → **FTS5 提前到 H2 完成**（MVP-1 范围内）。
3. 评估不通过 → FTS5 从 H12 提前到 **H8 之前**（MVP-2A 起点），不允许放到 H12。
4. MVP-1 期间 `inv library search` 即使无 FTS5，也必须支持：title LIKE + tags 过滤 + related_stocks 精确匹配 + tier 过滤 的组合查询。

---

## 四、架构 / 代码视角决议

### D11 — DECIMAL 金融精度（P0，影响 internal/core/doctor、所有 sqlstore 读写）

**问题**：`internal/core/doctor/portfolio.go` 当前 `var pct float64` + `decimal.NewFromFloat(pct)` 违反 T24（`docs/04` §10.4.3），浮点误差已混入 lot 比例求和。

**决议**：

1. **SQLite 读写**：所有 `DECIMAL` 逻辑类型字段一律走 `string` 通道：
   - 写入：`decimal.Decimal.String()` 后写入 TEXT 列（或 REAL → 但读取时不依赖精度）。
   - 读取：`SELECT CAST(col AS TEXT)` 或扫描为 `sql.NullString`，再 `decimal.NewFromString`。
2. **新增工具**：`internal/core/store/sqlstore/decimal_scan.go`（H1 内交付），统一封装：
   ```go
   func ScanDecimal(row *sql.Row, dest *decimal.Decimal) error
   func ScanDecimalRows(rows *sql.Rows, dest *decimal.Decimal) error
   ```
3. **现存代码立即修正**：`internal/core/doctor/portfolio.go` 的 `checkOpenLotsPct`。
4. **测试覆盖**：H1 PR 必须含 `0.1 + 0.2` 边界用例的 doctor 测试。

### D12 — PortfolioPosition.Monitoring 字段补齐（P0，影响 internal/core/store/yamlstore/portfolio.go）

**问题**：`config/portfolio.yaml.example` 包含 `monitoring:` 子结构，但 Go struct 没有对应字段，load → save 会**整段丢失**。

**决议**（H1 立刻实施）：

1. 在 `PortfolioPosition` 添加：
   ```go
   Monitoring *PositionMonitoring `yaml:"monitoring,omitempty"`
   ```
2. 新增类型 `PositionMonitoring`，字段对齐 example：
   - `LastInspectionID string`
   - `LastInspectionAt string`
   - `NextInspectionDue string`
   - `Classification string`（枚举 02 §16 巡检 Checklist）
   - `PlannedAction string`
3. 配套单元测试：load → save → 二次 load，断言 `Monitoring` 完整保留。

### D13 — yamlstore 接口化（P0，影响 internal/core/store/yamlstore、internal/core/app）

**问题**：`docs/04` §20.3.1 明确定义 `PortfolioStore interface`，当前实现为包级函数，**无法 mock**，H4 起阻塞集成测试。

**决议**（H1 末前完成）：

1. 每个 YAML 文件一个 Store 接口 + File 实现。
   - 为避免 `account` ↔ `yamlstore` 循环依赖，接口直接接收 **文件路径 string** 而非 `*account.Context`；上层 wiring（cli/app）从 AccountContext 拼装路径后传入。这与 `docs/04` §20.3.1 原始签名（`*account.Context`）有一处取舍调整，原因记入此处不重复回填 04。

   ```go
   type PortfolioStore interface {
       Load(ctx context.Context, path string) (*Portfolio, error)
       Save(ctx context.Context, path string, p *Portfolio) error
   }
   func NewFilePortfolioStore() PortfolioStore { ... }
   func NewMemoryPortfolioStore() PortfolioStore { ... }
   ```

2. 覆盖范围：**H1 仅交付 PortfolioStore**；Watchlist / Candidates / RiskRules / PersonalRedlines / ControlledTags 在 H1 后续批次按 D19"不超前 2H"原则跟进，需在 H4 启动前完成全部接口化。
3. 包级 `LoadPortfolio` / `SavePortfolio` **保留为向后兼容 alias**（当前 doctor 直接使用），H2 起 cli/app 新增逻辑必须走接口。
4. 测试：H1 PR 含 `MemoryPortfolioStore` 内存实现 + 深拷贝隔离测试（供 H4–H5 use case 测试）。

### D14 — 用例输出端口 `Result` 结构体（P2，影响 internal/core/app、internal/cli、internal/mcp）

**问题**：当前 cli 直接 fmt.Println 输出，H7 加 MCP / 未来加 HTTP 会重复实现一遍格式化逻辑。

**决议**（H4 启动时落地）：

1. 每个 use case 返回结构化 `Result`：
   ```go
   type ApproveBuyResult struct {
       JournalID   string
       LotID       string
       SnapshotID  string
       YAMLPatched []string  // 实际写入的 yaml 文件相对路径
       Warnings    []RuleHit
       SyncRepairs []string  // 若失败的修复任务 id
   }
   ```
2. 渲染层（cli / mcp / 未来 http）各自实现 `func RenderApproveBuy(r *ApproveBuyResult) string`。
3. cli 默认人类可读；`--json` flag 输出 `json.Marshal(r)`。
4. MCP 返回原结构（已是 JSON）。
5. **MVP-1 不引入** wire / DI 框架，手写构造即可。

### D15 — 模块按业务域拆包（P1，影响 internal/core/app/）

**问题**：当前 `internal/core/app/` 计划扁平存放 checklist / doctor / ports；H5 起 ApproveHandler 横切 M1–M7，单文件会 > 1500 行。

**决议**（H4 启动时落地）：

```text
internal/core/app/
├── ports.go              # 全局依赖接口（Worker / Notify / Events）
├── doctor/               # 既有
├── checklist/            # use case 入口（Submit/Approve/Reject）
│   ├── service.go
│   ├── submit.go
│   ├── approve.go
│   └── handlers/
│       ├── buy.go
│       ├── add.go
│       ├── sell.go
│       ├── inspect.go
│       ├── review.go
│       ├── watch.go
│       └── import.go
├── risk/                 # M7 模拟与拦截
├── library/              # L1 use case（ingest/promote/search）
├── journal/              # 查询 / 搜索
└── events/               # 见 D16
```

handler 注册到 `Registry`，按 `checklist_type` 分发。

### D16 — 事件 / 通知极薄抽象（P0/P1，影响 internal/core/app/events、MVP-2 H16）

**问题**：监控、调度、报告、推送将散在多处；MVP-2 加邮件 / Server 酱时改动面大。

**决议**：

1. **H7 完成前**新增 `internal/core/app/events`：
   ```go
   type DomainEvent interface {
       Type() string
       OccurredAt() time.Time
       Payload() map[string]any
   }
   type Publisher interface {
       Publish(ctx context.Context, e DomainEvent) error
   }
   ```
2. MVP-1 唯一实现：`LogPublisher`（写 SQLite `monitor_events`）+ stdout 调试模式。
3. MVP-2 H16 新增：`EmailPublisher` / `ServerChanPublisher`，业务层零改动。
4. 事件来源：M4 监控触发、M7 例外提交、调度器任务完成、approve 成功。

### D17 — Python worker 必要性重评估（P1，影响 docs/04 §七 / §二十一、docs/05 H3）

**问题**：Go ↔ Python 双向 gRPC + Windows 端口文件 + Unix socket + supervisor + retry / backoff 对一人项目偏重。

**决议**（H3 启动前完成评估）：

1. **数据源清单梳理**：在 `docs/03` §10D.7 P0 数据源旁加列 `go_native_possible: yes/no`：
   - akshare 底层 HTTP 接口能否直接 Go 调用；
   - 雪球 / 东财 API 是否纯 HTTP。
2. **决策门槛**：
   - 若 P0 数据源 ≥ 70% 可 Go 原生 → **Python worker 推迟到 MVP-2**，MVP-1 用 Go HTTP + 用户手动粘贴公告原文。
   - 若 < 70% → 保持 H3 计划，但**仅实现 FetchQuote + FetchAnnouncements 两个 RPC**，FetchValuation / FetchMarketSnapshot / ExtractDocument 推到 MVP-2。
3. 评估结论必须写入 `docs/00` 与本文档 §六 H3 变更点。

### D18 — 删除 MVP-1 cron 长驻（P2，影响 docs/04 §十四、docs/05 H7）

**决议**：

1. MVP-1 `inv serve` **不内嵌 robfig/cron**，仅保留 MCP stdio 模式。
2. daily crawl 改为**用户手动触发**：`inv library crawl --since=7d`。
3. 调度器与 cron 推到 MVP-2 H7+（H16 之前）。
4. Windows 用户避免后台进程关机/休眠/重启的稳定性黑洞。

### D19 — 文档冻结 + 不超前 2H 规则（P0，影响所有 docs/）

**决议**（H1 立刻生效）：

1. **冻结目标**：H1 结束前 **`docs/03` / `docs/04` 不再做细节字段或章节修改**。
2. **新需求处理**：H1–H5 期间任何新发现的字段需求、schema 调整、接口变更，**只在 `docs/00` 决策日志追加**，**不**回填 `docs/03 / 04`。
3. **回填时机**：H5 跑通后（buy approve 端到端 pass），统一回填 `docs/03 / 04` 并发布 v 次版本号。
4. **超前限制**：任何 docs/ 文件提到的功能 / 字段，**不得超前当前已完成的最高里程碑 + 2H**。例如当前 H1 进行中，docs/ 可以谈到 H3 内容，但不应详写 H5+ 字段。
5. **执行检查**：每个 H 完成验收时，新增一项"文档同步检查"：
   - 是否有"未实现但已在 docs 中详写"的字段被引用到代码？
   - 若有，要么实现要么从 docs 删除，**不允许文档欠债**。
6. **唯一例外**：`docs/06`（本文档）和 `docs/00` 决策日志可以超前——它们的职责就是记录未实现的决策。

---

## 五、本次 Review 影响的文档章节索引

> 此表用于回填时核对；本次 Review 后第一次 H 完成的回填批次应**优先**覆盖此处所有条目。

| 决议 | 影响文件 | 章节 / 文件位置 | 处理方式 |
|---|---|---|---|
| D1 | docs/01 | §二 用户画像 | 追加"演进路径"小节 |
| D1 | docs/05 | §六 MVP-2 | H8+ 加 SaaS 预研项 |
| D2 | docs/05 | §3.1 DoD | 追加 Q1–Q5 定量门槛 |
| D3 | docs/01 | §三.2 决策权与结论权 | 加 emotion_retrospect 段 |
| D3 | docs/02 | §16 卖出 Checklist | 加 emotion_retrospect 字段位 |
| D4 | docs/05 | §七 风险 | 加 dogfood 软退路硬门禁 |
| D5 | docs/02 | §18.7 例外说明 | hard / soft 拦截差异表 |
| D5 | config/personal_redlines.yaml.example | 全文 | 注释说明拦截差异 |
| D6 | docs/01 | §七.7.4 + §十 | 加 90 天 stale 软退路 |
| D6 | docs/05 | §6.2 H10 | 验收清单加 stale 字段 |
| D7 | docs/02 | §16 表头 | 加 swap 类型位 |
| D7 | docs/03 | §10A.1 | 加 swap 行 + swap_pair_id |
| D8 | docs/03 | §10A.6 lots | 追加 3 个字段位 |
| D9 | docs/03 | §9.5 表头 | MVP-1 启用集说明 |
| D10 | docs/03 | §9.10 表 | FTS5 评估前置 |
| D10 | docs/05 | §6.2 H12 | 触发条件改为"H2 评估未过" |
| D11 | docs/04 | §10.4.3 | 加 decimal_scan.go 约定 |
| D12 | docs/03 | §10B.2 portfolio | 加 monitoring 子结构正文 |
| D13 | docs/04 | §20.3.1 | 标注"MVP-1 H1 末完成接口化" |
| D14 | docs/04 | §六 + §二十二 | 加 Result 结构体约定 |
| D15 | docs/04 | §五 + §六 | 调整 internal/core/app 结构图 |
| D16 | docs/04 | §六 + §十四 | 加 events 包 |
| D17 | docs/04 | §七 + §二十一 | H3 启动前评估门槛 |
| D17 | docs/03 | §10D.7 | 加 go_native_possible 列 |
| D18 | docs/04 | §十四 | 删除 MVP-1 cron |
| D18 | docs/05 | H7 验收 | 删除 daily crawl |
| D19 | docs/05 | §九 | 加文档冻结规则 |

---

## 六、代码迁移台账（按 H 顺序）

### H1（进行中）必须完成

| ID | 决议 | 文件 | 验收 |
|---|---|---|---|
| H1-C1 | D11 | `internal/core/store/sqlstore/decimal_scan.go` 新增 | 含 `0.1 + 0.2` 测试用例 |
| H1-C2 | D11 | `internal/core/doctor/portfolio.go` 改用 ScanDecimal | doctor 旧 float 路径删除 |
| H1-C3 | D12 | `internal/core/store/yamlstore/portfolio.go` 加 Monitoring | round-trip 测试通过 |
| H1-C4 | D13 | yamlstore 全量接口化 + Memory 实现 | H4 mock 可用 |
| H1-C5 | D3 | `checklist_submissions.payload_json` schema 预留 `emotion_retrospect: null` | H1 yamlstore PR 内说明 |
| H1-C6 | D8 | `lots` 表 schema 加 3 字段位（dividends_received / adjusted_cost_basis / corporate_actions_json） | migration 002 或并入 001 时机决议 |

> **C6 时机讨论**：原则上 `001_initial.up.sql` 不应再改（T20）。本决议**例外**允许 H1 期间因尚未在生产数据使用而直接修改 001；但必须在 `migrations/README.md` 注明"H1 期间 001 仍可微调，MVP-1 发布后冻结"。

### H2 完成时

| ID | 决议 | 内容 |
|---|---|---|
| H2-C1 | D10 | FTS5 接入难度评估 + 决定是否本 H 完成 |
| H2-C2 | D9 | doctor 新增 media_type 子集警告 |

### H3 启动前

| ID | 决议 | 内容 |
|---|---|---|
| H3-C1 | D17 | Python worker 必要性评估报告（写入 docs/00） |

### H4 启动时

| ID | 决议 | 内容 |
|---|---|---|
| H4-C1 | D14 | 引入 Result 结构体 + json 渲染 |
| H4-C2 | D15 | `internal/core/app` 按业务域拆子包 |
| H4-C3 | D5 | M7 hard/soft 差异拦截 + exception_json schema |
| H4-C4 | D3 | emotion_tag 二次确认 UI（CLI 文案） |

### H5 完成时

| ID | 决议 | 内容 |
|---|---|---|
| H5-C1 | D19 | 启动 docs/03 + docs/04 回填批次 |

### H6 起

| ID | 决议 | 内容 |
|---|---|---|
| H6-C1 | D3 | sell checklist 30/90 天事后回溯调度器 |

### H7 完成前

| ID | 决议 | 内容 |
|---|---|---|
| H7-C1 | D16 | 事件层抽象 + LogPublisher |
| H7-C2 | D18 | 删除 MVP-1 cron / 长驻 |

### MVP-2

| ID | 决议 | 内容 |
|---|---|---|
| MV2-C1 | D6 | 信号包 90 天 stale 实现 |
| MV2-C2 | D7 | swap checklist 实现 |
| MV2-C3 | D3 | emotion_retrospect 统计与画像 |

---

## 七、本文档变更日志

| 版本 | 日期 | 变更 |
|---|---|---|
| v0.1 | 2026-05-22 | 集中 Review 决议落档：D1–D19，覆盖产品/投资/架构/代码四视角；文档冻结规则起算 |
