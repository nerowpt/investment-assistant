package api

import (
	"database/sql"

	"github.com/investment-assistant/investment-assistant/internal/core/account"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
)

// getDB 按 account 复用 SQLite 连接池（避免并发 Open+MigrateUp 导致 database is locked）。
func (s *Server) getDB(ac *account.Context) (*sql.DB, error) {
	key := ac.DBPath
	s.dbMu.Lock()
	defer s.dbMu.Unlock()
	if s.dbs == nil {
		s.dbs = make(map[string]*sql.DB)
	}
	if db, ok := s.dbs[key]; ok {
		return db, nil
	}
	db, err := sqlstore.Open(ac.DBPath)
	if err != nil {
		return nil, err
	}
	if err := sqlstore.MigrateUp(db); err != nil {
		db.Close()
		return nil, err
	}
	s.dbs[key] = db
	return db, nil
}

// closeDBs 关闭所有缓存连接（测试/优雅退出用）。
func (s *Server) closeDBs() {
	s.dbMu.Lock()
	defer s.dbMu.Unlock()
	for _, db := range s.dbs {
		_ = db.Close()
	}
	s.dbs = nil
}
