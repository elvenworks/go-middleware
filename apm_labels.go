package middleware

import (
	"context"
	"fmt"
	"reflect"

	"github.com/sirupsen/logrus"
	"go.elastic.co/apm/v2"
)

// SetLabelTransactionAPM copies the fields of a struct (or the entries of a map) in obj to labels
// of the APM transaction stored in ctx. Pointers are followed; nil values and other kinds are
// ignored with a warning instead of panicking.
//
// Example: SetLabelTransactionAPM(&ctx, &struct{ OrgUid string }{OrgUid: "abc"})
func SetLabelTransactionAPM[T any](ctx *context.Context, obj *T) {
	if ctx == nil || *ctx == nil || obj == nil {
		return
	}
	transaction := apm.TransactionFromContext(*ctx)
	if transaction == nil {
		// The object itself is not logged: it may carry customer data.
		logrus.Warnf("No APM transaction found in context; skipping labels for %T", *obj)
		return
	}

	value := reflect.ValueOf(*obj)
	for value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return
		}
		value = value.Elem()
	}

	setLabels(transaction, value)
}

func setLabels(transaction *apm.Transaction, value reflect.Value) {
	switch value.Kind() {
	case reflect.Map:
		for _, key := range value.MapKeys() {
			transaction.Context.SetLabel(fmt.Sprintf("%v", key), fmt.Sprintf("%v", value.MapIndex(key)))
		}
	case reflect.Struct:
		typ := value.Type()
		for i := 0; i < value.NumField(); i++ {
			transaction.Context.SetLabel(typ.Field(i).Name, fmt.Sprintf("%v", value.Field(i)))
		}
	default:
		logrus.Warnf("SetLabelTransactionAPM expects a struct or map, got %s", value.Kind())
	}
}
