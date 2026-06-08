package cli

import (
	invMCP "github.com/investment-assistant/investment-assistant/internal/mcp"
	"github.com/spf13/cobra"
)

func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "启动 MCP stdio server（Cursor 只读 9 tool）",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := resolveAccount(cmd)
			if err != nil {
				return err
			}
			return invMCP.RunStdio(ac)
		},
	}
	return cmd
}

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "MVP-1 等价于 inv mcp（stdio）",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := resolveAccount(cmd)
			if err != nil {
				return err
			}
			return invMCP.RunStdio(ac)
		},
	}
}
