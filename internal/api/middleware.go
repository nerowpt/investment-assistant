package api

import (
	"context"
	"net/http"

	"github.com/investment-assistant/investment-assistant/internal/core/account"
)

type ctxKey int

const accountCtxKey ctxKey = 1

// AccountFromContext 从请求上下文取 account。
func AccountFromContext(ctx context.Context) *account.Context {
	ac, _ := ctx.Value(accountCtxKey).(*account.Context)
	return ac
}

// WithAccountMiddleware 解析 X-Account-Id（缺省 default），注入 account.Context。
func WithAccountMiddleware(base *account.Context) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			accountID := r.Header.Get("X-Account-Id")
			if accountID == "" {
				accountID = base.AccountID
			}
			ac, err := account.WithAccount(base.DataRoot, accountID)
			if err != nil {
				WriteError(w, http.StatusBadRequest, "invalid_account", err.Error())
				return
			}
			if err := ac.EnsureInitialized(); err != nil {
				WriteError(w, http.StatusInternalServerError, "init_failed", err.Error())
				return
			}
			ctx := context.WithValue(r.Context(), accountCtxKey, ac)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AuthMiddleware 鉴权占位（MVP-2 托管时接入 JWT）；当前默认放行。
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO(MVP-2): 校验 Authorization Bearer token
		next.ServeHTTP(w, r)
	})
}
