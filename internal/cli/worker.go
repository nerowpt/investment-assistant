package cli

import (
	"context"
	"fmt"
	"os"

	dataworkerv1 "github.com/investment-assistant/investment-assistant/gen/go/dataworker/v1"
	"github.com/investment-assistant/investment-assistant/internal/core/account"
	"github.com/investment-assistant/investment-assistant/internal/worker"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func newWorkerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worker",
		Short: "Python data-worker gRPC（H3）",
	}
	cmd.AddCommand(
		newWorkerHealthCmd(),
		newWorkerRestartCmd(),
		newWorkerQuoteCmd(),
		newWorkerValuationCmd(),
		newWorkerAnnouncementsCmd(),
	)
	return cmd
}

func newWorkerHealthCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "health",
		Short: "探活 data-worker（必要时自动启动）",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, client, err := openWorkerClient(cmd)
			if err != nil {
				return err
			}
			defer client.Close()
			if !worker.WorkerMainExists() {
				return fmt.Errorf("未找到 Python worker：%s（请安装依赖，见 services/data-worker/README.md）", worker.RepoWorkerPath())
			}
			res, err := client.HealthCheck(context.Background())
			if err != nil {
				return err
			}
			if asJSON {
				return printProtoJSON(res)
			}
			fmt.Printf("worker OK: version=%s python=%s providers=%v\nport_file=%s\n",
				res.GetVersion(), res.GetPythonVersion(), res.GetProviders(), ac.WorkerPortPath())
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON 输出")
	return cmd
}

func newWorkerRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "重启 Python worker（改代码后或 valuation/quote 异常时用）",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := resolveAccount(cmd)
			if err != nil {
				return err
			}
			if err := worker.RestartWorker(ac); err != nil {
				return err
			}
			fmt.Println("worker 已重启准备：旧进程已停止，worker.port 已清除")
			fmt.Println("请再执行: inv worker health  （将拉起新进程）")
			fmt.Println("若仍失败，查看 data/.run/worker.log")
			return nil
		},
	}
}

func newWorkerQuoteCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "quote [code]",
		Short: "拉取实时行情",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, err := openWorkerClient(cmd)
			if err != nil {
				return err
			}
			defer client.Close()
			res, err := client.FetchQuote(context.Background(), args[0])
			if err != nil {
				return err
			}
			if asJSON {
				return printProtoJSON(res)
			}
			p := res.GetProvenance()
			fmt.Printf("%s %s  price=%.2f change_pct=%.2f%% tier=%s source=%s\n",
				res.GetCode(), res.GetName(), res.GetPrice(), res.GetChangePct(),
				p.GetTier(), p.GetSource())
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON 输出")
	return cmd
}

func newWorkerValuationCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "valuation [code]",
		Short: "拉取估值指标",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, err := openWorkerClient(cmd)
			if err != nil {
				return err
			}
			defer client.Close()
			res, err := client.FetchValuation(context.Background(), args[0])
			if err != nil {
				return err
			}
			if asJSON {
				return printProtoJSON(res)
			}
			fmt.Printf("%s PE=%.2f PB=%.2f as_of=%s tier=%s\n",
				res.GetCode(), res.GetPeTtm(), res.GetPb(), res.GetAsOfDate(),
				res.GetProvenance().GetTier())
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON 输出")
	return cmd
}

func newWorkerAnnouncementsCmd() *cobra.Command {
	var codes []string
	var since string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "announcements",
		Short: "拉取公告列表（简版）",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(codes) == 0 {
				return fmt.Errorf("须指定 --code（可重复）")
			}
			_, client, err := openWorkerClient(cmd)
			if err != nil {
				return err
			}
			defer client.Close()
			res, err := client.FetchAnnouncements(context.Background(), &dataworkerv1.FetchAnnouncementsRequest{
				Codes: codes,
				Since: since,
				Kind:  dataworkerv1.AnnouncementKind_ANNOUNCEMENT_KIND_ALL,
			})
			if err != nil {
				return err
			}
			if asJSON {
				return printProtoJSON(res)
			}
			for _, it := range res.GetItems() {
				fmt.Printf("%s\t%s\t%s\n", it.GetCode(), it.GetPublishedAt(), it.GetTitle())
			}
			for _, e := range res.GetErrors() {
				fmt.Fprintf(os.Stderr, "error %s: %s\n", e.GetCode(), e.GetMessage())
			}
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&codes, "code", nil, "标的代码")
	cmd.Flags().StringVar(&since, "since", "", "起始时间 ISO8601")
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON 输出")
	return cmd
}

func openWorkerClient(cmd *cobra.Command) (*account.Context, *worker.Client, error) {
	ac, err := resolveAccount(cmd)
	if err != nil {
		return nil, nil, err
	}
	if err := ac.EnsureInitialized(); err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(ac.RunDir(), 0o755); err != nil {
		return nil, nil, err
	}
	return ac, worker.NewClient(ac), nil
}

func printProtoJSON(msg proto.Message) error {
	b, err := protojson.MarshalOptions{Indent: "  ", EmitUnpopulated: true}.Marshal(msg)
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
