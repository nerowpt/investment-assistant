package worker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// pythonCommand 返回启动 data_worker 的 executable 与额外前缀参数（如 py -3）。
func pythonCommand() (exe string, prefixArgs []string, err error) {
	if v := strings.TrimSpace(os.Getenv("IA_PYTHON")); v != "" {
		return v, nil, nil
	}
	if runtime.GOOS == "windows" {
		if path, lookErr := exec.LookPath("py"); lookErr == nil {
			return path, []string{"-3"}, nil
		}
		if path, lookErr := exec.LookPath("python"); lookErr == nil {
			if !isWindowsStorePythonStub(path) {
				return path, nil, nil
			}
		}
		for _, candidate := range windowsPythonCandidates() {
			if _, statErr := os.Stat(candidate); statErr == nil {
				return candidate, nil, nil
			}
		}
		return "", nil, fmt.Errorf("未找到可用的 Python（请安装 Python 3.10+ 或设置 IA_PYTHON）")
	}
	for _, name := range []string{"python3", "python"} {
		if path, lookErr := exec.LookPath(name); lookErr == nil {
			return path, nil, nil
		}
	}
	return "", nil, fmt.Errorf("未找到 python3/python（可设置 IA_PYTHON）")
}

func isWindowsStorePythonStub(path string) bool {
	return strings.Contains(strings.ToLower(filepath.Clean(path)), `\windowsapps\`)
}

func windowsPythonCandidates() []string {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		return nil
	}
	root := filepath.Join(local, "Programs", "Python")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		out = append(out, filepath.Join(root, e.Name(), "python.exe"))
	}
	return out
}

// VerifyPythonEnv 确认 Python 能 import data_worker（pip install -e .）。
func VerifyPythonEnv(workerDir string) error {
	exe, prefix, err := pythonCommand()
	if err != nil {
		return err
	}
	args := append(append([]string{}, prefix...), "-c", "import data_worker; print(data_worker.__version__)")
	cmd := exec.Command(exe, args...)
	cmd.Dir = workerDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s: %s", exe, msg)
	}
	return nil
}

func pythonExecutable() string {
	exe, _, err := pythonCommand()
	if err != nil {
		if runtime.GOOS == "windows" {
			return "python"
		}
		return "python3"
	}
	return exe
}
