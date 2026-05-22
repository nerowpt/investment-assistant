-- MVP-1 初始 schema（对齐 docs/03 §九、§十A 与 docs/04 §二十）
-- 版本：1；每 account 独立 assistant.sqlite
--
-- 【方言】本文件为 SQLite DDL，不可原样在 MySQL 执行。
-- 【逻辑类型】字段注释 `-- logical: TYPE` 表示跨库语义；物理类型为 SQLite 存储类。
-- 映射表见 docs/04-技术架构.md §10.4（STRING_ID / JSON / TIMESTAMP / DECIMAL 等）。
--
-- 约定：
--   STRING_ID  → SQLite TEXT  ; MySQL VARCHAR(64)
--   JSON       → SQLite TEXT  ; MySQL JSON
--   TIMESTAMP  → SQLite TEXT ISO8601
--   DECIMAL    → SQLite REAL  ; MySQL DECIMAL(18,6) — Go 层须用 decimal，禁 float 比较
--   BOOL       → SQLite INTEGER 0/1

-- ---------------------------------------------------------------------------
-- 基础设施
-- ---------------------------------------------------------------------------

-- 日序 ID 生成（T9）：prefix 如 cs / j / lot
CREATE TABLE IF NOT EXISTS id_sequences (
    prefix      TEXT NOT NULL,           -- ID 前缀，如 cs、j、lot
    seq_date    TEXT NOT NULL,           -- 日期 YYYYMMDD
    next_seq    INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (prefix, seq_date)
);

-- SQL 已提交但 YAML 未写完的修复队列（T5）
CREATE TABLE IF NOT EXISTS sync_repairs (
    id                        TEXT PRIMARY KEY,  -- logical: STRING_ID
    checklist_submission_id   TEXT,
    journal_id                TEXT,
    yaml_files_json           TEXT NOT NULL,  -- 待写文件清单 JSON
    error_message             TEXT NOT NULL,
    status                    TEXT NOT NULL DEFAULT 'pending',  -- pending | resolved | aborted
    created_at                TEXT NOT NULL,
    resolved_at               TEXT
);

CREATE INDEX IF NOT EXISTS idx_sync_repairs_status ON sync_repairs (status);

-- ---------------------------------------------------------------------------
-- 决策流水（§10A）
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS checklist_submissions (
    id                          TEXT PRIMARY KEY,
    checklist_type              TEXT NOT NULL,
    code                        TEXT,
    name                        TEXT,
    payload_json                TEXT NOT NULL,
    payload_schema_version      INTEGER NOT NULL DEFAULT 1,
    status                      TEXT NOT NULL,  -- draft | submitted | approved | rejected
    submitted_by                TEXT NOT NULL DEFAULT 'user',
    tier_acknowledgement        INTEGER,
    emotion_self_check          TEXT,
    risk_guardrail_result_json  TEXT NOT NULL DEFAULT '{}',
    exception_json              TEXT,
    linked_library_ids_json     TEXT,
    generated_journal_id        TEXT,
    generated_inspection_id     TEXT,
    generated_review_id         TEXT,
    created_at                  TEXT NOT NULL,
    submitted_at                TEXT,
    approved_at                 TEXT
);

CREATE INDEX IF NOT EXISTS idx_checklist_type_code_submitted ON checklist_submissions (checklist_type, code, submitted_at);
CREATE INDEX IF NOT EXISTS idx_checklist_status ON checklist_submissions (status);
CREATE INDEX IF NOT EXISTS idx_checklist_generated_journal ON checklist_submissions (generated_journal_id);

