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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAssertion(t *testing.T) {
	t.Parallel()

	// Arrange
	parser := ParseAssertion()

	// Act
	trendAssertionResult := parser([]rune("p(99.9)<300"))
	rateAssertionResult := parser([]rune("rate>0.95"))
	gaugeAssertionResult := parser([]rune("value<4000"))
	counterAssertionResult := parser([]rune("count<100"))
	controlCharactersResult := parser([]rune("med  \t<\t\t  200\r\n"))

	// Assert
	assert.Equal(t, []interface{}{string("p(99.9)"), "<", float64(300)}, trendAssertionResult.Payload)
	assert.Equal(t, "", string(trendAssertionResult.Remaining))
	assert.Equal(t, []interface{}{"rate", ">", float64(0.95)}, rateAssertionResult.Payload)
	assert.Equal(t, "", string(rateAssertionResult.Remaining))
	assert.Equal(t, []interface{}{"value", "<", float64(4000)}, gaugeAssertionResult.Payload)
	assert.Equal(t, "", string(gaugeAssertionResult.Remaining))
	assert.Equal(t, []interface{}{"count", "<", float64(100)}, counterAssertionResult.Payload)
	assert.Equal(t, "", string(counterAssertionResult.Remaining))
	assert.Nil(t, controlCharactersResult.Err)
	assert.Equal(t, []interface{}{"med", "<", float64(200)}, controlCharactersResult.Payload)
	assert.Equal(t, "", string(controlCharactersResult.Remaining))
}

func TestParseOperator(t *testing.T) {
	t.Parallel()

	// Arrange
	parser := ParseOperator()

	// Act
	greaterOrEqualThan := parser([]rune(">="))
	lowerOrEqualThan := parser([]rune("<="))
	greaterThan := parser([]rune(">"))
	lowerThan := parser([]rune("<"))
	strictlyEqual := parser([]rune("==="))
	looselyEqual := parser([]rune("=="))
	notEqual := parser([]rune("!="))

	// Assert
	assert.Equal(t, ">=", greaterOrEqualThan.Payload)
	assert.Equal(t, "", string(greaterOrEqualThan.Remaining))
	assert.Equal(t, "<=", lowerOrEqualThan.Payload)
	assert.Equal(t, "", string(lowerOrEqualThan.Remaining))
	assert.Equal(t, ">", greaterThan.Payload)
	assert.Equal(t, "", string(greaterThan.Remaining))
	assert.Equal(t, "<", lowerThan.Payload)
	assert.Equal(t, "", string(lowerThan.Remaining))
	assert.Equal(t, "===", strictlyEqual.Payload)
	assert.Equal(t, "", string(strictlyEqual.Remaining))
	assert.Equal(t, "==", looselyEqual.Payload)
	assert.Equal(t, "", string(looselyEqual.Remaining))
	assert.Equal(t, "!=", notEqual.Payload)
	assert.Equal(t, "", string(notEqual.Remaining))
}

func TestParseTrend(t *testing.T) {
	t.Parallel()

	// Arrange
	parser := ParseTrend()

	// Act
	avg := parser([]rune("avg"))
	min := parser([]rune("min"))
	max := parser([]rune("max"))
	med := parser([]rune("med"))
	p99 := parser([]rune("p(99.9)"))

	// Assert
	assert.Equal(t, "avg", avg.Payload)
	assert.Equal(t, "", string(avg.Remaining))
	assert.Equal(t, "min", min.Payload)
	assert.Equal(t, "", string(min.Remaining))
	assert.Equal(t, "max", max.Payload)
	assert.Equal(t, "", string(max.Remaining))
	assert.Equal(t, "med", med.Payload)
	assert.Equal(t, "", string(med.Remaining))
	assert.Equal(t, string("p(99.9)"), p99.Payload)
	assert.Equal(t, "", string(p99.Remaining))
}

func TestParsePercentile_ValueWithoutDecimal(t *testing.T) {
	t.Parallel()

	// Arrange
	parser := ParsePercentile()

	// Act
	result := parser([]rune("p(99)"))

	// Assert
	assert.Equal(t, string("p(99)"), result.Payload)
	assert.Equal(t, "", string(result.Remaining))
}

func TestParsePercentile_ValueWithSingleDecimal(t *testing.T) {
	t.Parallel()

	// Arrange
	parser := ParsePercentile()

	// Act
	result := parser([]rune("p(99.9)"))

	// Assert
	assert.Equal(t, string("p(99.9)"), result.Payload)
	assert.Equal(t, "", string(result.Remaining))
}

func TestParsePercentile_ValueWithDoubleDecimal(t *testing.T) {
	t.Parallel()

	// Arrange
	parser := ParsePercentile()

	// Act
	result := parser([]rune("p(99.99)"))

	// Assert
	assert.Equal(t, string("p(99.99)"), result.Payload)
	assert.Equal(t, "", string(result.Remaining))
}

func TestParseRate(t *testing.T) {
	t.Parallel()

	// Arrange
	parser := ParseRate()

	// Act
	result := parser([]rune("rate"))

	// Assert
	assert.Equal(t, "rate", result.Payload)
	assert.Equal(t, "", string(result.Remaining))
}

func TestParseGauge(t *testing.T) {
	t.Parallel()

	// Arrange
	parser := ParseGauge()

	// Act
	value := parser([]rune("value"))

	// Assert
	assert.Equal(t, "value", value.Payload)
	assert.Equal(t, "", string(value.Remaining))
}

func TestParseCounter(t *testing.T) {
	t.Parallel()

	// Arrange
	parser := ParseCounter()

	// Act
	count := parser([]rune("count"))
	rate := parser([]rune("rate"))

	// Assert
	assert.Equal(t, "count", count.Payload)
	assert.Equal(t, "", string(count.Remaining))
	assert.Equal(t, "rate", rate.Payload)
	assert.Equal(t, "", string(rate.Remaining))
}
