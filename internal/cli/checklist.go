package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	"github.com/investment-assistant/investment-assistant/internal/core/account"
	chksvc "github.com/investment-assistant/investment-assistant/internal/core/checklist"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"github.com/investment-assistant/investment-assistant/internal/worker"
	"github.com/spf13/cobra"
)

func newChecklistCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checklist",
		Short: "决策 Checklist（H4 draft/submit + H5 approve）",
	}
	cmd.AddCommand(
		newChecklistDraftCmd(),
		newChecklistSubmitCmd(),
		newChecklistApproveCmd(),
		newChecklistShowCmd(),
		newChecklistListCmd(),
	)
	return cmd
}

func newChecklistDraftCmd() *cobra.Command {
	var typ, code, name, file string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "draft",
		Short: "创建 draft checklist",
		RunE: func(cmd *cobra.Command, args []string) error {
			if typ == "" {
				return fmt.Errorf("须指定 --type")
			}
			_, db, svc, err := openChecklistService(cmd)
			if err != nil {
				return err
			}
			defer db.Close()

			payload := ""
			if file != "" {
				raw, err := os.ReadFile(file)
				if err != nil {
					return err
				}
				payload = string(raw)
			}
			res, err := svc.CreateDraft(chksvc.DraftInput{
				ChecklistType: typ,
				Code:          code,
				Name:          name,
				PayloadJSON:   payload,
			})
			if err != nil {
				return err
			}
			if asJSON {
				return printJSON(res)
			}
			fmt.Printf("draft OK: id=%s status=%s\n", res.ID, res.Status)
			if file == "" {
				fmt.Println("提示: 已写入默认模板 payload，请编辑后 submit（或用 --file 指定完整 JSON）")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&typ, "type", "", "checklist 类型: watch|buy|add|inspect|sell|review|import")
	cmd.Flags().StringVar(&code, "code", "", "标的代码")
	cmd.Flags().StringVar(&name, "name", "", "标的名称")
	cmd.Flags().StringVar(&file, "file", "", "payload JSON 文件路径")
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON 输出")
	return cmd
}

func newChecklistSubmitCmd() *cobra.Command {
	var emotionCheck, exceptionFile string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "submit [id]",
		Short: "提交 draft → submitted（运行 M7）",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, db, svc, err := openChecklistService(cmd)
			if err != nil {
				return err
			}
			defer db.Close()

			exceptionJSON := ""
			if exceptionFile != "" {
				raw, err := os.ReadFile(exceptionFile)
				if err != nil {
					return err
				}
				exceptionJSON = string(raw)
			}
			res, err := svc.Submit(chksvc.SubmitInput{
				ID:               args[0],
				EmotionSelfCheck: emotionCheck,
				ExceptionJSON:    exceptionJSON,
			})
			if err != nil {
				return err
			}
			if asJSON {
				return printJSON(res)
			}
			fmt.Printf("submit OK: id=%s status=%s hard_blocks=%d warnings=%d approve_blocked=%v\n",
				res.ID, res.Status, res.HardBlockCount, res.WarningCount, res.ApproveBlocked)
			if res.ApproveBlocked {
				fmt.Println("注意: 触发 hard_block，H5 approve 将被门禁（须已填 exception_json）")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&emotionCheck, "emotion-check", "", "fomo/greedy/anxious 二次确认文案")
	cmd.Flags().StringVar(&exceptionFile, "exception-file", "", "hard/soft 例外说明 JSON 文件")
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON 输出")
	return cmd
}

func newChecklistApproveCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "approve [id]",
		Short: "submitted → approved（journal/lot/snapshot + portfolio.yaml）",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, db, svc, err := openChecklistServiceWithMarket(cmd)
			if err != nil {
				return err
			}
			defer db.Close()

			res, err := svc.Approve(context.Background(), args[0])
			if err != nil {
				return err
			}
			if asJSON {
				return printJSON(res)
			}
			fmt.Printf("approve OK: checklist=%s status=approved\n", res.ChecklistID)
			if res.JournalID != "" {
				fmt.Printf("  journal=%s lot=%s snapshot=%s\n", res.JournalID, res.LotID, res.SnapshotID)
			}
			if res.InspectionID != "" {
				fmt.Printf("  inspection=%s\n", res.InspectionID)
			}
			if res.ReviewID != "" {
				fmt.Printf("  review=%s\n", res.ReviewID)
			}
			if res.WatchID != "" {
				fmt.Printf("  watchlist_item=%s\n", res.WatchID)
			}
			if res.SyncRepairID != "" {
				fmt.Printf("  警告: YAML 写入失败，已记录 sync_repair=%s（SQL 已提交）\n", res.SyncRepairID)
			} else if res.YAMLSynced {
				fmt.Println("  yaml: portfolio/watchlist 已同步")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON 输出")
	return cmd
}

func newChecklistShowCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "show [id]",
		Short: "查看 checklist 详情",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, db, svc, err := openChecklistService(cmd)
			if err != nil {
				return err
			}
			defer db.Close()

			cs, err := svc.Get(args[0])
			if err != nil {
				return err
			}
			if cs == nil {
				return fmt.Errorf("不存在: %s", args[0])
			}
			if asJSON {
				return printJSON(cs)
			}
			fmt.Printf("id=%s type=%s code=%s name=%s status=%s\n",
				cs.ID, cs.ChecklistType, cs.Code, cs.Name, cs.Status)
			fmt.Printf("created=%s submitted=%s\n", cs.CreatedAt, cs.SubmittedAt)
			if cs.RiskGuardrailResultJSON != "" {
				fmt.Printf("m7=%s\n", cs.RiskGuardrailResultJSON)
			}
			if cs.ExceptionJSON != "" {
				fmt.Printf("exception=%s\n", cs.ExceptionJSON)
			}
			fmt.Println("--- payload ---")
			fmt.Println(cs.PayloadJSON)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON 输出")
	return cmd
}

