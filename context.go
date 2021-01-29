package worker

import (
	"context"
)

type contextKey string

const (
	idContextKey = contextKey("id")
)

func contextWithID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, idContextKey, id)
}

// IDFromContext retrieves worker id from context.
func IDFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(idContextKey)
	if v == nil {
		return "", false
	}
	return v.(string), true
}
