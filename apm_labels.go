package middleware

import (
	"context"
	"fmt"
	"reflect"

	"github.com/sirupsen/logrus"
	"go.elastic.co/apm/v2"
)

func SetLabelTransactionAPM[T any](ctx *context.Context, obj *T) {
	transaction := apm.TransactionFromContext(*ctx)
	if transaction == nil {
		logrus.Errorf("No transaction found in context: %+v", *obj)
		return
	}

	value := reflect.ValueOf(*obj)
	typ := reflect.TypeOf(*obj)

	if typ.Kind() == reflect.Map {

		for _, key := range value.MapKeys() {
			val := value.MapIndex(key)
			transaction.Context.SetLabel(fmt.Sprintf("%v", key), val)
		}
		return
	}

	for i := 0; i < value.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := value.Field(i)

		transaction.Context.SetLabel(field.Name, fmt.Sprintf("%v", fieldValue))
	}
}
