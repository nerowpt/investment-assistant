-- 003 lot_allocations 股数级盈亏字段（B 模型：手动成交价 + 股数）
ALTER TABLE lot_allocations ADD COLUMN allocated_shares REAL;
ALTER TABLE lot_allocations ADD COLUMN proceeds_amount REAL;
ALTER TABLE lot_allocations ADD COLUMN realized_pnl_amount REAL;

INSERT OR REPLACE INTO schema_meta (key, value) VALUES ('migration:003_lot_allocation_shares', '1');
UPDATE schema_meta SET value = '3' WHERE key = 'schema_version';
