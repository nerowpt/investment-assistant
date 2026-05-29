// Package account 解析 DATA_ROOT 与 account 目录布局（对齐 03 §4.1、04 §九）。
package account

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
)

// Context 单 account 运行时路径上下文。
type Context struct {
	AccountID  string
	DataRoot   string
	StateDir   string // {DATA_ROOT}/accounts/{id}/state
	DBPath     string // .../db/assistant.sqlite
	LibraryDir string
	ReportsDir string
	BackupRoot string // {DATA_ROOT}/_backups
}

// ResolveFromEnv 从环境变量解析 AccountContext。
// DATA_ROOT 默认 ./data；IA_ACCOUNT_ID 默认 default。
func ResolveFromEnv() (*Context, error) {
	dataRoot := os.Getenv("DATA_ROOT")
	if dataRoot == "" {
		dataRoot = "./data"
	}
	accountID := os.Getenv("IA_ACCOUNT_ID")
	if accountID == "" {
		accountID = "default"
	}
	return WithAccount(dataRoot, accountID)
}

// WithAccount 构造指定 account 的路径上下文。
func WithAccount(dataRoot, accountID string) (*Context, error) {
	if accountID == "" {
		return nil, fmt.Errorf("account_id 不能为空")
	}
	absRoot, err := filepath.Abs(dataRoot)
	if err != nil {
		return nil, fmt.Errorf("解析 DATA_ROOT: %w", err)
	}
	base := filepath.Join(absRoot, "accounts", accountID)
	return &Context{
		AccountID:  accountID,
		DataRoot:   absRoot,
		StateDir:   filepath.Join(base, "state"),
		DBPath:     filepath.Join(base, "db", "assistant.sqlite"),
		LibraryDir: filepath.Join(base, "library"),
		ReportsDir: filepath.Join(base, "reports"),
		BackupRoot: filepath.Join(absRoot, "_backups"),
	}, nil
}

// EnsureDirs 创建 account 所需目录（不含 state 初始化）。
func (c *Context) EnsureDirs() error {
	dirs := []string{
		filepath.Dir(c.DBPath),
		c.LibraryDir,
		c.ReportsDir,
		c.StateDir,
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("创建目录 %s: %w", d, err)
		}
	}
	return nil
}

// PortfolioPath 返回 portfolio.yaml 路径。
func (c *Context) PortfolioPath() string {
	return filepath.Join(c.StateDir, "portfolio.yaml")
}

// WatchlistPath 返回 watchlist.yaml 路径。
func (c *Context) WatchlistPath() string {
	return filepath.Join(c.StateDir, "watchlist.yaml")
}

// ControlledTagsPath 返回 controlled_tags.yaml 路径。
func (c *Context) ControlledTagsPath() string {
	return filepath.Join(c.StateDir, "controlled_tags.yaml")
}

// RiskRulesPath 返回 risk_rules.yaml 路径。
func (c *Context) RiskRulesPath() string {
	return filepath.Join(c.StateDir, "risk_rules.yaml")
}

// PersonalRedlinesPath 返回 personal_redlines.yaml 路径。
func (c *Context) PersonalRedlinesPath() string {
	return filepath.Join(c.StateDir, "personal_redlines.yaml")
}

// RunDir 返回运行时目录（worker.port、supervisor 状态）。
func (c *Context) RunDir() string {
	return filepath.Join(c.DataRoot, ".run")
}

// WorkerPortPath 返回 worker 端口文件路径（Windows）。
func (c *Context) WorkerPortPath() string {
	return filepath.Join(c.RunDir(), "worker.port")
}

// WorkerRepoPath 返回 Python data-worker 包根目录。
func WorkerRepoPath() string {
	if root := os.Getenv("IA_REPO_ROOT"); root != "" {
		return filepath.Join(root, "services", "data-worker")
	}
	// 从当前工作目录向上查找 go.mod
	wd, err := os.Getwd()
	if err == nil {
		dir := wd
		for i := 0; i < 6; i++ {
			candidate := filepath.Join(dir, "services", "data-worker")
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return candidate
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return filepath.Join("services", "data-worker")
}

// EnsureInitialized 创建目录并从 config 模板复制缺失的 state 文件。
func (c *Context) EnsureInitialized() error {
	if err := c.EnsureDirs(); err != nil {
		return err
	}
	return yamlstore.InitStateFiles(c.StateDir)
}
