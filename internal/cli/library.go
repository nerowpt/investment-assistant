package cli

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/investment-assistant/investment-assistant/internal/core/account"
	libsvc "github.com/investment-assistant/investment-assistant/internal/core/library"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"github.com/spf13/cobra"
)

func newLibraryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "library",
		Short: "L1 研究素材归纳",
	}
	cmd.AddCommand(
		newLibraryIngestCmd(),
		newLibraryCandidatesCmd(),
		newLibraryPromoteCmd(),
		newLibrarySupplementCmd(),
		newLibraryDismissCmd(),
		newLibraryReviewCmd(),
		newLibraryListCmd(),
		newLibraryShowCmd(),
		newLibrarySearchCmd(),
		newLibraryArchiveCmd(),
		newLibraryPruneCmd(),
		newLibraryMergeCmd(),
		newLibraryLinkClusterCmd(),
	)
	return cmd
}

func newLibraryIngestCmd() *cobra.Command {
	var url, file, text, title, source, tier, contentType, mediaType string
	var stocks []string
	var noReview bool

	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "主动录入 → candidate",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, db, svc, err := openLibraryService(cmd)
			if err != nil {
				return err
			}
			defer db.Close()

			res, err := svc.Ingest(libsvc.IngestInput{
				URL:         url,
				FilePath:    file,
				Text:        text,
				Title:       title,
				Source:      source,
				Tier:        tier,
				Stocks:      stocks,
				ContentType: contentType,
				MediaType:   mediaType,
				AutoDismiss: true,
			})
			if err != nil {
				return err
			}
			fmt.Printf("ingest OK: candidate=%s status=%s match_tier=%s dedup_key=%s\n",
				res.CandidateID, res.Status, res.MatchTier, res.DedupKey)
			if res.AutoAction != "" {
				fmt.Printf("auto: %s\n", res.AutoAction)
			}
			if !noReview && res.Status == "pending" {
				return runReviewInteractive(svc, db, res.CandidateID, false)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&url, "url", "", "网页 URL")
	cmd.Flags().StringVar(&file, "file", "", "本地文件路径")
	cmd.Flags().StringVar(&text, "text", "", "纯文本笔记")
	cmd.Flags().StringVar(&title, "title", "", "标题")
	cmd.Flags().StringVar(&source, "source", "", "来源名称")
	cmd.Flags().StringVar(&tier, "tier", "", "tier S/A/B/C/D")
	cmd.Flags().StringVar(&contentType, "content-type", "", "content_type")
	cmd.Flags().StringVar(&mediaType, "media-type", "", "media_type")
	cmd.Flags().StringSliceVar(&stocks, "stock", nil, "关联标的（可重复）")
	cmd.Flags().BoolVar(&noReview, "no-review", false, "仅创建 candidate，不进入 review")
	return cmd
}

func newLibraryCandidatesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "candidates",
		Short: "候选队列",
	}
	cmd.AddCommand(newLibraryCandidatesListCmd(), newLibraryCandidatesShowCmd(), newLibraryCandidatesExpireCmd())
	return cmd
}

func newLibraryCandidatesListCmd() *cobra.Command {
	var status, matchTier string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出候选",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, db, _, err := openLibraryService(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			rows, err := sqlstore.ListLibraryCandidates(db, status, matchTier)
			if err != nil {
				return err
			}
			for _, r := range rows {
				fmt.Printf("%s\t%s\t%s\t%s\tmatch=%s\n", r.ID, r.Status, r.Tier, r.Title, r.MatchTier)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "pending|dismissed|promoted|expired")
	cmd.Flags().StringVar(&matchTier, "match-tier", "", "none|exact|near|theme")
	return cmd
}

func newLibraryCandidatesShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [lc_id]",
		Short: "查看候选详情",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, db, _, err := openLibraryService(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			c, err := sqlstore.GetLibraryCandidate(db, args[0])
			if err != nil {
				return err
			}
			fmt.Printf("id: %s\nstatus: %s\nmatch_tier: %s\ntitle: %s\ndedup_key: %s\nsimilarity: %s\n",
				c.ID, c.Status, c.MatchTier, c.Title, c.DedupKey, c.SimilarityJSON)
			return nil
		},
	}
}

func newLibraryCandidatesExpireCmd() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "expire",
		Short: "TTL 180d 过期 pending → expired",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, db, svc, err := openLibraryService(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			n, err := svc.ExpireCandidates(dryRun)
			if err != nil {
				return err
			}
			if dryRun {
				fmt.Printf("dry-run: 将过期 %d 条 candidate\n", n)
			} else {
				fmt.Printf("expire OK: %d 条 candidate → expired\n", n)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "仅统计")
	return cmd
}

