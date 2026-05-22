package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "打印版本与 AccountContext",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := resolveAccount(cmd)
			if err != nil {
				return err
			}
			fmt.Printf("inv %s\n", Version)
			fmt.Printf("account_id: %s\n", ac.AccountID)
			fmt.Printf("data_root:  %s\n", ac.DataRoot)
			fmt.Printf("db_path:    %s\n", ac.DBPath)
			return nil
		},
	}
}
