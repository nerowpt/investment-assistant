// Package backup 实现 account 数据备份与恢复（03 §10D）。
package backup

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/investment-assistant/investment-assistant/internal/core/account"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
)

// Mode 备份范围。
type Mode string

const (
	ModeLite Mode = "lite" // state + db
	ModeFull Mode = "full" // lite + library + reports
)

// Manifest 备份包元数据（manifest.json）。
type Manifest struct {
	BackupID  string `json:"backup_id"`
	AccountID string `json:"account_id"`
	Mode      Mode   `json:"mode"`
	CreatedAt string `json:"created_at"`
	DataRoot  string `json:"data_root"`
	Tool      string `json:"tool"`
}

// Create 创建备份目录并写入 manifest。
func Create(ac *account.Context, mode Mode, destination string) (*Manifest, error) {
	if mode != ModeLite && mode != ModeFull {
		return nil, fmt.Errorf("mode 须为 lite 或 full")
	}
	src := ac.AccountRoot()
	if _, err := os.Stat(src); err != nil {
		return nil, fmt.Errorf("account 目录不存在: %s", src)
	}
	if err := checkpointSQLite(ac.DBPath); err != nil {
		return nil, err
	}

	ts := time.Now().Format("20060102_150405")
	dest := destination
	if dest == "" {
		dest = filepath.Join(ac.BackupRoot, ac.AccountID, ts)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return nil, err
	}

	for _, rel := range liteDirs(mode) {
		if err := copyTree(filepath.Join(src, rel), filepath.Join(dest, rel)); err != nil {
			return nil, fmt.Errorf("复制 %s: %w", rel, err)
		}
	}

	m := &Manifest{
		BackupID:  ts,
		AccountID: ac.AccountID,
		Mode:      mode,
		CreatedAt: time.Now().Format(time.RFC3339),
		DataRoot:  ac.DataRoot,
		Tool:      "inv backup create",
	}
	raw, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(dest, "manifest.json"), raw, 0o644); err != nil {
		return nil, err
	}
	return m, nil
}

func liteDirs(mode Mode) []string {
	dirs := []string{"state", "db"}
	if mode == ModeFull {
		dirs = append(dirs, "library", "reports")
	}
	return dirs
}

// List 列出 account 下所有备份 manifest（新→旧）。
func List(ac *account.Context) ([]Manifest, error) {
	root := filepath.Join(ac.BackupRoot, ac.AccountID)
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Manifest
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		m, err := readManifest(filepath.Join(root, e.Name()))
		if err != nil {
			continue
		}
		out = append(out, *m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].BackupID > out[j].BackupID })
	return out, nil
}

// Show 读取单个备份 manifest。
func Show(ac *account.Context, backupID string) (*Manifest, error) {
	path, err := backupDir(ac, backupID)
	if err != nil {
		return nil, err
	}
	return readManifest(path)
}

// Restore 从备份恢复 account 数据；restore 前自动 lite 快照 pre_restore_*。
func Restore(ac *account.Context, backupID string, dryRun, yes bool) error {
	src, err := backupDir(ac, backupID)
	if err != nil {
		return err
	}
	m, err := readManifest(src)
	if err != nil {
		return err
	}
	dst := ac.AccountRoot()
	dirs := liteDirs(m.Mode)

	if dryRun {
		fmt.Printf("dry-run: 将从 %s 覆盖 %s\n", src, dst)
		for _, rel := range dirs {
			fmt.Printf("  - %s\n", rel)
		}
		return nil
	}
	if !yes {
		return fmt.Errorf("restore 将覆盖当前 account 数据；须加 --yes 确认（建议先 inv backup create）")
	}

	preID := "pre_restore_" + time.Now().Format("20060102_150405")
	preDest := filepath.Join(ac.BackupRoot, ac.AccountID, preID)
	if _, preErr := Create(ac, ModeLite, preDest); preErr != nil {
		return fmt.Errorf("restore 前自动快照失败: %w", preErr)
	}

	for _, rel := range dirs {
		from := filepath.Join(src, rel)
		to := filepath.Join(dst, rel)
		if err := replaceTree(from, to); err != nil {
			return fmt.Errorf("恢复 %s: %w", rel, err)
		}
	}
	fmt.Printf("restore OK: from=%s pre_restore=%s\n", backupID, preID)
	fmt.Println("请运行: inv doctor --scope all")
	return nil
}

// Prune 保留最新 keep 份备份，删除更旧的目录。
func Prune(ac *account.Context, keep int) (int, error) {
	if keep < 1 {
		return 0, fmt.Errorf("keep 须 >= 1")
	}
	all, err := List(ac)
	if err != nil {
		return 0, err
	}
	if len(all) <= keep {
		return 0, nil
	}
	removed := 0
	for _, m := range all[keep:] {
		dir, err := backupDir(ac, m.BackupID)
		if err != nil {
			continue
		}
		if err := os.RemoveAll(dir); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

func backupDir(ac *account.Context, backupID string) (string, error) {
	if backupID == "" {
		return "", fmt.Errorf("backup_id 不能为空")
	}
	dir := filepath.Join(ac.BackupRoot, ac.AccountID, backupID)
	if _, err := os.Stat(filepath.Join(dir, "manifest.json")); err != nil {
		return "", fmt.Errorf("备份不存在: %s", backupID)
	}
	return dir, nil
}

func readManifest(dir string) (*Manifest, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if m.BackupID == "" {
		m.BackupID = filepath.Base(dir)
	}
	return &m, nil
}

func checkpointSQLite(dbPath string) error {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil
	}
	db, err := sqlstore.Open(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`)
	return err
}

func copyTree(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return copyFile(src, dst)
	}
	return filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if fi.IsDir() {
			return os.MkdirAll(target, fi.Mode())
		}
		return copyFile(path, target)
	})
}

func replaceTree(src, dst string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	return copyTree(src, dst)
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
