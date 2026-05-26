package worker

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

func tailFile(path string, maxBytes int) string {
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) == 0 {
		return ""
	}
	if len(raw) > maxBytes {
		raw = raw[len(raw)-maxBytes:]
	}
	return strings.TrimSpace(string(raw))
}

func copyCmdOutput(r io.Reader, logPath string) {
	raw, _ := io.ReadAll(r)
	if len(raw) == 0 {
		return
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err == nil {
		_, _ = logFile.Write(raw)
		_ = logFile.Close()
	}
	_, _ = os.Stderr.Write(raw)
}

func workerLogPath(runDir string) string {
	return filepath.Join(runDir, "worker.log")
}
