package cli

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/investment-assistant/investment-assistant/internal/core/account"
	"github.com/investment-assistant/investment-assistant/internal/core/doctor"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	var scope string
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "数据一致性检查",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := resolveAccount(cmd)
			if err != nil {
				return err
			}
			if err := ac.EnsureInitialized(); err != nil {
				return err
			}

			switch scope {
			case "library":
				return doctorLibrary(ac.DBPath)
			case "portfolio":
				return doctorPortfolio(ac)
			case "watchlist":
				return doctorWatchlist(ac)
			case "all":
				if err := doctorLibrary(ac.DBPath); err != nil {
					return err
				}
				if err := doctorPortfolio(ac); err != nil {
					return err
				}
				return doctorWatchlist(ac)
			default:
				return fmt.Errorf("暂支持 --scope library|portfolio|watchlist|all，当前: %s", scope)
			}
		},
	}
	cmd.Flags().StringVar(&scope, "scope", "library", "检查范围")
	return cmd
}

func doctorLibrary(dbPath string) error {
	db, err := openMigratedDB(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	ver, err := sqlstore.SchemaVersion(db)
	if err != nil {
		return err
	}
	if ver == "" {
		return fmt.Errorf("schema_meta 无 schema_version")
	}

	missing, err := sqlstore.VerifyTables(db)
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		return fmt.Errorf("缺少表: %s", sqlstore.FormatMissing(missing))
	}

	libIssues := doctor.CheckLibrary(db)
	if len(libIssues) > 0 {
		return fmt.Errorf("library 校验失败:\n  - %s", strings.Join(libIssues, "\n  - "))
	}

	fmt.Printf("doctor OK (scope=library, schema_version=%s)\n", ver)
	return nil
}

func doctorPortfolio(ac *account.Context) error {
	db, err := openMigratedDB(ac.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	p, err := yamlstore.LoadPortfolio(ac.PortfolioPath())
	if err != nil {
		return fmt.Errorf("读取 portfolio.yaml: %w", err)
	}

	issues := doctor.CheckPortfolio(db, p)
	if len(issues) > 0 {
		return fmt.Errorf("portfolio 校验失败 (%d 项):\n%s", len(issues), doctor.FormatIssues(issues))
	}

	fmt.Println("doctor OK (scope=portfolio)")
	return nil
}

func doctorWatchlist(ac *account.Context) error {
	db, err := openMigratedDB(ac.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	w, err := yamlstore.LoadWatchlist(ac.WatchlistPath())
	if err != nil {
		return fmt.Errorf("读取 watchlist.yaml: %w", err)
	}

	p, err := yamlstore.LoadPortfolio(ac.PortfolioPath())
	if err != nil {
		return fmt.Errorf("读取 portfolio.yaml: %w", err)
	}

	issues := doctor.CheckWatchlist(db, w, p)
	if len(issues) > 0 {
		return fmt.Errorf("watchlist 校验失败:\n  - %s", strings.Join(issues, "\n  - "))
	}

	fmt.Println("doctor OK (scope=watchlist)")
	return nil
}

func openMigratedDB(dbPath string) (*sql.DB, error) {
	db, err := sqlstore.Open(dbPath)
	if err != nil {
		return nil, err
	}
	if err := sqlstore.MigrateUp(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
