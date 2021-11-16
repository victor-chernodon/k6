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

package stats

import (
	"fmt"

	"go.k6.io/k6/pkg/combinators"
)

// ParseAssertion parses any aggregation method as defined in
// the BNF: `aggregation_method whitespace* operator whitespace* float`.
// The Result's `Payload interface{}` value will hold the
// assertion expression as a `[]interface{}` slice of len 3, its content
// will hold values of type `string` at position 0, `string` at position 1,
// and `float64` at position 2.
func ParseAssertion() combinators.Parser {
	parser := combinators.Sequence(
		ParseAggregationMethod(),
		combinators.DiscardAll(combinators.Whitespace()),
		ParseOperator(),
		combinators.DiscardAll(combinators.Whitespace()),
		ParseValue(),
		combinators.DiscardAll(combinators.Newline()),
	)

	return func(input []rune) combinators.Result {
		res := parser(input)
		if res.Err != nil {
			return res
		}

		return combinators.Success(res.Payload, res.Remaining)
	}
}

// ParseAggregationMethod parses any aggregation method as defined in
// the BNF: `aggregation_method -> trend | rate | gauge | counter`.
// The Result's `Payload interface{}` value will hold the
// aggregation method name as a string.
func ParseAggregationMethod() combinators.Parser {
	parser := combinators.Expect(combinators.Alternative(
		ParseCounter(),
		ParseGauge(),
		ParseRate(),
		ParseTrend(),
		ParsePercentile(),
	), "aggregation method")

	return func(input []rune) combinators.Result {
		res := parser(input)
		if res.Err != nil {
			return res
		}

		return combinators.Success(res.Payload, res.Remaining)
	}
}

// ParseOperator parses a threshold expression supported operator
// as defined in the BNF: `operator -> ">" | ">=" | "<=" | "<" | "==" | "===" | "!="`.
// The Result's `Payload interface{}` value will hold the
// operator expression as a string.
func ParseOperator() combinators.Parser {
	parser := combinators.Expect(combinators.Alternative(
		combinators.Tag(">="),
		combinators.Tag("<="),
		combinators.Tag(">"),
		combinators.Tag("<"),
		combinators.Tag("==="),
		combinators.Tag("=="),
		combinators.Tag("!="),
	), "operator")

	return func(input []rune) combinators.Result {
		res := parser(input)
		if res.Err != nil {
			return res
		}

		return combinators.Success(res.Payload.(string), res.Remaining)
	}
}

// ParseTrend parses a trend aggregation method as defined in
// the BNF: `trend -> "avg" | "min" | "max" | "med" | percentile`.
// The Result's `Payload interface{}` value will hold the
// trend's aggregation method name as a string.
func ParseTrend() combinators.Parser {
	parser := combinators.Expect(combinators.Alternative(
		combinators.Tag("mean"),
		combinators.Tag("min"),
		combinators.Tag("max"),
		combinators.Tag("avg"),
		combinators.Tag("med"),
		ParsePercentile(),
	), "trend aggregation method")

	return func(input []rune) combinators.Result {
		res := parser(input)
		if res.Err != nil {
			return res
		}

		return combinators.Success(res.Payload, res.Remaining)
	}
}

// ParsePercentile parses a percentile as defined in
// the BNF: `percentile -> "p(" float ")"`. The Result's `Payload interface{}`
// value will hold the percentile expression as a string.
func ParsePercentile() combinators.Parser {
	parser := combinators.Expect(combinators.Sequence(
		combinators.Tag("p("),
		combinators.Float(),
		combinators.Char(')'),
	), "percentile")

	return func(input []rune) combinators.Result {
		res := parser(input)
		if res.Err != nil {
			return res
		}

		parsed, ok := res.Payload.([]interface{})
		if !ok {
			err := fmt.Errorf("failed parsing percentile expression; " +
				"reason: converting ParsePercentile() parser result's payload to []interface{} failed." +
				"it looks like you've found a bug, we'd be grateful if you would consider " +
				"opening an issue in the K6 repository (https://github.com/grafana/k6/issues/new)",
			)
			res.Err = combinators.NewFatalError(input, err, "percentile")
		}

		percentile := fmt.Sprintf("%s%g%s", parsed[0].(string), parsed[1].(float64), parsed[2].(string))

		return combinators.Success(percentile, res.Remaining)
	}
}

// ParseRate parses a rate aggregation method as defined in
// the BNF: `rate -> "rate"`. The Result's `Payload interface{}` value
// will hold the rate's aggregation method name as a string.
func ParseRate() combinators.Parser {
	parser := combinators.Expect(combinators.Tag("rate"), "rate aggregation method")

	return func(input []rune) combinators.Result {
		res := parser(input)
		if res.Err != nil {
			return res
		}

		return combinators.Success(res.Payload, res.Remaining)
	}
}

// ParseGauge parses a gauge aggregation method as defined in
// the BNF: `gauge -> "value"`. The Result's `Payload interface{}` value
// will hold the gauge's aggregation method name as a string.
func ParseGauge() combinators.Parser {
	parser := combinators.Expect(combinators.Alternative(
		combinators.Tag("value"),
	), "gauge aggregation method")

	return func(input []rune) combinators.Result {
		res := parser(input)
		if res.Err != nil {
			return res
		}

		return combinators.Success(res.Payload, res.Remaining)
	}
}

// ParseCounter parses a counter aggregation method as defined in
// the BNF: `counter -> "count" | "rate"`. The Result's `Payload interface{}` value
// will hold the counter's aggregation method name as a string.
func ParseCounter() combinators.Parser {
	parser := combinators.Expect(combinators.Alternative(
		combinators.Tag("count"),
		combinators.Tag("rate"),
	), "counter aggregation method")

	return func(input []rune) combinators.Result {
		res := parser(input)
		if res.Err != nil {
			return res
		}

		return combinators.Success(res.Payload, res.Remaining)
	}
}

// ParseValue parses a threshold assertion value as defined in
// the BNF:
// ```
// float -> digit+ (. digit+)?
// digit -> "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9"
// ```
//
// The Result's `Payload interface{}` value will hold the assertion's right
// hand side's as a float64.
func ParseValue() combinators.Parser {
	parser := combinators.Expect(combinators.Float(), "numerical value")

	return func(input []rune) combinators.Result {
		res := parser(input)
		if res.Err != nil {
			return res
		}

		return combinators.Success(res.Payload, res.Remaining)
	}
}
