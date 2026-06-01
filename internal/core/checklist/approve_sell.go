package checklist

import (
	"context"
	"fmt"
	"time"

	"github.com/investment-assistant/investment-assistant/internal/core/ids"
	"github.com/investment-assistant/investment-assistant/internal/core/lot"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore"
	"github.com/investment-assistant/investment-assistant/internal/core/store/sqlstore/schema"
	"github.com/investment-assistant/investment-assistant/internal/core/store/yamlstore"
	"github.com/shopspring/decimal"
)

func (s *Service) approveSell(ctx context.Context, cs *schema.ChecklistSubmission) (*ApproveResult, error) {
	payload, err := ParseSellPayload(cs.PayloadJSON)
	if err != nil {
		return nil, err
	}
	if payload.ExecutionPrice <= 0 {
		return nil, fmt.Errorf("sell payload.execution_price 须 > 0（实际成交价，手动填写）")
	}
	if payload.SellShares <= 0 {
		return nil, fmt.Errorf("sell payload.sell_shares 须 > 0")
	}
	sellShares := decimal.NewFromFloat(payload.SellShares)
	execPrice := decimal.NewFromFloat(payload.ExecutionPrice)
	openLots, err := loadOpenLots(s.db, cs.Code)
	if err != nil {
		return nil, err
	}
	plan, matchMethod, err := resolveSellPlanShares(sellShares, openLots, payload.LotAllocationPlan)
	if err != nil {
		return nil, err
	}

	if len(payload.LotAllocationPlan) == 0 {
		if merged, mErr := mergePlanIntoPayload(cs.PayloadJSON, plan); mErr == nil {
			_ = sqlstore.UpdateChecklistPayload(s.db, cs.ID, merged)
			cs.PayloadJSON = merged
		}
	}

	snapJSON, err := BuildBuySnapshot(ctx, cs.Code, s.market)
	if err != nil {
		return nil, err
	}

	journalID, err := ids.Next(s.db, "j")
	if err != nil {
		return nil, err
	}
	snapID, err := ids.Next(s.db, "snap")
	if err != nil {
		return nil, err
	}

	now := nowISO()
	today := time.Now().Format("2006-01-02")
	lotIndex := lotByID(openLots)

	allocIDs := make([]string, len(plan))
	for i := range plan {
		allocIDs[i], err = ids.Next(s.db, "la")
		if err != nil {
			return nil, err
		}
	}

	port, _ := yamlstore.LoadPortfolio(s.ac.PortfolioPath())
	portPatch, err := BuildSellPortfolioPatch(port, cs, payload, journalID, sellShares)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if err := sqlstore.InsertJournal(tx, &schema.Journal{
		ID: journalID, ActionType: schema.JournalActionSell,
		Code: cs.Code, Name: cs.Name, ChecklistSubmissionID: cs.ID,
		DataSnapshotID: snapID, PayloadJSON: cs.PayloadJSON,
		Summary: fmt.Sprintf("sell %s %s @%s", cs.Code, payload.SellReason, execPrice.StringFixed(2)),
		CreatedAt: now,
	}); err != nil {
		return nil, err
	}
	if err := sqlstore.InsertDataSnapshot(tx, &schema.DataSnapshot{
		ID: snapID, JournalID: journalID, SnapshotJSON: snapJSON,
		SchemaVersion: snapshotSchemaVersion, CreatedAt: now,
	}); err != nil {
		return nil, err
	}

	for i, item := range plan {
		ol, ok := lotIndex[item.LotID]
		if !ok {
			return nil, fmt.Errorf("lot %s 不在 open 列表", item.LotID)
		}
		allocID := allocIDs[i]
		realizedPct := lot.RealizedReturnPct(execPrice, ol.CostBasis)
		pnlAmount := lot.RealizedPnLAmount(execPrice, ol.CostBasis, item.AllocatedShares)
		proceeds := execPrice.Mul(item.AllocatedShares)

		if err := sqlstore.InsertLotAllocation(tx, &schema.LotAllocation{
			ID: allocID, SellJournalID: journalID, LotID: item.LotID,
			AllocatedPct:      item.AllocatedPct.StringFixed(4),
			CostBasisAtSale:   ol.CostBasis.StringFixed(4),
			RealizedReturnPct: realizedPct.StringFixed(4),
			AllocatedShares:   item.AllocatedShares.StringFixed(0),
			ProceedsAmount:    proceeds.StringFixed(4),
			RealizedPnLAmount: pnlAmount.StringFixed(4),
			MatchMethod:       matchMethod, UserConfirmed: 1, CreatedAt: now,
		}); err != nil {
			return nil, err
		}

		afterShares := lot.SharesAfterSell(ol.CurrentShares, item.AllocatedShares)
		afterPct := PctAfterShareSell(ol.CurrentPct, ol.CurrentShares, item.AllocatedShares)
		status := lot.LotStatusAfterShareSell(ol.CurrentShares, item.AllocatedShares)
		closeAt := ""
		if status == schema.LotStatusClosed {
			closeAt = today
		}
		if err := sqlstore.UpdateLotAfterSell(tx, item.LotID, afterPct.StringFixed(4), afterShares.StringFixed(0), status, closeAt); err != nil {
			return nil, err
		}
	}

	if err := sqlstore.UpdateRiskExceptionsJournalID(tx, cs.ID, journalID); err != nil {
		return nil, err
	}
	if err := sqlstore.ApproveChecklistUpdate(tx, cs.ID, now, journalID, "", ""); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	bundle := &YamlBundle{Portfolio: portPatch}
	res := &ApproveResult{ChecklistID: cs.ID, JournalID: journalID, SnapshotID: snapID}
	if err := ApplyYamlBundle(s.ac, bundle); err != nil {
		srID, _ := RecordSyncRepair(s.db, s.ac, cs.ID, journalID, bundle, err)
		res.SyncRepairID = srID
	} else {
		res.YAMLSynced = true
	}
	return res, nil
}
