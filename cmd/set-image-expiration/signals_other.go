//go:build !unix

package main

import "context"

func osHandleSignals(ctx context.Context) (context.Context, context.CancelFunc) {
	return ctx, func() {}
}
