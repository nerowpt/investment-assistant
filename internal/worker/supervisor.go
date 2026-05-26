package worker

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	dataworkerv1 "github.com/investment-assistant/investment-assistant/gen/go/dataworker/v1"
	"github.com/investment-assistant/investment-assistant/internal/core/account"
	"github.com/investment-assistant/investment-assistant/internal/coreingest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	maxStartAttempts   = 3
	healthPollInterval = 300 * time.Millisecond
	healthWaitTimeout  = 30 * time.Second
)

// Supervisor 懒启动并监控 Python data-worker。
type Supervisor struct {
	ac      *account.Context
	mu      sync.Mutex
	conn    *grpc.ClientConn
	client  dataworkerv1.DataWorkerClient
	cmd     *exec.Cmd
	ingest  *coreingest.Server
	addr    string
	starts  int
}

// NewSupervisor 构造 supervisor。
func NewSupervisor(ac *account.Context) *Supervisor {
	return &Supervisor{ac: ac}
}

// EnsureWorker 确保 worker 可 HealthCheck；必要时拉起子进程。
func (s *Supervisor) EnsureWorker(ctx context.Context) (dataworkerv1.DataWorkerClient, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		if _, err := s.client.HealthCheck(ctx, &dataworkerv1.HealthCheckRequest{}); err == nil {
			return s.client, nil
		}
		s.resetConn()
	}

	if addr, err := readWorkerAddr(s.ac.WorkerPortPath()); err == nil {
		if client, err := s.dial(ctx, addr); err == nil {
			return client, nil
		}
		_ = os.Remove(s.ac.WorkerPortPath())
	}

	var lastErr error
	for attempt := 0; attempt < maxStartAttempts; attempt++ {
		if attempt > 0 {
			s.cleanupPartialStart()
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
		}
		client, err := s.startWorker(ctx)
		if err == nil {
			return client, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("supervisor 启动 worker 失败（%d 次）: %w", maxStartAttempts, lastErr)
}

func (s *Supervisor) startWorker(ctx context.Context) (dataworkerv1.DataWorkerClient, error) {
	if err := os.MkdirAll(s.ac.RunDir(), 0o755); err != nil {
		return nil, err
	}

	if err := s.ensureIngestServer(); err != nil {
		return nil, err
	}

	_ = os.Remove(s.ac.WorkerPortPath())

	python, pyArgs, err := pythonCommand()
	if err != nil {
		return nil, err
	}
	workerDir := account.WorkerRepoPath()
	if _, err := os.Stat(workerDir); err != nil {
		return nil, fmt.Errorf("worker 目录不存在: %s", workerDir)
	}
	if err := VerifyPythonEnv(workerDir); err != nil {
		return nil, fmt.Errorf("Python 环境未就绪: %w（请在 services/data-worker 执行: pip install -e .）", err)
	}
	repoRoot := findRepoRoot(workerDir)
	portFile := s.ac.WorkerPortPath()
	args := append(append([]string{}, pyArgs...), "-m", "data_worker")
	cmd := exec.CommandContext(ctx, python, args...)
	cmd.Dir = workerDir
	cmd.Env = append(os.Environ(),
		"IA_WORKER_LISTEN=127.0.0.1:0",
		"IA_WORKER_PORT_FILE="+portFile,
		"IA_ACCOUNT_ID="+s.ac.AccountID,
		"IA_CORE_INGEST_TARGET="+coreIngestAddr(),
		"IA_REPO_ROOT="+repoRoot,
		"PYTHONUNBUFFERED=1",
	)
	logPath := workerLogPath(s.ac.RunDir())
	stdoutR, stdoutW := io.Pipe()
	cmd.Stdout = stdoutW
	cmd.Stderr = stdoutW
	if err := cmd.Start(); err != nil {
		_ = stdoutW.Close()
		return nil, fmt.Errorf("启动 python worker (%s): %w", python, err)
	}
	_ = stdoutW.Close()
	waitDone := make(chan error, 1)
	go func() {
		copyCmdOutput(stdoutR, logPath)
		waitDone <- cmd.Wait()
	}()
	s.cmd = cmd
	s.starts++

	pidFile := filepath.Join(s.ac.RunDir(), "worker.pid")
	_ = os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0o644)

	deadline := time.Now().Add(healthWaitTimeout)
	var lastErr error
	for time.Now().Before(deadline) {
		select {
		case waitErr := <-waitDone:
			if waitErr != nil {
				return nil, fmt.Errorf("python worker 已退出: %w\n%s", waitErr, tailFile(logPath, 4096))
			}
			return nil, fmt.Errorf("python worker 已退出（详见 %s）", logPath)
		default:
		}
		addr, err := readWorkerAddr(portFile)
		if err == nil {
			var client dataworkerv1.DataWorkerClient
			client, err = s.dial(ctx, addr)
			if err == nil {
				s.addr = addr
				return client, nil
			}
			lastErr = err
		}
		select {
		case <-ctx.Done():
			s.cleanupPartialStart()
			return nil, ctx.Err()
		case waitErr := <-waitDone:
			if waitErr != nil {
				return nil, fmt.Errorf("python worker 已退出: %w\n%s", waitErr, tailFile(logPath, 4096))
			}
			return nil, fmt.Errorf("python worker 已退出（详见 %s）", logPath)
		case <-time.After(healthPollInterval):
		}
	}
	s.cleanupPartialStart()
	if lastErr != nil {
		return nil, fmt.Errorf("worker 在 %s 内未就绪: %w（详见 %s）", healthWaitTimeout, lastErr, logPath)
	}
	if tail := tailFile(logPath, 4096); tail != "" {
		return nil, fmt.Errorf("worker 在 %s 内未就绪（未写入 worker.port）\n%s", healthWaitTimeout, tail)
	}
	return nil, fmt.Errorf("worker 在 %s 内未就绪（未写入 worker.port，详见 %s）", healthWaitTimeout, logPath)
}

