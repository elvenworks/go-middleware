package middleware

import "context"

// SetLabelTransactionAPM does nothing. It is kept so that v1 callers keep compiling after the
// library dropped its Elastic APM dependency.
//
// Deprecated: put identifiers in structured log fields or OpenTelemetry span attributes
// instead. It will be removed in the next major version.
//
// Example: SetLabelTransactionAPM(&ctx, &struct{ OrgUid string }{OrgUid: "abc"})
func SetLabelTransactionAPM[T any](ctx *context.Context, obj *T) {}
