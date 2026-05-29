-- 002 lots 复权与分红字段（docs/06 §D8）
-- 旧版 001 库缺下列列；新版 001 已含则 inv 运行时跳过（见 sqlstore.MigrateUp）。
-- golang-migrate 路径：若报 duplicate column，说明已升级，可标记本迁移为已应用。

ALTER TABLE lots ADD COLUMN dividends_received REAL;
ALTER TABLE lots ADD COLUMN adjusted_cost_basis REAL;
ALTER TABLE lots ADD COLUMN corporate_actions_json TEXT;

INSERT OR REPLACE INTO schema_meta (key, value) VALUES ('migration:002_lots_d8_columns', '1');
UPDATE schema_meta SET value = '2' WHERE key = 'schema_version';
