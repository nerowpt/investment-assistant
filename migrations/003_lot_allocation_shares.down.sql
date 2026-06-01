-- 回滚标记；列保留（SQLite 无 DROP COLUMN 便捷路径）
UPDATE schema_meta SET value = '2' WHERE key = 'schema_version';
DELETE FROM schema_meta WHERE key = 'migration:003_lot_allocation_shares';
