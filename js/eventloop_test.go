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
	f := func() { ran++ }
	loop.start(ctx, f)
	require.Equal(t, 1, ran)
	loop.start(ctx, f)
	require.Equal(t, 2, ran)
	loop.start(ctx, func() {
		f()
		loop.reserve()(f)
		cancel()
	})
	require.Equal(t, 3, ran)
}

func TestEventLoopReserve(t *testing.T) {
	t.Parallel()
	loop := newEventLoop()
	var ran int
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	start := time.Now()
	loop.start(ctx, func() {
		ran++
		r := loop.reserve()
		go func() {
			time.Sleep(time.Second)
			r(func() {
				ran++
			})
		}()
	})
	took := time.Since(start)
	require.Equal(t, 2, ran)
	require.Less(t, time.Second, took)
	require.Greater(t, time.Second+time.Millisecond*100, took)
}

func TestEventLoopReserveStopBetweenStarts(t *testing.T) {
	t.Parallel()
	loop := newEventLoop()
	var ran int
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()
	loop.start(ctx, func() {
		ran++
		r := loop.reserve()
		go func() {
			time.Sleep(time.Second)
			r(func() {
				ran++
			})
		}()
	})
	require.Equal(t, 1, ran)

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	loop.start(ctx, func() {
		ran++
		r := loop.reserve()
		go func() {
			time.Sleep(time.Second)
			r(func() {
				ran++
			})
		}()
	})
	require.Equal(t, 3, ran)
}
