package js

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBasicEventLoop(t *testing.T) {
	t.Parallel()
	loop := newEventLoop()
	var ran int
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	loop.RunOnLoop(func() { ran++ })
	loop.Start(ctx)
	require.Equal(t, ran, 1)
	loop.RunOnLoop(func() { ran++ })
	loop.RunOnLoop(func() { ran++ })
	loop.Start(ctx)
	require.Equal(t, ran, 3)
	loop.RunOnLoop(func() { ran++; cancel() })
	loop.RunOnLoop(func() { ran++ })
	loop.Start(ctx)
	require.Equal(t, ran, 4)
}

func TestEventLoopReserve(t *testing.T) {
	t.Parallel()
	loop := newEventLoop()
	var ran int
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	loop.RunOnLoop(func() {
		ran++
		r := loop.Reserve()
		go func() {
			time.Sleep(time.Second)
			r(func() {
				ran++
			})
		}()
	})
	start := time.Now()
	loop.Start(ctx)
	took := time.Since(start)
	require.Equal(t, ran, 2)
	require.Greater(t, took, time.Second)
}
