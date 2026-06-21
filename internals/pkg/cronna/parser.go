package cronna

import (
	"errors"
	"strconv"
	"strings"
)

type Parser struct {
	_expressionPart ExpressionPart
	_rawExpression  string
}

// New Parse Function
func (p *Parser) ParseV2(expression string) *ExpressionField {
	value := strings.TrimSpace(expression)
	values := strings.Split(value, " ")
	if len(values) != 5 {
		panic(errors.New("invalid expression"))
	}

	minutes := p.PP(values[0], 59)
	hours := p.PP(values[0], 24)
	dayOfMonth := p.PP(values[0], 31)
	month := p.PP(values[0], 12)
	dayOfWeek := p.PP(values[0], 7)

	expressionField := &ExpressionField{
		_minutes:     minutes,
		_hours:       hours,
		_dayOfMonth:  dayOfMonth,
		_monthOfYear: month,
		_dayOfWeek:   dayOfWeek,
	}

	return expressionField
}

func (p *Parser) ParseForwardSlash(expression string, max int) []int {
	// */4, 2-30/5 -> possible format
	vl := strings.Split(expression, "/")
	f := vl[0]
	l := vl[1]

	lv, err := strconv.Atoi(l)
	if err != nil {
		panic(err)
	}

	if f == "*" { // Format: */4

		results := appendRange(0, max, lv, false)
		return results
	}

	// Format 2-30/5
	if strings.Contains(f, "-") {
		fs, sc := _hyphenValues(f)
		results := appendRange(fs, sc, lv, false)
		return results
	}

	if strings.Contains(l, "-") {
		panic(errors.New("invalid expression"))
	}

	return make([]int, 0)
}

func (p *Parser) PP(value string, max int) []int {
	// *, */4, 2-30/5, n1-n3, n,n1,n2 -> possible format
	result := make([]int, 0)
	if value == "*" { //Format: *
		result = appendRange(0, max, 0, false)
	}

	if strings.Contains(value, "/") { //format:   */4, 2-30/5
		parsedValues := p.ParseForwardSlash(value, max)
		result = append(result, parsedValues...)
	}

	if strings.Contains(value, "-") && !strings.Contains(value, "/") { // n1-n3
		f, l := _hyphenValues(value)
		values := appendRange(f, l, 0, false)
		result = append(result, values...)
	}

	if strings.Contains(value, ",") { // format n,n1,n2
		values := strings.Split(value, ",")
		for _, vl := range values {
			c, err := strconv.Atoi(vl)
			if err != nil {
				panic(err)
			}
			if c > max {
				panic(errors.New("invalid expression"))
			}
			result = append(result, c)
		}
	}

	return result
}

func (p *Parser) CheckIfSpecifiedInt(value string, toCheck int) bool {
	v, err := strconv.Atoi(value)
	if err != nil {
		panic(err)
	}

	return v == toCheck
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
	modulo int,
	includeIndex bool,
) []int {
	result := make([]int, 0)
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

	return result
}

func splitForwardSlash(value string, min, max int) []int {
	results := make([]int, max)
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
				appendResult := appendRange(0, 59, s, false)
				results = append(results, appendResult...)
			}

			if first == "*" && strings.Contains(second, "-") {
				f, l := _hyphenValues(second)
				for i := f; i <= l; i++ {
					appendResult := appendRange(0, 59, i, false)
					results = append(results, appendResult...)
				}

			}
		}
	}

	return results
}

func commonValue(values []string, min, max int) []int {
	// format -> *, */n, n, n-n, n,n1,n2,n3, */2,0-4/4, */2-4
	// n,n1,n2,n3-n4, */5
	result := make([]int, max)
	for _, value := range values {
		if strings.Contains(value, "/") { // */5
			result = append(result, splitForwardSlash(value, min, max)...)
		}

		if strings.Contains(value, "-") && !strings.Contains(value, "/") { // n3-n4
			f, l := _hyphenValues(value)
			appendResult := appendRange(f, l, 0, true)
			result = append(result, appendResult...)
		} else {
			vl, _ := strconv.Atoi(value)
			result = append(result, vl)
		}

	}

	return result
}

func _parse(part string, min, max int) ([]int, error) {
	// format -> *, */n, n, n-n, n,n1,n2,n3
	// 0 8-18 * * 1-5 */5 1-5/2
	result := make([]int, max)
	value := part

	if value == "*" { // *
		appendResult := appendRange(0, 59, 0, true)
		result = append(result, appendResult...)
	} else {
		if strings.Contains(value, "-") && !strings.Contains(value, "/") { // 1-5
			f, l := _hyphenValues(value)
			appendResult := appendRange(f, l, 0, true)
			result = append(result, appendResult...)
		} else {

			if strings.Contains(value, "/") { // */5
				result = append(result, splitForwardSlash(value, min, max)...)
			} else {
				if strings.Contains(value, ",") { //n,n1,n2,n3
					result = append(result, commonValue(strings.Split(value, ","), min, max)...)
				}
			}
		}
	}

	return result, nil
}

func _parseField(part ExpressionDataField, min, max int) ([]int, error) {
	// format -> *, */n, n, n-n, n,n1,n2,n3
	// 0 8-18 * * 1-5 */5 1-5/2
	result := make([]int, max)
	value := part.String()
	parseResult, parseError := _parse(value, min, max)
	if parseError != nil {
		return make([]int, 0), parseError
	}

	result = append(result, parseResult...)
	return result, nil
}

func withMinute(expressionPart ExpressionPart) ExpressionFieldConfig {
	return func(config *ExpressionField) {
		if minutes, _ := _parseField(expressionPart._minute, 0, 59); minutes != nil {
			config._minutes = minutes
		}
	}
}

func withHour(expressionPart ExpressionPart) ExpressionFieldConfig {
	return func(config *ExpressionField) {
		if hour, _ := _parseField(expressionPart._hour, 0, 24); hour != nil {
			config._hours = hour
		}
	}
}

func withDayOfMonth(expressionPart ExpressionPart) ExpressionFieldConfig {
	return func(config *ExpressionField) {
		if dayOfMonth, _ := _parseField(expressionPart._minute, 0, 31); dayOfMonth != nil {
			config._dayOfMonth = dayOfMonth
		}
	}
}

func withMonth(expressionPart ExpressionPart) ExpressionFieldConfig {
	return func(config *ExpressionField) {
		if month, _ := _parseField(expressionPart._month, 0, 12); month != nil {
			config._monthOfYear = month
		}
	}
}

func withDayOfTheWeek(expressionPart ExpressionPart) ExpressionFieldConfig {
	return func(config *ExpressionField) {
		if dayOfWeek, _ := _parseField(expressionPart._minute, 0, 7); dayOfWeek != nil {
			config._dayOfWeek = dayOfWeek
		}
	}
}

func (p *Parser) ParseSingle(expression string, min, max int) []int {
	partResult := make([]int, 0)
	result, _ := _parseField(NewExpressionDataField(expression), min, max)
	partResult = append(partResult, result...)

	return partResult
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

	p._rawExpression = expressionPart.ToString()

	return expressionField, nil
}
