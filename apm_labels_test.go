package middleware

import (
	"context"
	"testing"

	logtest "github.com/sirupsen/logrus/hooks/test"
)

type labeledObject struct {
	OrgUid string
}

func TestSetLabelTransactionAPMIsANoOp(t *testing.T) {
	hook := logtest.NewGlobal()
	defer hook.Reset()

	ctx := context.Background()
	var nilPointer *labeledObject
	number := 42

	SetLabelTransactionAPM(&ctx, &labeledObject{OrgUid: "abc"})
	SetLabelTransactionAPM(&ctx, &map[string]string{"orgUid": "abc"})
	SetLabelTransactionAPM(&ctx, &nilPointer)
	SetLabelTransactionAPM(&ctx, &number)
	SetLabelTransactionAPM[labeledObject](&ctx, nil)
	SetLabelTransactionAPM[labeledObject](nil, &labeledObject{})

	if ctx != context.Background() {
		t.Fatalf("context was replaced: got %v, want context.Background()", ctx)
	}
	if entries := hook.AllEntries(); len(entries) != 0 {
		t.Fatalf("expected no log entries, got %d: %v", len(entries), entries)
	}
}
