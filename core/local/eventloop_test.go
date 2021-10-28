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

package local

import (
	"io/ioutil"
	"net/url"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.k6.io/k6/js"
	"go.k6.io/k6/lib"
	"go.k6.io/k6/lib/metrics"
	"go.k6.io/k6/lib/testutils"
	"go.k6.io/k6/lib/types"
	"go.k6.io/k6/loader"
)

func TestEventLoop(t *testing.T) {
	t.Parallel()
	script := []byte(`
		setTimeout(()=> {console.log("initcontext setTimeout")}, 200)
		console.log("initcontext");
		export default function() {
			setTimeout(()=> {console.log("default setTimeout")}, 200)
			console.log("default");
		};
		export function setup() {
			setTimeout(()=> {console.log("setup setTimeout")}, 200)
			console.log("setup");
		};
		export function teardown() {
			setTimeout(()=> {console.log("teardown setTimeout")}, 200)
			console.log("teardown");
		};
		export function handleSummary() {
			setTimeout(()=> {console.log("handleSummary setTimeout")}, 200)
			console.log("handleSummary");
		};
`)

	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)
	logHook := testutils.SimpleLogrusHook{HookedLevels: []logrus.Level{logrus.InfoLevel}}
	logger.AddHook(&logHook)

	registry := metrics.NewRegistry()
	builtinMetrics := metrics.RegisterBuiltinMetrics(registry)
	runner, err := js.New(
		logger,
		&loader.SourceData{
			URL:  &url.URL{Path: "/script.js"},
			Data: script,
		},
		nil,
		lib.RuntimeOptions{},
		builtinMetrics,
		registry,
	)
	require.NoError(t, err)

	ctx, cancel, execScheduler, samples := newTestExecutionScheduler(t, runner, logger,
		lib.Options{
			TeardownTimeout: types.NullDurationFrom(time.Second),
			SetupTimeout:    types.NullDurationFrom(time.Second),
		})
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- execScheduler.Run(ctx, ctx, samples, builtinMetrics) }()

	select {
	case err := <-errCh:
		require.NoError(t, err)
		_, err = runner.HandleSummary(ctx, &lib.Summary{RootGroup: &lib.Group{}})
		require.NoError(t, err)
		entries := logHook.Drain()
		msgs := make([]string, len(entries))
		for i, entry := range entries {
			msgs[i] = entry.Message
		}
		require.Equal(t, []string{
			"initcontext", // first initialization
			"initcontext setTimeout",
			"initcontext", // for vu
			"initcontext setTimeout",
			"initcontext", // for setup
			"initcontext setTimeout",
			"setup", // setup
			"setup setTimeout",
			"default", // one iteration
			"default setTimeout",
			"initcontext", // for teardown
			"initcontext setTimeout",
			"teardown", // teardown
			"teardown setTimeout",
			"initcontext", // for handleSummary
			"initcontext setTimeout",
			"handleSummary", // handleSummary
			"handleSummary setTimeout",
		}, msgs)
	case <-time.After(10 * time.Second):
		t.Fatal("timed out")
	}
}
