-- SQLite 不支持 DROP COLUMN（旧版）；回滚仅标记版本，列保留。
UPDATE schema_meta SET value = '1' WHERE key = 'schema_version';
DELETE FROM schema_meta WHERE key = 'migration:002_lots_d8_columns';
