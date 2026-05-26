package worker

import (
	"os/exec"
	"runtime"
)

// killAllWorkerProcesses 结束所有 python -m data_worker 进程（含 restart 未记录的孤儿进程）。
func killAllWorkerProcesses() {
	if runtime.GOOS == "windows" {
		script := `Get-CimInstance Win32_Process -Filter "Name='python.exe'" -ErrorAction SilentlyContinue | Where-Object { $_.CommandLine -like '*-m data_worker*' } | ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }`
		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
		_ = cmd.Run()
		return
	}
	cmd := exec.Command("pkill", "-f", "python.*-m data_worker")
	_ = cmd.Run()
}
