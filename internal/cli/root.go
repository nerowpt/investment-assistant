// Package cli 实现 inv 命令行（对齐 03 §十C、04 §十一）。
package cli

import (
	"fmt"
	"os"

	"github.com/investment-assistant/investment-assistant/internal/core/account"
	"github.com/spf13/cobra"
)

// Version 由 main 注入。
var Version = "dev"

// Execute 运行根命令。
func Execute() {
	if err := NewRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// NewRoot 构造 cobra 根命令。
func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "inv",
		Short: "个人投资助手 CLI",
	}
	root.PersistentFlags().String("account", "", "account id（覆盖 IA_ACCOUNT_ID）")
	root.AddCommand(newVersionCmd())
	root.AddCommand(newDoctorCmd())
	root.AddCommand(newLibraryCmd())
	root.AddCommand(newChecklistCmd())
	root.AddCommand(newTagsCmd())
	root.AddCommand(newWorkerCmd())
	root.AddCommand(newBackupCmd())
	root.AddCommand(newMCPCmd())
	root.AddCommand(newServeCmd())
	root.AddCommand(newAPICmd())
	return root
}

func resolveAccount(cmd *cobra.Command) (*account.Context, error) {
	ac, err := account.ResolveFromEnv()
	if err != nil {
		return nil, err
	}
	flagAccount, _ := cmd.Flags().GetString("account")
	if flagAccount != "" {
		return account.WithAccount(ac.DataRoot, flagAccount)
	}
	return ac, nil
}