func newLibraryPromoteCmd() *cobra.Command {
	var contentType, mediaType, tier, summary string
	var tags, stocks []string
	var yes bool

	cmd := &cobra.Command{
		Use:   "promote [lc_id]",
		Short: "candidate → 新 library_item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, db, svc, err := openLibraryService(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			if !yes {
				fmt.Println("将 promote 为新 library_item（--yes 跳过确认）")
			}
			libID, err := svc.Promote(libsvc.PromoteInput{
				CandidateID: args[0],
				ContentType: contentType,
				MediaType:   mediaType,
				Tier:        tier,
				Tags:        tags,
				Stocks:      stocks,
				Summary:     summary,
			})
			if err != nil {
				return err
			}
			fmt.Printf("promote OK: %s\n", libID)
			return nil
		},
	}
	cmd.Flags().StringVar(&contentType, "content-type", "", "announcement|report|...")
	cmd.Flags().StringVar(&mediaType, "media-type", "", "text|pdf|html|structured")
	cmd.Flags().StringVar(&tier, "tier", "", "S/A/B/C/D")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "受控标签 id")
	cmd.Flags().StringSliceVar(&stocks, "stock", nil, "关联标的")
	cmd.Flags().StringVar(&summary, "summary", "", "用户摘要")
	cmd.Flags().BoolVar(&yes, "yes", false, "跳过确认")
	return cmd
}

func newLibrarySupplementCmd() *cobra.Command {
	var into, note string
	var tagsAdd, tagsRemove []string
	var yes bool

	cmd := &cobra.Command{
		Use:   "supplement [lc_id]",
		Short: "candidate 追加到已有 item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, db, svc, err := openLibraryService(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			if into == "" {
				return fmt.Errorf("须指定 --into lib_*")
			}
			if !yes {
				fmt.Printf("将 supplement 到 %s\n", into)
			}
			if err := svc.Supplement(libsvc.SupplementInput{
				CandidateID: args[0],
				IntoItemID:  into,
				Note:        note,
				TagsAdd:     tagsAdd,
				TagsRemove:  tagsRemove,
			}); err != nil {
				return err
			}
			fmt.Println("supplement OK")
			return nil
		},
	}
	cmd.Flags().StringVar(&into, "into", "", "目标 library_item id")
	cmd.Flags().StringVar(&note, "note", "", "补充说明")
	cmd.Flags().StringSliceVar(&tagsAdd, "tags-add", nil, "合并追加 tags")
	cmd.Flags().StringSliceVar(&tagsRemove, "tags-remove", nil, "合并移除 tags")
	cmd.Flags().BoolVar(&yes, "yes", false, "跳过确认")
	return cmd
}

func newLibraryDismissCmd() *cobra.Command {
	var reason string
	cmd := &cobra.Command{
		Use:   "dismiss [lc_id]",
		Short: "丢弃 candidate",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, db, svc, err := openLibraryService(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			if err := svc.Dismiss(args[0], reason); err != nil {
				return err
			}
			fmt.Println("dismiss OK")
			return nil
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "丢弃原因")
	return cmd
}

func newLibraryReviewCmd() *cobra.Command {
	var id string
	var batch bool
	var limit int
	var matchTier string

	cmd := &cobra.Command{
		Use:   "review",
		Short: "交互式归纳",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, db, svc, err := openLibraryService(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			if batch {
				rows, err := sqlstore.ListLibraryCandidates(db, "pending", matchTier)
				if err != nil {
					return err
				}
				n := 0
				for _, r := range rows {
					if limit > 0 && n >= limit {
						break
					}
					fmt.Printf("\n=== %s ===\n", r.ID)
					if err := runReviewInteractive(svc, db, r.ID, true); err != nil {
						if err.Error() == "review 用户退出" {
							return nil
						}
						return err
					}
					n++
				}
				return nil
			}
			if id == "" {
				return fmt.Errorf("须指定 --id 或 --batch")
			}
			return runReviewInteractive(svc, db, id, false)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "candidate id")
	cmd.Flags().BoolVar(&batch, "batch", false, "批量审阅 pending")
	cmd.Flags().IntVar(&limit, "limit", 20, "batch 上限")
	cmd.Flags().StringVar(&matchTier, "match-tier", "", "筛选 match_tier")
	return cmd
}

func newLibraryListCmd() *cobra.Command {
	var status, stock, tag string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出 library_items",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, db, _, err := openLibraryService(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			if status == "" {
				status = "active"
			}
			rows, err := sqlstore.ListLibraryItems(db, status, stock, tag)
			if err != nil {
				return err
			}
			for _, r := range rows {
				fmt.Printf("%s\t%s\t%s\t%s\t%s\n", r.ID, r.Status, r.Tier, r.ContentType, r.Title)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "active", "active|archived|merged")
	cmd.Flags().StringVar(&stock, "stock", "", "标的过滤")
	cmd.Flags().StringVar(&tag, "tag", "", "标签过滤")
	return cmd
}

func newLibraryShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [lib_id]",
		Short: "查看 library_item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, db, _, err := openLibraryService(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			item, err := sqlstore.GetLibraryItem(db, args[0])
			if err != nil {
				return err
			}
			fmt.Printf("id: %s\nstatus: %s\ntitle: %s\ntier: %s\ncontent_type: %s\nmedia_type: %s\ntags: %s\nstocks: %s\n",
				item.ID, item.Status, item.Title, item.Tier, item.ContentType, item.MediaType, item.TagsJSON, item.RelatedStocksJSON)
			return nil
		},
	}
}

