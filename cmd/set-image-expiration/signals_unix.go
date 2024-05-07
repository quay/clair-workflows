//go:build unix

package main

import (
	"context"
	"os/signal"
	"syscall"
)

func osHandleSignals(ctx context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGHUP)
}