func (s *Supervisor) dial(ctx context.Context, addr string) (dataworkerv1.DataWorkerClient, error) {
	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(dialCtx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}
	client := dataworkerv1.NewDataWorkerClient(conn)
	if _, err := client.HealthCheck(dialCtx, &dataworkerv1.HealthCheckRequest{}); err != nil {
		conn.Close()
		return nil, err
	}
	s.conn = conn
	s.client = client
	return client, nil
}

func (s *Supervisor) ensureIngestServer() error {
	if s.ingest != nil && s.ingest.Running() {
		return nil
	}
	addr := coreIngestAddr()
	if tcpReachable(addr) {
		// 上一 inv 进程已启动 CoreIngest，本进程复用即可
		return nil
	}
	srv := coreingest.NewServer(s.ac.DataRoot)
	if err := srv.Start(addr); err != nil {
		return err
	}
	s.ingest = srv
	return nil
}

func (s *Supervisor) resetConn() {
	if s.conn != nil {
		_ = s.conn.Close()
	}
	s.conn = nil
	s.client = nil
}

func (s *Supervisor) cleanupPartialStart() {
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_, _ = s.cmd.Process.Wait()
	}
	s.cmd = nil
	_ = os.Remove(s.ac.WorkerPortPath())
	_ = os.Remove(filepath.Join(s.ac.RunDir(), "worker.pid"))
}

// Close 关闭 gRPC 连接（不杀 worker 子进程，便于复用）。
func (s *Supervisor) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resetConn()
	return nil
}

func readWorkerAddr(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	addr := strings.TrimSpace(string(raw))
	if addr == "" {
		return "", fmt.Errorf("worker.port 为空")
	}
	return addr, nil
}

func pickFreePort() (int, error) {
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var ctrlErr error
			if err := c.Control(func(fd uintptr) {
				ctrlErr = setReuseAddr(fd)
			}); err != nil {
				return err
			}
			return ctrlErr
		},
	}
	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

func coreIngestAddr() string {
	if v := os.Getenv("IA_CORE_INGEST_ADDR"); v != "" {
		return v
	}
	return "127.0.0.1:50051"
}

// tcpReachable 探测 addr 是否已有服务监听（跨 inv 进程复用 CoreIngest）。
func tcpReachable(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// findRepoRoot 从 worker 目录向上查找含 go.mod 的仓库根。
func findRepoRoot(start string) string {
	dir := start
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	wd, err := os.Getwd()
	if err != nil {
		return start
	}
	return wd
}

// WorkerPortFile 返回当前 worker 地址（调试用）。
func (s *Supervisor) WorkerPortFile() string {
	return s.ac.WorkerPortPath()
}

// RepoWorkerPath 返回 worker 目录是否存在。
func RepoWorkerPath() string {
	return account.WorkerRepoPath()
}

// WorkerMainExists 检查 data_worker 模块是否可导入。
func WorkerMainExists() bool {
	p := filepath.Join(account.WorkerRepoPath(), "data_worker", "__main__.py")
	_, err := os.Stat(p)
	return err == nil
}
