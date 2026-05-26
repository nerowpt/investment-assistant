package cli

import (
	"fmt"

	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
	"github.com/spf13/cobra"
)

func newTagsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tags",
		Short: "受控标签词表",
	}
	cmd.AddCommand(newTagsListCmd(), newTagsAddCmd(), newTagsDisableCmd(), newTagsSuggestCmd(), newTagsConfirmCmd(), newTagsRejectCmd())
	return cmd
}

func newTagsListCmd() *cobra.Command {
	var layer string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "查看词表",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := resolveAccount(cmd)
			if err != nil {
				return err
			}
			if err := ac.EnsureInitialized(); err != nil {
				return err
			}
			tags, err := yamlstore.LoadControlledTags(ac.ControlledTagsPath())
			if err != nil {
				return err
			}
			printLayer := func(name string, list []yamlstore.ControlledTag) {
				if layer != "" && layer != name {
					return
				}
				fmt.Printf("[%s]\n", name)
				for _, tag := range list {
					enabled := "true"
					if tag.Enabled != nil && !*tag.Enabled {
						enabled = "false"
					}
					fmt.Printf("  %s\t%s\t%s\tenabled=%s\n", tag.ID, tag.Dimension, tag.Label, enabled)
				}
			}
			printLayer("system", tags.System)
			printLayer("user", tags.User)
			printLayer("suggested", tags.Suggested)
			return nil
		},
	}
	cmd.Flags().StringVar(&layer, "layer", "", "system|user|suggested")
	return cmd
}

func newTagsAddCmd() *cobra.Command {
	var id, label, dimension string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "追加 user 层标签",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := resolveAccount(cmd)
			if err != nil {
				return err
			}
			if err := ac.EnsureInitialized(); err != nil {
				return err
			}
			path := ac.ControlledTagsPath()
			tags, err := yamlstore.LoadControlledTags(path)
			if err != nil {
				return err
			}
			if id == "" || label == "" || dimension == "" {
				return fmt.Errorf("须指定 --id --label --dimension")
			}
			if err := tags.ValidateNewUserTagID(id); err != nil {
				return err
			}
			tags.User = append(tags.User, yamlstore.ControlledTag{
				ID:        id,
				Label:     label,
				Dimension: dimension,
			})
			if err := yamlstore.SaveControlledTags(path, tags); err != nil {
				return err
			}
			fmt.Printf("tags add OK: %s\n", id)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "标签 id")
	cmd.Flags().StringVar(&label, "label", "", "中文标签名")
	cmd.Flags().StringVar(&dimension, "dimension", "", "sector|theme|event|...")
	return cmd
}

func newTagsDisableCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "disable [id]",
		Short: "禁用 system 层标签",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := resolveAccount(cmd)
			if err != nil {
				return err
			}
			if err := ac.EnsureInitialized(); err != nil {
				return err
			}
			path := ac.ControlledTagsPath()
			tags, err := yamlstore.LoadControlledTags(path)
			if err != nil {
				return err
			}
			id = args[0]
			found := false
			for i, tag := range tags.System {
				if tag.ID == id {
					f := false
					tags.System[i].Enabled = &f
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("system 层未找到 id: %s", id)
			}
			if err := yamlstore.SaveControlledTags(path, tags); err != nil {
				return err
			}
			fmt.Printf("tags disable OK: %s\n", id)
			return nil
		},
	}
	_ = id
	return cmd
}

func newTagsSuggestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "suggest",
		Short: "写入 suggested 层（MVP-1 stub）",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("tags suggest 尚未在 MVP-1 实现，词表结构已就绪（见 docs/03 §10C.10）")
		},
	}
}

func newTagsConfirmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "confirm",
		Short: "suggested → user（MVP-1 stub）",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("tags confirm 尚未在 MVP-1 实现，词表结构已就绪（见 docs/03 §10C.10）")
		},
	}
}

func newTagsRejectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reject",
		Short: "删除 suggested 项（MVP-1 stub）",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("tags reject 尚未在 MVP-1 实现，词表结构已就绪（见 docs/03 §10C.10）")
		},
	}
}
