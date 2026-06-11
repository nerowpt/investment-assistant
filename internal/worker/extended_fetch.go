package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/investment-assistant/investment-assistant/internal/core/account"
	dataworkerv1 "github.com/investment-assistant/investment-assistant/gen/go/dataworker/v1"
)

// ExtendedPackResult 研究扩展包拉取结果。
type ExtendedPackResult struct {
	Code       string
	StockName  string
	Title      string
	Summary    string
	Body       string
	Source     string
	Tier       string
	CapturedAt string
}

// FetchResearchExtended 拉取 sector_valuation / volume 等扩展包。
// 扩展包固定走 subprocess，避免长期运行的 gRPC worker 未热更新 Python 逻辑。
func (c *Client) FetchResearchExtended(ctx context.Context, code, pack string) (*ExtendedPackResult, error) {
	code = strings.TrimSpace(code)
	pack = strings.TrimSpace(pack)
	if code == "" || pack == "" {
		return nil, fmt.Errorf("code 与 pack 必填")
	}
	return fetchExtendedSubprocess(ctx, c.ac, pack, code)
}

func extendedFromProto(res *dataworkerv1.FetchResearchExtendedResponse) *ExtendedPackResult {
	src, tier, captured := "akshare", "B", time.Now().Format(time.RFC3339)
	if p := res.GetProvenance(); p != nil {
		src, tier, captured = p.GetSource(), p.GetTier(), p.GetCapturedAt()
	}
	return &ExtendedPackResult{
		Code: res.GetCode(), StockName: res.GetStockName(),
		Title: res.GetTitle(), Summary: res.GetSummary(), Body: res.GetBody(),
		Source: src, Tier: tier, CapturedAt: captured,
	}
}

func fetchExtendedSubprocess(ctx context.Context, ac *account.Context, pack, code string) (*ExtendedPackResult, error) {
	workerDir := account.WorkerRepoPath()
	if err := VerifyPythonEnv(workerDir); err != nil {
		return nil, err
	}
	exe, prefix, err := pythonCommand()
	if err != nil {
		return nil, err
	}
	args := append(append([]string{}, prefix...), "-m", "data_worker.fetch_pack", pack, code)
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Dir = workerDir
	cmd.Env = append(os.Environ(), "TQDM_DISABLE=1", "PYTHONWARNINGS=ignore", "PYTHONIOENCODING=utf-8")
	out, err := cmd.Output()
	if err != nil {
		msg := subprocessErrorMessage(out, err)
		if raw, parseErr := parseWorkerJSON(out); parseErr == nil {
			if e, ok := raw["error"].(string); ok && e != "" {
				return nil, fmt.Errorf("%s", e)
			}
		}
		return nil, fmt.Errorf("扩展包拉取失败: %s", msg)
	}
	raw, err := parseWorkerJSON(out)
	if err != nil {
		return nil, fmt.Errorf("解析 worker 输出: %w", err)
	}
	if e, ok := raw["error"].(string); ok && e != "" {
		return nil, fmt.Errorf("%s", e)
	}
	prov, _ := raw["provenance"].(map[string]any)
	src, tier, captured := "akshare", "B", time.Now().Format(time.RFC3339)
	if prov != nil {
		if s, ok := prov["source"].(string); ok {
			src = s
		}
		if t, ok := prov["tier"].(string); ok {
			tier = t
		}
		if c, ok := prov["captured_at"].(string); ok {
			captured = c
		}
	}
	return &ExtendedPackResult{
		Code:       strField(raw, "code", code),
		StockName:  strField(raw, "stock_name", ""),
		Title:      strField(raw, "title", ""),
		Summary:    strField(raw, "summary", ""),
		Body:       strField(raw, "body", ""),
		Source:     src,
		Tier:       tier,
		CapturedAt: captured,
	}, nil
}

func strField(m map[string]any, key, def string) string {
	if v, ok := m[key].(string); ok && v != "" {
		return v
	}
	return def
}

// parseWorkerJSON 从 subprocess stdout 解析 JSON（容忍前后夹杂的进度条等非 JSON 文本）。
func parseWorkerJSON(out []byte) (map[string]any, error) {
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil, fmt.Errorf("空输出")
	}
	dec := json.NewDecoder(bytes.NewReader([]byte(trimmed)))
	var raw map[string]any
	if err := dec.Decode(&raw); err == nil {
		return raw, nil
	}
	// fallback：取最后一行形如 {...} 的输出
	lines := strings.Split(trimmed, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "{") || !strings.HasSuffix(line, "}") {
			continue
		}
		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err == nil {
			return row, nil
		}
	}
	return nil, fmt.Errorf("invalid character after top-level value")
}

func subprocessErrorMessage(stdout []byte, err error) string {
	msg := strings.TrimSpace(string(stdout))
	if msg != "" {
		return msg
	}
	if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
		return strings.TrimSpace(string(ee.Stderr))
	}
	return err.Error()
}
