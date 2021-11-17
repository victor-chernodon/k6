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
	"sync"
)

// an event loop
type eventLoop struct {
	lock          sync.Mutex
	queue         []func()
	started       int
	wakeupCh      chan struct{} // maybe use sync.Cond ?
	reservedCount int
}

func newEventLoop() *eventLoop {
	return &eventLoop{
		wakeupCh: make(chan struct{}, 1),
	}
}

func (e *eventLoop) wakeup() {
	select {
	case e.wakeupCh <- struct{}{}:
	default:
	}
}

// reserve "reserves" a spot on the loop, preventing it from returning/finishing. The returning function will queue it's
// argument and wakeup the loop if needed and also unreserve the spot so that the loop can exit. If the eventLoop has
// since stopped it will return `false` and it will mean that this won't even be queued.
// Even if it's queued it doesn't mean that it will definitely be executed.
// this should be used instead of MakeHandledPromise if a promise will not be returned
// TODO better name
func (e *eventLoop) reserve() func(func()) bool {
	e.lock.Lock()
	e.reservedCount++
	started := e.started
	e.lock.Unlock()

	return func(f func()) bool {
		e.lock.Lock()
		if started != e.started {
			e.lock.Unlock()
			return false
		}
		e.queue = append(e.queue, f)
		e.reservedCount--
		e.lock.Unlock()
		e.wakeup()
		return true
	}
}

// start will run the event loop until it's empty and there are no reserved spots
// or the context is done. The provided function will be queued.
// After it returns any Reserved function from this start will not be queued even if the eventLoop is restarted
//nolint:cyclop
func (e *eventLoop) start(ctx context.Context, f func()) {
	e.lock.Lock()
	e.started++
	e.reservedCount = 0
	e.queue = append(e.queue, f)
	e.lock.Unlock()
	done := ctx.Done()
	for {
		select { // check if done
		case <-done:
			return
		default:
		}

		// acquire the queue
		e.lock.Lock()
		queue := e.queue
		e.queue = make([]func(), 0, len(queue))
		reserved := e.reservedCount != 0
		e.lock.Unlock()

		if len(queue) == 0 {
			if !reserved { // we have empty queue and nothing that reserved a spot
				return
			}
			select { // wait until the reserved is done
			case <-done:
				return
			case <-e.wakeupCh:
			}
		}

		for _, f := range queue {
			// run each function in the queue if not done
			select {
			case <-done:
				return
			default:
				f()
			}
		}
	}
}