func newLibrarySearchCmd() *cobra.Command {
	var query, stock string
	cmd := &cobra.Command{
		Use:   "search",
		Short: "标题/notes 模糊搜索（MVP-1 LIKE）",
		RunE: func(cmd *cobra.Command, args []string) error {
			if query == "" {
				return fmt.Errorf("须指定 --query")
			}
			_, db, _, err := openLibraryService(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			rows, err := sqlstore.SearchLibraryItems(db, query, stock)
			if err != nil {
				return err
			}
			for _, r := range rows {
				fmt.Printf("%s\t%s\n", r.ID, r.Title)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "搜索关键词")
	cmd.Flags().StringVar(&stock, "stock", "", "标的过滤")
	return cmd
}

func newLibraryArchiveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "archive [lib_id]",
		Short: "归档 library_item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, db, svc, err := openLibraryService(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			if err := svc.ArchiveItem(args[0]); err != nil {
				return err
			}
			fmt.Println("archive OK")
			return nil
		},
	}
}

func newLibraryPruneCmd() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "过期 pending candidates",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, db, svc, err := openLibraryService(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			n, err := svc.ExpireCandidates(dryRun)
			if err != nil {
				return err
			}
			fmt.Printf("prune: %d 条\n", n)
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "仅统计")
	return cmd
}

func newLibraryMergeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "merge [lib_id]",
		Short: "canonical 合并（MVP-2 stub）",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("library merge 尚未在 MVP-1 实现，计划在 MVP-2 完成（见 docs/03 §10C.7）")
		},
	}
}

func newLibraryLinkClusterCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "link-cluster [lc_id]",
		Short: "关联 cluster（MVP-1 stub）",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("library link-cluster 尚未在 MVP-1 实现，计划在 MVP-2 完成（见 docs/03 §10C.8）")
		},
	}
}

func openLibraryService(cmd *cobra.Command) (*account.Context, *sql.DB, *libsvc.Service, error) {
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
	svc, err := libsvc.NewService(ac, db)
	if err != nil {
		db.Close()
		return nil, nil, nil, err
	}
	return ac, db, svc, nil
}

func runReviewInteractive(svc *libsvc.Service, db *sql.DB, lcID string, batchMode bool) error {
	c, err := sqlstore.GetLibraryCandidate(db, lcID)
	if err != nil {
		return err
	}
	if c.Status != "pending" {
		fmt.Printf("跳过 %s（status=%s）\n", lcID, c.Status)
		return nil
	}

	fmt.Printf("候选: %s | tier=%s | match=%s\n%s\n", c.Title, c.Tier, c.MatchTier, c.SimilarityJSON)
	if c.MatchTier == "exact" {
		fmt.Println("建议: dismiss（精确重复）")
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("选择 [1=promote 2=supplement 3=dismiss s=跳过 q=退出]: ")
	line, _ := reader.ReadString('\n')
	choice := strings.TrimSpace(line)

	switch choice {
	case "1", "promote":
		fmt.Print("content-type: ")
		ct, _ := reader.ReadString('\n')
		fmt.Print("media-type: ")
		mt, _ := reader.ReadString('\n')
		fmt.Print("tier: ")
		tr, _ := reader.ReadString('\n')
		fmt.Print("tags(逗号): ")
		tg, _ := reader.ReadString('\n')
		var tags []string
		for _, t := range strings.Split(strings.TrimSpace(tg), ",") {
			if t = strings.TrimSpace(t); t != "" {
				tags = append(tags, t)
			}
		}
		libID, err := svc.Promote(libsvc.PromoteInput{
			CandidateID: lcID,
			ContentType: strings.TrimSpace(ct),
			MediaType:   strings.TrimSpace(mt),
			Tier:        strings.TrimSpace(tr),
			Tags:        tags,
		})
		if err != nil {
			return err
		}
		fmt.Printf("promote OK: %s\n", libID)
	case "2", "supplement":
		fmt.Print("into lib_id: ")
		into, _ := reader.ReadString('\n')
		fmt.Print("note: ")
		note, _ := reader.ReadString('\n')
		if err := svc.Supplement(libsvc.SupplementInput{
			CandidateID: lcID,
			IntoItemID:  strings.TrimSpace(into),
			Note:        strings.TrimSpace(note),
		}); err != nil {
			return err
		}
		fmt.Println("supplement OK")
	case "3", "dismiss":
		if err := svc.Dismiss(lcID, "review_dismiss"); err != nil {
			return err
		}
		fmt.Println("dismiss OK")
	case "s", "":
		fmt.Println("跳过")
	case "q":
		if batchMode {
			return fmt.Errorf("review 用户退出")
		}
		return nil
	default:
		return fmt.Errorf("未知选择: %s", choice)
	}
	return nil
}