CREATE TABLE IF NOT EXISTS journals (
    id                      TEXT PRIMARY KEY,
    action_type             TEXT NOT NULL,
    code                    TEXT,
    name                    TEXT,
    checklist_submission_id TEXT,
    data_snapshot_id        TEXT,
    payload_json            TEXT NOT NULL,
    summary                 TEXT,
    lot_id                  TEXT,
    created_at              TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_journals_code_created ON journals (code, created_at);
CREATE INDEX IF NOT EXISTS idx_journals_action_type ON journals (action_type);
CREATE INDEX IF NOT EXISTS idx_journals_checklist ON journals (checklist_submission_id);

CREATE TABLE IF NOT EXISTS data_snapshots (
    id              TEXT PRIMARY KEY,
    journal_id      TEXT NOT NULL,
    snapshot_json   TEXT NOT NULL,
    schema_version  INTEGER NOT NULL DEFAULT 1,
    created_at      TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_snapshots_journal ON data_snapshots (journal_id);

CREATE TABLE IF NOT EXISTS lots (
    id                      TEXT PRIMARY KEY,
    code                    TEXT NOT NULL,
    name                    TEXT,
    journal_id              TEXT NOT NULL,
    action_type             TEXT NOT NULL,
    position_type           TEXT NOT NULL,
    open_at                 TEXT NOT NULL,
    close_at                TEXT,
    initial_pct             REAL NOT NULL,   -- logical: DECIMAL（仓位比例）
    current_pct             REAL NOT NULL,   -- logical: DECIMAL
    cost_basis              REAL NOT NULL,   -- logical: DECIMAL（成本价）
    shares                  REAL,            -- logical: DECIMAL
    status                  TEXT NOT NULL,  -- open | partial | closed
    linked_buy_journal_id   TEXT,
    -- 复权与分红（docs/06 §D8；MVP-1 字段位预留，MVP-2 H13 起填充）
    dividends_received       REAL,           -- logical: DECIMAL；已收现金分红累计
    adjusted_cost_basis      REAL,           -- logical: DECIMAL；前复权调整后成本价（NULL = 用 cost_basis）
    corporate_actions_json   TEXT,           -- logical: JSON；送转/拆股/配股事件流水
    created_at              TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_lots_code_status ON lots (code, status);
CREATE INDEX IF NOT EXISTS idx_lots_journal ON lots (journal_id);

CREATE TABLE IF NOT EXISTS lot_allocations (
    id                    TEXT PRIMARY KEY,
    sell_journal_id       TEXT NOT NULL,
    lot_id                TEXT NOT NULL,
    allocated_pct         REAL NOT NULL,   -- logical: DECIMAL
    cost_basis_at_sale    REAL NOT NULL,   -- logical: DECIMAL
    proceeds_pct          REAL,            -- logical: DECIMAL
    realized_return_pct   REAL,            -- logical: DECIMAL
    match_method          TEXT NOT NULL,   -- recommended_fifo | user_adjusted
    user_confirmed        INTEGER NOT NULL DEFAULT 1,
    created_at            TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_lot_alloc_sell ON lot_allocations (sell_journal_id);
CREATE INDEX IF NOT EXISTS idx_lot_alloc_lot ON lot_allocations (lot_id);

CREATE TABLE IF NOT EXISTS risk_exceptions (
    id                      TEXT PRIMARY KEY,
    severity                TEXT NOT NULL,   -- warning | hard_block
    rule_source             TEXT NOT NULL,
    rule_id                 TEXT NOT NULL,
    checklist_submission_id TEXT NOT NULL,
    journal_id              TEXT,
    exception_reason        TEXT,
    expected_compensation   TEXT,
    review_date             TEXT,
    outcome_note            TEXT,
    created_at              TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_risk_exc_checklist ON risk_exceptions (checklist_submission_id);

CREATE TABLE IF NOT EXISTS inspection_records (
    id                      TEXT PRIMARY KEY,
    checklist_submission_id TEXT NOT NULL,
    code                    TEXT NOT NULL,
    inspection_type         TEXT NOT NULL,
    linked_buy_journal_id   TEXT,
    fact_update_summary_json TEXT,
    user_judgment_json      TEXT NOT NULL,
    report_path             TEXT,
    created_at              TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS review_reports (
    id                      TEXT PRIMARY KEY,
    checklist_submission_id TEXT NOT NULL,
    review_type             TEXT NOT NULL,
    period_start            TEXT NOT NULL,
    period_end              TEXT NOT NULL,
    stats_json              TEXT,
    user_judgment_json      TEXT NOT NULL,
    report_path             TEXT,
    created_at              TEXT NOT NULL
);

-- ---------------------------------------------------------------------------
-- L1 研究素材（§九）
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS library_candidates (
    id                          TEXT PRIMARY KEY,
    status                      TEXT NOT NULL,
    source_entry                TEXT NOT NULL,
    title                       TEXT NOT NULL,
    source                      TEXT NOT NULL,
    tier                        TEXT NOT NULL,
    timestamp                   TEXT NOT NULL,
    author                      TEXT,
    content_type                TEXT,
    media_type                  TEXT,
    related_stocks_json         TEXT,
    tags_json                   TEXT,
    dedup_key                   TEXT NOT NULL UNIQUE,
    staging_path                TEXT,
    canonical_url               TEXT,
    extract_json                TEXT,
    summary_draft               TEXT,
    similarity_json             TEXT,
    match_tier                  TEXT,
    resolution                  TEXT,
    resolution_target_item_id   TEXT,
    expires_at                  TEXT NOT NULL,
    promoted_library_item_id    TEXT,
    dismissed_reason            TEXT,
    created_at                  TEXT NOT NULL,
    updated_at                  TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_library_candidates_status_expires ON library_candidates (status, expires_at);

CREATE TABLE IF NOT EXISTS library_items (
    id                          TEXT PRIMARY KEY,
    status                      TEXT NOT NULL,
    title                       TEXT NOT NULL,
    source                      TEXT NOT NULL,
    tier                        TEXT NOT NULL,
    timestamp                   TEXT NOT NULL,
    collected_at                TEXT NOT NULL,
    author                      TEXT,
    content_type                TEXT NOT NULL,
    media_type                  TEXT NOT NULL,
    related_stocks_json         TEXT,
    tags_json                   TEXT,
    dedup_key                   TEXT NOT NULL UNIQUE,
    canonical_url               TEXT,
    cluster_id                  TEXT,
    primary_asset_id            TEXT,
    summary_by_user             TEXT,
    user_notes                  TEXT,
    promoted_from_candidate_id  TEXT,
    merged_into_id              TEXT,
    duplicate_of_id             TEXT,
    schema_version              INTEGER NOT NULL DEFAULT 1,
    reference_count             INTEGER NOT NULL DEFAULT 0,
    last_referenced_at          TEXT,
    archived_at                 TEXT,
    created_at                  TEXT NOT NULL,
    updated_at                  TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_library_items_status ON library_items (status);

CREATE TABLE IF NOT EXISTS library_item_assets (
    id                          TEXT PRIMARY KEY,
    library_item_id             TEXT NOT NULL,
    asset_role                  TEXT NOT NULL,
    source                      TEXT NOT NULL,
    tier                        TEXT NOT NULL,
    timestamp                   TEXT NOT NULL,
    file_path                   TEXT,
    file_sha256                 TEXT,
    canonical_url               TEXT,
    promoted_from_candidate_id  TEXT,
    supplement_note             TEXT,
    created_at                  TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_library_assets_item ON library_item_assets (library_item_id);

CREATE TABLE IF NOT EXISTS library_item_media (
    library_item_id       TEXT PRIMARY KEY,
    mime_type             TEXT,
    file_sha256           TEXT,
    file_size_bytes       INTEGER,
    media_profile_json    TEXT,
    derived_assets_json   TEXT,
    updated_at            TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS library_links (
    id                TEXT PRIMARY KEY,
    library_item_id   TEXT NOT NULL,
    entity_type       TEXT NOT NULL,
    entity_id         TEXT NOT NULL,
    link_role         TEXT NOT NULL,
    created_at        TEXT NOT NULL,
    created_by        TEXT NOT NULL,
    UNIQUE (library_item_id, entity_type, entity_id, link_role)
);

CREATE INDEX IF NOT EXISTS idx_library_links_entity ON library_links (entity_type, entity_id);

-- ---------------------------------------------------------------------------
-- 监控（MVP-1 简版）
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS monitor_events (
    id              TEXT PRIMARY KEY,
    code            TEXT,
    event_type      TEXT NOT NULL,
    payload_json    TEXT NOT NULL,
    acknowledged    INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_monitor_events_code ON monitor_events (code, created_at);

-- schema 版本标记（应用层读取，非 golang-migrate 自带表）
CREATE TABLE IF NOT EXISTS schema_meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

INSERT OR IGNORE INTO schema_meta (key, value) VALUES ('schema_version', '1');
