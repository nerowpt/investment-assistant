// Package worker 管理 Python data-worker 子进程与 gRPC 客户端（04 §21.6–§21.7）。
package worker

import (
	"context"
	"fmt"
	"time"

	dataworkerv1 "github.com/investment-assistant/investment-assistant/gen/go/dataworker/v1"
	"github.com/investment-assistant/investment-assistant/internal/core/account"
)

const defaultGRPCTimeout = 30 * time.Second

// Client 调用 Python DataWorker RPC。
type Client struct {
	ac         *account.Context
	supervisor *Supervisor
	rpc        dataworkerv1.DataWorkerClient
}

// NewClient 构造 worker 客户端（懒连接）。
func NewClient(ac *account.Context) *Client {
	return &Client{
		ac:         ac,
		supervisor: NewSupervisor(ac),
	}
}

// HealthCheck 探活 worker。
func (c *Client) HealthCheck(ctx context.Context) (*dataworkerv1.HealthCheckResponse, error) {
	if err := c.ensure(ctx); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return c.rpc.HealthCheck(ctx, &dataworkerv1.HealthCheckRequest{})
}

// FetchQuote 拉取实时行情。
func (c *Client) FetchQuote(ctx context.Context, code string) (*dataworkerv1.FetchQuoteResponse, error) {
	if err := c.ensure(ctx); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, defaultGRPCTimeout)
	defer cancel()
	return c.rpc.FetchQuote(ctx, &dataworkerv1.FetchQuoteRequest{
		Code:      code,
		AccountId: c.ac.AccountID,
	})
}

// FetchValuation 拉取估值指标。
func (c *Client) FetchValuation(ctx context.Context, code string) (*dataworkerv1.FetchValuationResponse, error) {
	if err := c.ensure(ctx); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, defaultGRPCTimeout)
	defer cancel()
	return c.rpc.FetchValuation(ctx, &dataworkerv1.FetchValuationRequest{Code: code})
}

// FetchAnnouncements 拉取公告列表。
func (c *Client) FetchAnnouncements(ctx context.Context, req *dataworkerv1.FetchAnnouncementsRequest) (*dataworkerv1.FetchAnnouncementsResponse, error) {
	if err := c.ensure(ctx); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, defaultGRPCTimeout)
	defer cancel()
	return c.rpc.FetchAnnouncements(ctx, req)
}

// FetchMarketSnapshot 拉取指数摘要。
func (c *Client) FetchMarketSnapshot(ctx context.Context, indexCodes []string) (*dataworkerv1.FetchMarketSnapshotResponse, error) {
	if err := c.ensure(ctx); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, defaultGRPCTimeout)
	defer cancel()
	return c.rpc.FetchMarketSnapshot(ctx, &dataworkerv1.FetchMarketSnapshotRequest{IndexCodes: indexCodes})
}

// ExtractDocument 抽取 URL/文件/文本。
func (c *Client) ExtractDocument(ctx context.Context, req *dataworkerv1.ExtractDocumentRequest) (*dataworkerv1.ExtractDocumentResponse, error) {
	if err := c.ensure(ctx); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, defaultGRPCTimeout)
	defer cancel()
	return c.rpc.ExtractDocument(ctx, req)
}

func (c *Client) ensure(ctx context.Context) error {
	rpc, err := c.supervisor.EnsureWorker(ctx)
	if err != nil {
		return fmt.Errorf("data-worker unavailable: %w", err)
	}
	c.rpc = rpc
	return nil
}

// Close 关闭 supervisor 持有的连接。
func (c *Client) Close() error {
	return c.supervisor.Close()
}
