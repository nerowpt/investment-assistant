// Package yamlstore：Store 接口集（docs/04 §20.3.1、docs/06 §D13）。
//
// 设计目标：
//   - app 层（H4 起）通过接口注入 Store，便于单测 mock；
//   - File 实现走 atomicWrite + 真实文件系统；
//   - Memory 实现仅供测试用，零文件 I/O。
//
// 注意：
//   - 为避免 account ↔ yamlstore 循环依赖，Store 接口直接接收文件路径 string
//     而非 *account.Context；上层 wiring（cli/app）负责从 AccountContext 拼装路径。
//   - 包级 LoadPortfolio / SavePortfolio 函数保留为向后兼容 alias，
//     当前 cli/doctor 仍直接调用；H2 起所有新增 cli 命令必须走接口。
package yamlstore

import (
	"context"
	"errors"
	"os"
	"sync"
)

// ErrNotFound store 未持久化任何内容时返回（Memory 实现 + File 实现读不到文件时统一语义）。
var ErrNotFound = errors.New("yamlstore: not found")

// PortfolioStore portfolio.yaml 仓储（03 §10B.2 SoT）。
// 实现须保证 Save 是原子写（tmp + rename），失败不破坏既有文件。
type PortfolioStore interface {
	Load(ctx context.Context, path string) (*Portfolio, error)
	Save(ctx context.Context, path string, p *Portfolio) error
}

// NewFilePortfolioStore 文件系统实现，按传入路径读写。
func NewFilePortfolioStore() PortfolioStore {
	return &filePortfolioStore{}
}

type filePortfolioStore struct{}

func (s *filePortfolioStore) Load(ctx context.Context, path string) (*Portfolio, error) {
	p, err := LoadPortfolio(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func (s *filePortfolioStore) Save(ctx context.Context, path string, p *Portfolio) error {
	return SavePortfolio(path, p)
}

// NewMemoryPortfolioStore 内存实现（仅测试用）。
// 以 path 为 key 隔离，与 File 实现的物理隔离语义一致。
func NewMemoryPortfolioStore() PortfolioStore {
	return &memoryPortfolioStore{data: map[string]*Portfolio{}}
}

type memoryPortfolioStore struct {
	mu   sync.RWMutex
	data map[string]*Portfolio
}

func (s *memoryPortfolioStore) Load(ctx context.Context, path string) (*Portfolio, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.data[path]
	if !ok {
		return nil, ErrNotFound
	}
	return clonePortfolio(p), nil
}

func (s *memoryPortfolioStore) Save(ctx context.Context, path string, p *Portfolio) error {
	if p == nil {
		return errors.New("yamlstore: nil portfolio")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[path] = clonePortfolio(p)
	return nil
}
