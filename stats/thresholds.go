/*
 *
 * k6 - a next-generation load testing tool
 * Copyright (C) 2016 Load Impact
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
package stats

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"go.k6.io/k6/lib/types"
)

// Threshold is a representation of a single threshold for a single metric
type Threshold struct {
	// Source is the text based source of the threshold
	Source string
	// LastFailed is a marker if the last testing of this threshold failed
	LastFailed bool
	// AbortOnFail marks if a given threshold fails that the whole test should be aborted
	AbortOnFail bool
	// AbortGracePeriod is a the minimum amount of time a test should be running before a failing
	// this threshold will abort the test
	AbortGracePeriod types.NullDuration
	// parsed is the threshold condition parsed from the Source expression
	parsed *thresholdCondition
}

func newThreshold(src string, abortOnFail bool, gracePeriod types.NullDuration) (*Threshold, error) {
	condition, err := parseThresholdCondition(src)
	if err != nil {
		return nil, err
	}

	return &Threshold{
		Source:           src,
		parsed:           condition,
		AbortOnFail:      abortOnFail,
		AbortGracePeriod: gracePeriod,
	}, nil
}

func (t *Threshold) runNoTaint(sinks map[string]float64) (bool, error) {
	// Extract the sink value for the aggregation method used in the threshold
	// expression
	lhs, ok := sinks[t.parsed.AggregationMethod]
	if !ok {
		return false, fmt.Errorf("unable to apply threshold %s over metrics; reason: "+
			"no metric supporting the %s aggregation method found",
			t.parsed.AggregationMethod,
			t.parsed.AggregationMethod)
	}

	// Apply the threshold expression operator to the left and
	// right hand side values
	var passes bool
	switch t.parsed.Operator {
	case ">":
		passes = lhs > t.parsed.Value
	case ">=":
		passes = lhs >= t.parsed.Value
	case "<=":
		passes = lhs <= t.parsed.Value
	case "<":
		passes = lhs < t.parsed.Value
	case "==":
		passes = lhs == t.parsed.Value
	case "===":
		// Considering a sink always maps to float64 values,
		// strictly equal is equivalent to loosely equal
		passes = lhs == t.parsed.Value
	case "!=":
		passes = lhs != t.parsed.Value
	default:
		// The ParseThresholdCondition constructor should ensure that no invalid
		// operator gets through, but let's protect our future selves anyhow.
		return false, fmt.Errorf("unable to apply threshold %s over metrics; "+
			"reason: %s is an invalid operator",
			t.Source,
			t.parsed.Operator,
		)
	}

	// Perform the actual threshold verification
	return passes, nil
}

func (t *Threshold) run(sinks map[string]float64) (bool, error) {
	passes, err := t.runNoTaint(sinks)
	t.LastFailed = !passes
	return passes, err
}

type thresholdCondition struct {
	AggregationMethod string
	Operator          string
	Value             float64
}

// ParseThresholdCondition parses a threshold condition expression,
// as defined in a JS script (for instance p(95)<1000), into a ThresholdCondition
// instance, using our parser combinators package.

// This parser expect a threshold expression matching the following BNF
//
// ```
// assertion           -> aggregation_method whitespace* operator whitespace* float newline*
// aggregation_method  -> trend | rate | gauge | counter
// counter             -> "count" | "sum" | "rate"
// gauge               -> "last" | "min" | "max" | "value"
// rate                -> "rate"
// trend               -> "min" | "mean" | "avg" | "max" | percentile
// percentile          -> "p(" float ")"
// operator            -> ">" | ">=" | "<=" | "<" | "==" | "===" | "!="
// float               -> digit+ (. digit+)?
// digit               -> "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9"
// whitespace          -> space | tab
// newline						 -> linefeed | crlf
// crlf 							 -> carriage_return linefeed
// linefeed						 -> "\n"
// carriage_return		 -> "\r"
// tab                 -> "\t"
// space               -> " "
// ```
func parseThresholdCondition(expression string) (*thresholdCondition, error) {
	parser := ParseAssertion()

	// Parse the Threshold as provided in the JS script options thresholds value (p(95)<1000)
	result := parser([]rune(expression))
	if result.Err != nil {
		return nil, fmt.Errorf("parsing threshold condition %s failed; "+
			"reason: the parser failed on %s",
			expression,
			result.Err.ErrorAtChar([]rune(expression)))
	}

	// The Sequence combinator will return a slice of interface{}
	// instances. Up to us to decide what we want to cast them down
	// to.
	// Considering our expression format, the parser should return a slice
	// of size 3 to us: aggregation_method operator sink_value. The type system
	// ensures us it should be the case too, but let's protect our future selves anyhow.
	var ok bool
	parsed, ok := result.Payload.([]interface{})
	if !ok {
		return nil, fmt.Errorf("parsing threshold condition %s failed"+
			"; reason: unable to cast parsed expression to []interface{}"+
			"it looks like you've found a bug, we'd be grateful if you would consider "+
			"opening an issue in the K6 repository (https://github.com/grafana/k6/issues/new)",
			expression,
		)
	} else if len(parsed) != 3 {
		return nil, fmt.Errorf("parsing threshold condition %s failed"+
			"; reason: parsed %d expression tokens, expected 3 (aggregation_method operator value, as in rate<100)"+
			"it looks like you've found a bug, we'd be grateful if you would consider "+
			"opening an issue in the K6 repository (https://github.com/grafana/k6/issues/new)",
			expression,
			len(parsed),
		)
	}

	// Unpack the various components of the parsed threshold expression
	method, ok := parsed[0].(string)
	if !ok {
		return nil, fmt.Errorf("the threshold expression parser failed; " +
			"reason: unable to cast parsed aggregation method to string" +
			"it looks like you've found a bug, we'd be grateful if you would consider " +
			"opening an issue in the K6 repository (https://github.com/grafana/k6/issues/new)",
		)
	}
	operator, ok := parsed[1].(string)
	if !ok {
		return nil, fmt.Errorf("the threshold expression parser failed; " +
			"reason: unable to cast parsed operator to string" +
			"it looks like you've found a bug, we'd be grateful if you would consider " +
			"opening an issue in the K6 repository (https://github.com/grafana/k6/issues/new)",
		)
	}

	value, ok := parsed[2].(float64)
	if !ok {
		return nil, fmt.Errorf("the threshold expression parser failed; " +
			"reason: unable to cast parsed value to underlying type (float64)" +
			"it looks like you've found a bug, we'd be grateful if you would consider " +
			"opening an issue in the K6 repository (https://github.com/grafana/k6/issues/new)",
		)
	}

	return &thresholdCondition{AggregationMethod: method, Operator: operator, Value: value}, nil
}

type thresholdConfig struct {
	Threshold        string             `json:"threshold"`
	AbortOnFail      bool               `json:"abortOnFail"`
	AbortGracePeriod types.NullDuration `json:"delayAbortEval"`
}

// used internally for JSON marshalling
type rawThresholdConfig thresholdConfig

func (tc *thresholdConfig) UnmarshalJSON(data []byte) error {
	// shortcircuit unmarshalling for simple string format
	if err := json.Unmarshal(data, &tc.Threshold); err == nil {
		return nil
	}

	rawConfig := (*rawThresholdConfig)(tc)
	return json.Unmarshal(data, rawConfig)
}

func (tc thresholdConfig) MarshalJSON() ([]byte, error) {
	var data interface{} = tc.Threshold
	if tc.AbortOnFail {
		data = rawThresholdConfig(tc)
	}

	return MarshalJSONWithoutHTMLEscape(data)
}

// Thresholds is the combination of all Thresholds for a given metric
type Thresholds struct {
	Thresholds []*Threshold
	Abort      bool
	sinked     map[string]float64
}

// NewThresholds returns Thresholds objects representing the provided source strings
func NewThresholds(sources []string) (Thresholds, error) {
	tcs := make([]thresholdConfig, len(sources))
	for i, source := range sources {
		tcs[i].Threshold = source
	}

	return newThresholdsWithConfig(tcs)
}

func newThresholdsWithConfig(configs []thresholdConfig) (Thresholds, error) {
	thresholds := make([]*Threshold, len(configs))
	sinked := make(map[string]float64)

	for i, config := range configs {
		t, err := newThreshold(config.Threshold, config.AbortOnFail, config.AbortGracePeriod)
		if err != nil {
			return Thresholds{}, fmt.Errorf("threshold %d error: %w", i, err)
		}
		thresholds[i] = t
	}

	return Thresholds{thresholds, false, sinked}, nil
}

func (ts *Thresholds) runAll(duration time.Duration) (bool, error) {
	succeeded := true
	for i, threshold := range ts.Thresholds {
		b, err := threshold.run(ts.sinked)
		if err != nil {
			return false, fmt.Errorf("threshold %d run error: %w", i, err)
		}

		if !b {
			succeeded = false

			if ts.Abort || !threshold.AbortOnFail {
				continue
			}

			ts.Abort = !threshold.AbortGracePeriod.Valid ||
				threshold.AbortGracePeriod.Duration < types.Duration(duration)
		}
	}

	return succeeded, nil
}

// Run processes all the thresholds with the provided Sink at the provided time and returns if any
// of them fails
func (ts *Thresholds) Run(sink Sink, duration time.Duration) (bool, error) {
	// Update the sinks store
	ts.sinked = make(map[string]float64)
	f := sink.Format(duration)
	for k, v := range f {
		ts.sinked[k] = v
	}

	return ts.runAll(duration)
}

// UnmarshalJSON is implementation of json.Unmarshaler
func (ts *Thresholds) UnmarshalJSON(data []byte) error {
	var configs []thresholdConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return err
	}
	newts, err := newThresholdsWithConfig(configs)
	if err != nil {
		return err
	}
	*ts = newts
	return nil
}

// MarshalJSON is implementation of json.Marshaler
func (ts Thresholds) MarshalJSON() ([]byte, error) {
	configs := make([]thresholdConfig, len(ts.Thresholds))
	for i, t := range ts.Thresholds {
		configs[i].Threshold = t.Source
		configs[i].AbortOnFail = t.AbortOnFail
		configs[i].AbortGracePeriod = t.AbortGracePeriod
	}

	return MarshalJSONWithoutHTMLEscape(configs)
}

// MarshalJSONWithoutHTMLEscape marshals t to JSON without escaping characters
// for safe use in HTML.
func MarshalJSONWithoutHTMLEscape(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)
	bytes := buffer.Bytes()
	if err == nil && len(bytes) > 0 {
		// Remove the newline appended by Encode() :-/
		// See https://github.com/golang/go/issues/37083
		bytes = bytes[:len(bytes)-1]
	}
	return bytes, err
}

var _ json.Unmarshaler = &Thresholds{}
var _ json.Marshaler = &Thresholds{}
