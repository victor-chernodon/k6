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
// TODO: DO NOT USE AS IT'S NOT DONE
type eventLoop struct {
	queueLock     sync.Mutex
	queue         []func()
	wakeupCh      chan struct{} // maybe use sync.Cond ?
	reservedCount int
}

func newEventLoop() *eventLoop {
	return &eventLoop{
		wakeupCh: make(chan struct{}, 1),
	}
}

// RunOnLoop queues the function to be called from/on the loop
// This needs to be called before calling `Start`
// TODO maybe have only Reserve as this is equal to `e.Reserve()(f)`
func (e *eventLoop) RunOnLoop(f func()) {
	e.queueLock.Lock()
	e.queue = append(e.queue, f)
	e.queueLock.Unlock()
	select {
	case e.wakeupCh <- struct{}{}:
	default:
	}
}

// Reserve "reserves" a spot on the loop, preventing it from returning/finishing. The returning function will queue it's
// argument and wakeup the loop if needed and also unreserve the spot so that the loop can exit.
// this should be used instead of MakeHandledPromise if a promise will not be returned
// TODO better name
func (e *eventLoop) Reserve() func(func()) {
	e.queueLock.Lock()
	e.reservedCount++
	e.queueLock.Unlock()

	return func(f func()) {
		e.queueLock.Lock()
		e.queue = append(e.queue, f)
		e.reservedCount--
		e.queueLock.Unlock()
		select {
		case e.wakeupCh <- struct{}{}:
		default:
		}
	}
}

// Start will run the event loop until it's empty and there are no reserved spots
// or the context is done
func (e *eventLoop) Start(ctx context.Context) {
	done := ctx.Done()
	for {
		select { // check if done
		case <-done:
			return
		default:
		}

		// acquire the queue
		e.queueLock.Lock()
		queue := e.queue
		e.queue = make([]func(), 0, len(queue))
		reserved := e.reservedCount != 0
		e.queueLock.Unlock()

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
