package account

import "path/filepath"

// AccountRoot 返回 account 数据根目录（state/db/library/reports）。
func (c *Context) AccountRoot() string {
	return filepath.Join(c.DataRoot, "accounts", c.AccountID)
}
