package cli

import (
	"fmt"

	"github.com/investment-assistant/investment-assistant/internal/api"
	"github.com/spf13/cobra"
)

func newAPICmd() *cobra.Command {
	var addr string
	cmd := &cobra.Command{
		Use:   "api",
		Short: "启动 HTTP API server（H8，供 uni-app 前端）",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := resolveAccount(cmd)
			if err != nil {
				return err
			}
			srv, err := api.NewServer(ac, api.Options{Addr: addr})
			if err != nil {
				return err
			}
			fmt.Printf("启动 HTTP API: %s\n", addr)
			return srv.Run()
		},
	}
	cmd.Flags().StringVar(&addr, "addr", ":8787", "监听地址")
	return cmd
}
