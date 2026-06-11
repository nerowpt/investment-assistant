package api

import (
	"context"
	"database/sql"

	"github.com/investment-assistant/investment-assistant/internal/core/account"
	chksvc "github.com/investment-assistant/investment-assistant/internal/core/checklist"
	"github.com/investment-assistant/investment-assistant/internal/core/query"
	"github.com/investment-assistant/investment-assistant/internal/worker"
)

// Deps 单次请求依赖（account 来自中间件，db 按请求打开）。
type Deps struct {
	ac *account.Context
}

func newDeps(ac *account.Context) *Deps {
	return &Deps{ac: ac}
}

func (d *Deps) reader(db *sql.DB) *query.Reader {
	return query.NewReader(d.ac, db)
}

func (d *Deps) checklist(db *sql.DB) *chksvc.Service {
	return chksvc.NewService(d.ac, db)
}

func (d *Deps) worker() *worker.Client {
	return worker.NewClient(d.ac)
}

func (d *Deps) checklistWithMarket(ctx context.Context, db *sql.DB) *chksvc.Service {
	svc := chksvc.NewService(d.ac, db)
	client := worker.NewClient(d.ac)
	svc.SetMarketFetcher(&chksvc.WorkerMarketFetcher{Client: client})
	return svc
}
