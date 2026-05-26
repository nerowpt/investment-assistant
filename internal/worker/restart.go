package worker

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/investment-assistant/investment-assistant/internal/core/account"
)

// RestartWorker 停止旧 worker 进程并清除端口文件（改 Python 代码后须执行）。
func RestartWorker(ac *account.Context) error {
	if err := os.MkdirAll(ac.RunDir(), 0o755); err != nil {
		return err
	}
	pidPath := filepath.Join(ac.RunDir(), "worker.pid")
	if raw, err := os.ReadFile(pidPath); err == nil {
		pid, _ := strconv.Atoi(strings.TrimSpace(string(raw)))
		if pid > 0 {
			if proc, err := os.FindProcess(pid); err == nil {
				_ = proc.Kill()
				_, _ = proc.Wait()
			}
		}
	}
	killAllWorkerProcesses()
	_ = os.Remove(ac.WorkerPortPath())
	_ = os.Remove(pidPath)
	return nil
}
