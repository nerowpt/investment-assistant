package doctor

import (
	"fmt"
	"strings"
)

// Issue doctor 检查项；CLI 按「编号 + 含义 + 处理」输出。
type Issue struct {
	Code    string // 规则编号，如 P002
	Subject string // 标的 code 或 YAML 路径
	Title   string // 一句话标题
	Detail  string // 检测到的具体事实
	Hint    string // 用户可执行的处理建议
}

// FormatIssues 格式化 portfolio/watchlist 等问题列表（供 CLI）。
func FormatIssues(issues []Issue) string {
	var b strings.Builder
	for i, iss := range issues {
		if i > 0 {
			b.WriteByte('\n')
		}
		subj := iss.Subject
		if subj == "" {
			subj = "—"
		}
		fmt.Fprintf(&b, "  [%d] %s · %s · %s\n", i+1, iss.Code, subj, iss.Title)
		if iss.Detail != "" {
			fmt.Fprintf(&b, "      发现: %s\n", iss.Detail)
		}
		if iss.Hint != "" {
			fmt.Fprintf(&b, "      处理: %s\n", iss.Hint)
		}
	}
	return b.String()
}