func newChecklistListCmd() *cobra.Command {
	var status, typ, code string
	var limit int
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出 checklist",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, db, svc, err := openChecklistService(cmd)
			if err != nil {
				return err
			}
			defer db.Close()

			rows, err := svc.List(sqlstore.ChecklistListFilter{
				Status: status,
				Type:   typ,
				Code:   code,
				Limit:  limit,
			})
			if err != nil {
				return err
			}
			if asJSON {
				return printJSON(rows)
			}
			for _, cs := range rows {
				fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\n",
					cs.ID, cs.ChecklistType, cs.Code, cs.Status, cs.CreatedAt, cs.SubmittedAt)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "draft|submitted|approved|rejected")
	cmd.Flags().StringVar(&typ, "type", "", "checklist 类型")
	cmd.Flags().StringVar(&code, "code", "", "标的代码")
	cmd.Flags().IntVar(&limit, "limit", 20, "最大条数")
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON 输出")
	return cmd
}

func openChecklistService(cmd *cobra.Command) (*account.Context, *sql.DB, *chksvc.Service, error) {
	ac, err := resolveAccount(cmd)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := ac.EnsureInitialized(); err != nil {
		return nil, nil, nil, err
	}
	db, err := openMigratedDB(ac.DBPath)
	if err != nil {
		return nil, nil, nil, err
	}
	return ac, db, chksvc.NewService(ac, db), nil
}

func openChecklistServiceWithMarket(cmd *cobra.Command) (*account.Context, *sql.DB, *chksvc.Service, error) {
	ac, db, svc, err := openChecklistService(cmd)
	if err != nil {
		return nil, nil, nil, err
	}
	client := worker.NewClient(ac)
	svc.SetMarketFetcher(&chksvc.WorkerMarketFetcher{Client: client})
	return ac, db, svc, nil
}

func printJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
