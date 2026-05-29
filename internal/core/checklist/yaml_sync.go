package checklist

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/investment-assistant/investment-assistant/internal/core/account"
	"github.com/investment-assistant/investment-assistant/internal/core/ids"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore/schema"
	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
	"github.com/shopspring/decimal"
	"gopkg.in/yaml.v3"
)

// YamlBundle approve 后待写入的 Layer A YAML 集合（04 §20.10）。
type YamlBundle struct {
	Portfolio *yamlstore.Portfolio // nil = 不写
	Watchlist *yamlstore.Watchlist // nil = 不写
}

// ApplyYamlBundle 按固定顺序写 YAML；任一步失败返回 error（SQL 已提交，写 sync_repairs）。
func ApplyYamlBundle(ac *account.Context, bundle *YamlBundle) error {
	if bundle == nil {
		return nil
	}
	if bundle.Portfolio != nil {
		bundle.Portfolio.Meta.UpdatedAt = time.Now().Format(time.RFC3339)
		if err := yamlstore.SavePortfolio(ac.PortfolioPath(), bundle.Portfolio); err != nil {
			return fmt.Errorf("写入 portfolio.yaml: %w", err)
		}
	}
	if bundle.Watchlist != nil {
		bundle.Watchlist.Meta.UpdatedAt = time.Now().Format(time.RFC3339)
		if err := yamlstore.SaveWatchlist(ac.WatchlistPath(), bundle.Watchlist); err != nil {
			return fmt.Errorf("写入 watchlist.yaml: %w", err)
		}
	}
	return nil
}

// RecordSyncRepair YAML 写失败时记录修复队列（T5）。
func RecordSyncRepair(db *sql.DB, ac *account.Context, checklistID, journalID string, bundle *YamlBundle, err error) (string, error) {
	files := []string{}
	if bundle != nil && bundle.Portfolio != nil {
		files = append(files, ac.PortfolioPath())
	}
	if bundle != nil && bundle.Watchlist != nil {
		files = append(files, ac.WatchlistPath())
	}
	raw, _ := json.Marshal(files)
	repairID, idErr := ids.Next(db, "sr")
	if idErr != nil {
		return "", idErr
	}
	row := &schema.SyncRepair{
		ID:                    repairID,
		ChecklistSubmissionID: checklistID,
		JournalID:             journalID,
		YAMLFilesJSON:         string(raw),
		ErrorMessage:          err.Error(),
		Status:                "pending",
		CreatedAt:             time.Now().Format(time.RFC3339),
	}
	return repairID, sqlstore.InsertSyncRepair(db, row)
}

// BuildBuyPortfolioPatch 新建 holding position（buy approve）。
func BuildBuyPortfolioPatch(port *yamlstore.Portfolio, cs *schema.ChecklistSubmission, payload *BuyPayload, journalID, lotID string, costBasis decimal.Decimal, initialPct decimal.Decimal) (*yamlstore.Portfolio, error) {
	if port == nil {
		port = &yamlstore.Portfolio{SchemaVersion: yamlstore.PortfolioSchemaVersion}
	}
	for _, p := range port.Positions {
		if p.Code == cs.Code && p.State == "holding" {
			return nil, fmt.Errorf("%s 已在 holding，应走 add checklist", cs.Code)
		}
	}
	today := time.Now().Format("2006-01-02")
	pos := yamlstore.PortfolioPosition{
		Code:                     cs.Code,
		Name:                     cs.Name,
		State:                    "holding",
		PositionType:             payload.PositionType,
		PositionPct:              initialPct,
		CostBasis:                costBasis,
		EntryDate:                today,
		ThesisVersion:            1,
		InvestmentThesis:         payload.InvestmentThesis,
		TargetPrice:              anyToYAMLNode(payload.TargetPrice),
		StopLoss:                 anyToDecimal(payload.StopLoss),
		ReversalConditions:       payload.ReversalConditions,
		OpportunityCostBenchmark: payload.OpportunityCostBenchmark,
		Confidence:               payload.Confidence,
		RelatedLibraryIDs:        payload.RelatedLibraryIDs,
		LotIDs:                   []string{lotID},
		JournalIDs:               []string{journalID},
		WatchlistOriginID:        payload.WatchlistOriginID,
	}
	port.Positions = append(port.Positions, pos)
	return port, nil
}

