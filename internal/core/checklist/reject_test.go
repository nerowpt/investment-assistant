package checklist

import (
	"context"
	"testing"
)

func TestRejectSubmittedChecklist(t *testing.T) {
	_, _, svc := setupChecklistTest(t)

	draft, err := svc.CreateDraft(DraftInput{
		ChecklistType: "buy", Code: "600519", Name: "贵州茅台", PayloadJSON: buyPayloadJSON(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Submit(SubmitInput{ID: draft.ID}); err != nil {
		t.Fatal(err)
	}

	res, err := svc.Reject(RejectInput{ID: draft.ID, Reason: "误建 buy，标的已在 holding"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "rejected" {
		t.Fatalf("status=%s", res.Status)
	}

	cs, err := svc.Get(draft.ID)
	if err != nil || cs.Status != "rejected" {
		t.Fatalf("db status=%s err=%v", cs.Status, err)
	}
	if rejectReasonFromPayload(cs.PayloadJSON) != "误建 buy，标的已在 holding" {
		t.Fatalf("missing _reject_meta: %s", cs.PayloadJSON)
	}

	_, err = svc.Approve(context.Background(), draft.ID)
	if err == nil {
		t.Fatal("expected approve error on rejected")
	}
}

func TestRejectDraft(t *testing.T) {
	_, _, svc := setupChecklistTest(t)

	draft, err := svc.CreateDraft(DraftInput{
		ChecklistType: "watch", Code: "600519", Name: "贵州茅台",
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := svc.Reject(RejectInput{ID: draft.ID, Reason: "暂不入观察池"})
	if err != nil || res.Status != "rejected" {
		t.Fatalf("reject draft: %+v err=%v", res, err)
	}
}

func TestRejectRequiresReason(t *testing.T) {
	_, _, svc := setupChecklistTest(t)

	draft, err := svc.CreateDraft(DraftInput{ChecklistType: "watch", Code: "600519"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Reject(RejectInput{ID: draft.ID})
	if err == nil {
		t.Fatal("expected reason required")
	}
}

func TestRejectApprovedFails(t *testing.T) {
	_, _, svc := setupChecklistTest(t)

	draft, err := svc.CreateDraft(DraftInput{
		ChecklistType: "buy", Code: "000001", Name: "平安银行", PayloadJSON: buyPayloadJSON(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Submit(SubmitInput{ID: draft.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Approve(context.Background(), draft.ID); err != nil {
		t.Fatal(err)
	}
	_, err = svc.Reject(RejectInput{ID: draft.ID, Reason: "too late"})
	if err == nil {
		t.Fatal("expected reject approved error")
	}
}

func TestRejectIdempotent(t *testing.T) {
	_, _, svc := setupChecklistTest(t)

	draft, err := svc.CreateDraft(DraftInput{ChecklistType: "watch", Code: "600519"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Reject(RejectInput{ID: draft.ID, Reason: "cancel"}); err != nil {
		t.Fatal(err)
	}
	res, err := svc.Reject(RejectInput{ID: draft.ID})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "rejected" {
		t.Fatal("idempotent reject")
	}
}
