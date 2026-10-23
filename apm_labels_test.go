package middleware

import (
	"context"
	"strings"
	"testing"

	logtest "github.com/sirupsen/logrus/hooks/test"
	"go.elastic.co/apm/v2/apmtest"
	"go.elastic.co/apm/v2/model"
)

type labeledObject struct {
	OrgUid string
	Count  int
}

// labelsOf runs f inside a recorded APM transaction and returns the labels it ended with.
func labelsOf(t *testing.T, f func(ctx context.Context)) map[string]interface{} {
	t.Helper()
	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()

	tx, _, _ := tracer.WithTransaction(f)
	return tagsToMap(tx.Context)
}

func tagsToMap(ctx *model.Context) map[string]interface{} {
	labels := map[string]interface{}{}
	if ctx == nil {
		return labels
	}
	for _, item := range ctx.Tags {
		labels[item.Key] = item.Value
	}
	return labels
}

func assertLabel(t *testing.T, labels map[string]interface{}, key, want string) {
	t.Helper()
	if got := labels[key]; got != want {
		t.Fatalf("label %q = %v, want %q (all labels: %v)", key, got, want, labels)
	}
}

func TestSetLabelTransactionAPMStruct(t *testing.T) {
	labels := labelsOf(t, func(ctx context.Context) {
		obj := labeledObject{OrgUid: "org-1", Count: 3}
		SetLabelTransactionAPM(&ctx, &obj)
	})

	assertLabel(t, labels, "OrgUid", "org-1")
	assertLabel(t, labels, "Count", "3")
}

func TestSetLabelTransactionAPMPointerToStruct(t *testing.T) {
	labels := labelsOf(t, func(ctx context.Context) {
		obj := &labeledObject{OrgUid: "org-2"}
		SetLabelTransactionAPM(&ctx, &obj)
	})

	assertLabel(t, labels, "OrgUid", "org-2")
}

func TestSetLabelTransactionAPMMap(t *testing.T) {
	labels := labelsOf(t, func(ctx context.Context) {
		obj := map[string]int{"retries": 2}
		SetLabelTransactionAPM(&ctx, &obj)
	})

	assertLabel(t, labels, "retries", "2")
}

func TestSetLabelTransactionAPMIgnoresUnsupportedKinds(t *testing.T) {
	hook := logtest.NewGlobal()
	labels := labelsOf(t, func(ctx context.Context) {
		number := 42
		SetLabelTransactionAPM(&ctx, &number)
	})

	if len(labels) != 0 || len(hook.AllEntries()) != 1 {
		t.Fatalf("labels = %v, log entries = %d; want no labels and one warning", labels, len(hook.AllEntries()))
	}
}

func TestSetLabelTransactionAPMIgnoresNilValuesSilently(t *testing.T) {
	hook := logtest.NewGlobal()
	labels := labelsOf(t, func(ctx context.Context) {
		var nilPointer *labeledObject
		SetLabelTransactionAPM(&ctx, &nilPointer)
		SetLabelTransactionAPM[labeledObject](&ctx, nil)
		SetLabelTransactionAPM[labeledObject](nil, &labeledObject{})
	})

	if len(labels) != 0 || len(hook.AllEntries()) != 0 {
		t.Fatalf("labels = %v, log entries = %d; want none for nil values", labels, len(hook.AllEntries()))
	}
}

func TestSetLabelTransactionAPMWithoutTransactionDoesNotLogObject(t *testing.T) {
	hook := logtest.NewGlobal()
	ctx := context.Background()
	obj := labeledObject{OrgUid: "org-secret"}

	SetLabelTransactionAPM(&ctx, &obj)

	for _, entry := range hook.AllEntries() {
		if line, _ := entry.String(); strings.Contains(line, "org-secret") {
			t.Fatalf("log entry exposes the object contents: %s", line)
		}
	}
}
