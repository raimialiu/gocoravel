package cronna

import (
	"errors"
	"strconv"
	"strings"
)

type Parser struct {
	_expressionPart ExpressionPart
}

func _hyphenValues(value string) (int, int) {
	values := strings.Split(value, "-")
	first, _ := strconv.Atoi(values[0])
	last, _ := strconv.Atoi(values[1])

	if first > last {
		first, last = last, first
	}

	return first, last
}

func appendRange(
	min, max int,
	result []int,
	modulo int,
	includeIndex bool,
) {
	if modulo == 0 {
		for i := min; i <= max; i++ {
			result = append(result, i)
		}
	} else {
		for i := min; i <= max; i++ {
			c := i
			if includeIndex {
				if c%modulo == 0 || i == min || i == max {
					result = append(result, c)
				}
			} else {
				if c%modulo == 0 || i == 0 {
					result = append(result, c)
				}
			}

		}
	}

}

func splitForwardSlash(value string) []int {
	results := make([]int, 0)
	if strings.Contains(value, "/") {
		// possible format, */5, 0-30/2, */2-5, 2-30/4-5
		values := strings.Split(value, "/")
		first := values[0]
		second := values[1]
		s, _ := strconv.Atoi(second)

		if strings.Contains(first, "-") && !strings.Contains(second, "-") {
			// lets say first is 0(min) - 30(max), second is just n say 5
			f, l := _hyphenValues(first)
			for i := f; i <= l; i++ {
				c := i
				if c%s == 0 || i == f || i == l {
					results = append(results, c)
				}
			}
		} else {
			if first == "*" && !strings.Contains(second, "-") {
				appendRange(0, 59, results, s, false)
			}

			if first == "*" && strings.Contains(second, "-") {
				f, l := _hyphenValues(second)
				for i := f; i <= l; i++ {
					appendRange(0, 59, results, i, false)
				}

			}
		}
	}

	return results
}

func commonValue(values []string) []int {
	// format -> *, */n, n, n-n, n,n1,n2,n3, */2,0-4/4, */2-4
	// n,n1,n2,n3-n4, */5
	result := make([]int, 0)
	for _, value := range values {
		if strings.Contains(value, "/") { // */5
			result = append(result, splitForwardSlash(value)...)
		}

		if strings.Contains(value, "-") && !strings.Contains(value, "/") { // n3-n4
			f, l := _hyphenValues(value)
			appendRange(f, l, result, 0, true)
		} else {
			vl, _ := strconv.Atoi(value)
			result = append(result, vl)
		}

	}

	return result
}

func parseField(part ExpressionDataField) ([]int, error) {
	// format -> *, */n, n, n-n, n,n1,n2,n3
	// 0 8-18 * * 1-5 */5 1-5/2
	result := make([]int, 0)
	value := part.String()
	if value == "*" { // *
		appendRange(0, 59, result, 0, true)
	} else {
		if strings.Contains(value, "-") && !strings.Contains(value, "/") { // 1-5
			f, l := _hyphenValues(value)
			appendRange(f, l, result, 0, true)
		} else {

			if strings.Contains(value, "/") { // */5
				result = append(result, splitForwardSlash(value)...)
			} else {
				if strings.Contains(value, ",") { //n,n1,n2,n3
					result = append(result, commonValue(strings.Split(value, ","))...)
				}
			}
		}
	}

	return result, nil
}

func withMinute(expressionPart ExpressionPart) ExpressionFieldConfig {
	return func(config *ExpressionField) {
		if minutes, _ := parseField(expressionPart._minute); minutes != nil {
			config._minutes = minutes
		}
	}
}

func withHour(expressionPart ExpressionPart) ExpressionFieldConfig {
	return func(config *ExpressionField) {
		if hour, _ := parseField(expressionPart._hour); hour != nil {
			config._hours = hour
		}
	}
}

func withDayOfMonth(expressionPart ExpressionPart) ExpressionFieldConfig {
	return func(config *ExpressionField) {
		if dayOfMonth, _ := parseField(expressionPart._minute); dayOfMonth != nil {
			config._dayOfMonth = dayOfMonth
		}
	}
}

func withMonth(expressionPart ExpressionPart) ExpressionFieldConfig {
	return func(config *ExpressionField) {
		if month, _ := parseField(expressionPart._month); month != nil {
			config._monthOfYear = month
		}
	}
}

func withDayOfTheWeek(expressionPart ExpressionPart) ExpressionFieldConfig {
	return func(config *ExpressionField) {
		if dayOfWeek, _ := parseField(expressionPart._minute); dayOfWeek != nil {
			config._dayOfWeek = dayOfWeek
		}
	}
}

func (p *Parser) Parse(expression string) (*ExpressionField, error) {
	expressionField := &ExpressionField{}
	expression = strings.TrimSpace(expression)
	values := strings.Split(expression, " ")
	if len(values) != 5 {
		return nil, errors.New("invalid expression")
	}

	expressionPart := NewExpressionPart(values)

	configs := make([]ExpressionFieldConfig, 0)
	configs = append(configs, withMinute(*expressionPart))
	configs = append(configs, withHour(*expressionPart))
	configs = append(configs, withDayOfMonth(*expressionPart))
	configs = append(configs, withMonth(*expressionPart))
	configs = append(configs, withDayOfTheWeek(*expressionPart))

	for _, config := range configs {
		config(expressionField)
	}

	return expressionField, nil
}
