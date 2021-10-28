/*
 *
 * k6 - a next-generation load testing tool
 * Copyright (C) 2021 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

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
