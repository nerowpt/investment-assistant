package cli

import (
	"encoding/json"
	"fmt"

	"github.com/investment-assistant/investment-assistant/internal/core/account"
	"github.com/investment-assistant/investment-assistant/internal/core/backup"
	"github.com/spf13/cobra"
)

func newBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "account 数据备份与恢复（H7）",
	}
	cmd.AddCommand(newBackupCreateCmd(), newBackupListCmd(), newBackupShowCmd(), newBackupRestoreCmd(), newBackupPruneCmd())
	return cmd
}

func newBackupCreateCmd() *cobra.Command {
	var mode, dest string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "创建备份包",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := resolveAccount(cmd)
			if err != nil {
				return err
			}
			if err := ac.EnsureInitialized(); err != nil {
				return err
			}
			m, err := backup.Create(ac, backup.Mode(mode), dest)
			if err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(m)
			}
			fmt.Printf("backup OK: id=%s mode=%s path=%s\n", m.BackupID, m.Mode, backupPath(ac, m.BackupID))
			return nil
		},
	}
	cmd.Flags().StringVar(&mode, "mode", "lite", "lite|full")
	cmd.Flags().StringVar(&dest, "destination", "", "自定义输出目录（默认 BACKUP_ROOT/account/timestamp）")
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON 输出")
	return cmd
}

func newBackupListCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出备份",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := resolveAccount(cmd)
			if err != nil {
				return err
			}
			list, err := backup.List(ac)
			if err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(list)
			}
			for _, m := range list {
				fmt.Printf("%s\t%s\t%s\t%s\n", m.BackupID, m.Mode, m.CreatedAt, backupPath(ac, m.BackupID))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON 输出")
	return cmd
}

func newBackupShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [backup_id]",
		Short: "查看备份 manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := resolveAccount(cmd)
			if err != nil {
				return err
			}
			m, err := backup.Show(ac, args[0])
			if err != nil {
				return err
			}
			raw, _ := json.MarshalIndent(m, "", "  ")
			fmt.Println(string(raw))
			return nil
		},
	}
	return cmd
}

func newBackupRestoreCmd() *cobra.Command {
	var dryRun, yes bool
	cmd := &cobra.Command{
		Use:   "restore",
		Short: "从备份恢复（覆盖当前 account）",
		RunE: func(cmd *cobra.Command, args []string) error {
			from, err := cmd.Flags().GetString("from")
			if err != nil || from == "" {
				return fmt.Errorf("须指定 --from <backup_id>")
			}
			ac, err := resolveAccount(cmd)
			if err != nil {
				return err
			}
			return backup.Restore(ac, from, dryRun, yes)
		},
	}
	cmd.Flags().String("from", "", "备份 id")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "仅预览")
	cmd.Flags().BoolVar(&yes, "yes", false, "确认覆盖")
	return cmd
}

func newBackupPruneCmd() *cobra.Command {
	var keep int
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "删除超出保留份数的旧备份",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := resolveAccount(cmd)
			if err != nil {
				return err
			}
			n, err := backup.Prune(ac, keep)
			if err != nil {
				return err
			}
			fmt.Printf("prune OK: removed=%d keep=%d\n", n, keep)
			return nil
		},
	}
	cmd.Flags().IntVar(&keep, "keep", 8, "保留最新 N 份")
	return cmd
}

func backupPath(ac *account.Context, id string) string {
	return fmt.Sprintf("%s/%s/%s", ac.BackupRoot, ac.AccountID, id)
}
