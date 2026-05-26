// Package yamlstore 提供 Layer A YAML 原子读写（对齐 03 §十B、04 §10.2）。
package yamlstore

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// atomicWrite 写入 tmp 文件后 fsync 再 rename 到目标路径。
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建目录: %w", err)
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("创建临时文件: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("写入临时文件: %w", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("fsync: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// readYAML 读取并解析 YAML 文件。
func readYAML(path string, out any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("解析 YAML: %w", err)
	}
	return nil
}

// writeYAML 序列化并原子写入。
func writeYAML(path string, v any) error {
	raw, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("序列化 YAML: %w", err)
	}
	return atomicWrite(path, raw)
}

// CopyExampleIfMissing 若 dst 不存在则从 src 复制（首次初始化）。
func CopyExampleIfMissing(src, dst string) error {
	if _, err := os.Stat(dst); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	raw, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("读取模板 %s: %w", src, err)
	}
	return atomicWrite(dst, raw)
}

// ConfigRoot 返回仓库 config/ 目录（默认相对 cwd 的 ./config）。
func ConfigRoot() string {
	if root := os.Getenv("IA_CONFIG_ROOT"); root != "" {
		return root
	}
	return "config"
}

// InitStateFiles 从 config/*.example.yaml 复制缺失的 state 文件。
func InitStateFiles(stateDir string) error {
	pairs := []struct{ example, name string }{
		{"portfolio.yaml.example", "portfolio.yaml"},
		{"watchlist.yaml.example", "watchlist.yaml"},
		{"candidates.yaml.example", "candidates.yaml"},
		{"risk_rules.yaml.example", "risk_rules.yaml"},
		{"personal_redlines.yaml.example", "personal_redlines.yaml"},
		{"controlled_tags.yaml.example", "controlled_tags.yaml"},
	}
	root := ConfigRoot()
	for _, p := range pairs {
		src := filepath.Join(root, p.example)
		dst := filepath.Join(stateDir, p.name)
		if err := CopyExampleIfMissing(src, dst); err != nil {
			return fmt.Errorf("初始化 %s: %w", p.name, err)
		}
	}
	return nil
}
