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
func (p *Parser) Parse(expression string) *ExpressionField {
	value := strings.TrimSpace(expression)
	values := strings.Split(value, " ")
	if len(values) != 5 {
		panic(errors.New("invalid expression"))
	}

	minutes := p.PP(values[0], 59)
	hours := p.PP(values[1], 24)
	dayOfMonth := p.PP(values[2], 31)
	month := p.PP(values[3], 12)
	dayOfWeek := p.PP(values[4], 7)

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
		for i := min; i <= max; i += modulo {
			c := i
			if includeIndex && (i == min || i == max) {
				result = append(result, c)

			} else {
				if i == 0 {
					result = append(result, c)
				}
				result = append(result, c)
			}

		}
	}

	return result
}