// BuildAddPortfolioPatch 加仓更新 position_pct 与 lot_ids。
func BuildAddPortfolioPatch(port *yamlstore.Portfolio, cs *schema.ChecklistSubmission, payload *AddPayload, journalID, lotID string, addPct decimal.Decimal) (*yamlstore.Portfolio, error) {
	if port == nil {
		return nil, fmt.Errorf("portfolio 不存在，无法 add")
	}
	found := false
	for i, p := range port.Positions {
		if p.Code == cs.Code && p.State == "holding" {
			found = true
			port.Positions[i].PositionPct = p.PositionPct.Add(addPct)
			port.Positions[i].LotIDs = append(p.LotIDs, lotID)
			port.Positions[i].JournalIDs = append(p.JournalIDs, journalID)
			if payload.InvestmentThesis != "" {
				port.Positions[i].InvestmentThesis = payload.InvestmentThesis
				port.Positions[i].ThesisVersion++
			}
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("%s 不在 holding，应走 buy checklist", cs.Code)
	}
	return port, nil
}

// BuildWatchlistPatch 新增或更新观察项。
func BuildWatchlistPatch(wl *yamlstore.Watchlist, cs *schema.ChecklistSubmission, payload *WatchPayload, watchID string) (*yamlstore.Watchlist, error) {
	if wl == nil {
		wl = &yamlstore.Watchlist{SchemaVersion: yamlstore.WatchlistSchemaVersion}
	}
	now := time.Now().Format(time.RFC3339)
	entry := payload.SourceEntry
	if entry == "" {
		entry = "manual"
	}
	watchType := "stock"
	if cs.Code == "" {
		watchType = "theme"
	}
	item := yamlstore.WatchlistItem{
		ID:                watchID,
		Code:              cs.Code,
		Name:              cs.Name,
		WatchType:         watchType,
		State:             "watching",
		SourceEntry:       entry,
		WatchReason:       payload.WatchReason,
		Hypothesis:        payload.Hypothesis,
		KeyMetricsToWatch: payload.KeyMetricsToWatch,
		ExpectedTrigger:   payload.ExpectedTrigger,
		InvalidCondition:  payload.InvalidCondition,
		ReviewDate:        payload.ReviewDate,
		Priority:          payload.Priority,
		RelatedLibraryIDs: payload.RelatedLibraryIDs,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if item.Priority == "" {
		item.Priority = "medium"
	}
	if len(item.KeyMetricsToWatch) == 0 {
		item.KeyMetricsToWatch = []string{"待补充"}
	}
	wl.Items = append(wl.Items, item)
	return wl, nil
}

func anyToDecimal(v any) decimal.Decimal {
	switch n := v.(type) {
	case float64:
		return decimal.NewFromFloat(n)
	case int:
		return decimal.NewFromInt(int64(n))
	default:
		return decimal.Zero
	}
}

func anyToYAMLNode(v any) yaml.Node {
	if v == nil {
		return yaml.Node{Kind: yaml.ScalarNode, Value: "null"}
	}
	raw, err := yaml.Marshal(v)
	if err != nil {
		return yaml.Node{Kind: yaml.ScalarNode, Value: "null"}
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return yaml.Node{Kind: yaml.ScalarNode, Value: "null"}
	}
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		return *doc.Content[0]
	}
	return doc
}

func decimalFromPayloadPlan(plan map[string]any, key string) decimal.Decimal {
	if plan == nil {
		return decimal.Zero
	}
	return anyToDecimal(plan[key])
}
