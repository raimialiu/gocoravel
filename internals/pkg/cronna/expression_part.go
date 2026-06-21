package cronna

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/raimialiu/gostream/stream"
)

type ExpressionPart struct {
	_minute     ExpressionDataField
	_hour       ExpressionDataField
	_month      ExpressionDataField
	_dayOfMonth ExpressionDataField
	_dayOfWeek  ExpressionDataField
}

type ExpressionPartConfig func(part *ExpressionPart)

func _withMinute(minute ExpressionDataField) ExpressionPartConfig {
	return func(part *ExpressionPart) {
		part._minute = minute
	}
}

func (p *Parser) CheckIfTimeIsDue(part string, t time.Time) bool {
	isRange := strings.Contains(part, "-")
	isDelineatedArray := strings.Contains(part, ",")
	isDivisibleRange := strings.Contains(part, "/")

	if isRange && isDelineatedArray {
		panic(errors.New(fmt.Sprintf("Cron expression %s has mixed entry type.", part)))
	}

	if isDivisibleRange {

	}
}

func (p *Parser) _checkDivisibleRange(part string, toCheck int, parser Parser) bool {
	parser.
}

func (p *Parser) CheckIfSpecificSectionIsDue(s string, t int) bool {

}

func (p *ExpressionPart) ToString() string {
	value := fmt.Sprintf("%s %s %s %s %s", p._minute.String(), p._hour.String(), p._dayOfMonth.String(), p._month.String(), p._dayOfWeek.String())

	return value
}

func _withHour(hour ExpressionDataField) ExpressionPartConfig {
	return func(part *ExpressionPart) {
		part._hour = hour
	}
}

func _withMonth(month ExpressionDataField) ExpressionPartConfig {
	return func(part *ExpressionPart) {
		part._month = month
	}
}

func _withDayOfMonth(dayOfMonth ExpressionDataField) ExpressionPartConfig {
	return func(part *ExpressionPart) {
		part._dayOfMonth = dayOfMonth
	}
}

func _withDayOfWeek(dayOfWeek ExpressionDataField) ExpressionPartConfig {
	return func(part *ExpressionPart) {
		part._dayOfWeek = dayOfWeek
	}
}

func NewExpressionPart(partString []string, parts ...ExpressionPartConfig) *ExpressionPart {
	part := &ExpressionPart{}
	if parts == nil {
		parts = make([]ExpressionPartConfig, len(partString))
	}

	parts = append(parts, _withMinute(NewExpressionDataField(partString[0])))
	parts = append(parts, _withHour(NewExpressionDataField(partString[1])))
	parts = append(parts, _withDayOfMonth(NewExpressionDataField(partString[2])))
	parts = append(parts, _withMonth(NewExpressionDataField(partString[3])))
	parts = append(parts, _withDayOfWeek(NewExpressionDataField(partString[4])))

	parts = stream.From(parts).Filter(func(config ExpressionPartConfig) bool { return config != nil }).ToList()

	if len(parts) > 0 {
		for _, partConfig := range parts {
			partConfig(part)
		}
	}

	return part
}

func (p *ExpressionPart) IsDue(t time.Time, timePart int) bool {
	return p.expressionPartIsDue(t.Minute(), p._minute.String(), timePart)
}

func (P *ExpressionPart) expressionPartIsDue(time int, part string, timePart int) bool {
	if part == "" || part == " " {
		return false
	}
	if part == "*" {
		return true
	}

	isDivisibleUnit := strings.Index(part, "*/") // 0/2
	if isDivisibleUnit > -1 {
		divisor, err := strconv.Atoi(part[0:2])
		if err != nil {
			panic(err)
		}

		if divisor == 0 {
			panic(errors.New(fmt.Sprintf("Cron entry %s is attempting division by zero.", part)))
		}

		if time == 0 {
			time = timePart
		}

		return time%divisor == 0
	} else {
		parser := &Parser{}
		parseResult := parser.ParseSingle(part, 0, timePart)

	}
}

func (p *ExpressionPart) Minute() ExpressionDataField { return p._minute }

func (p *ExpressionPart) Hour() ExpressionDataField { return p._hour }

func (p *ExpressionPart) Month() ExpressionDataField { return p._month }

func (p *ExpressionPart) DayOfWeek() ExpressionDataField { return p._dayOfWeek }

func (p *ExpressionPart) DayOfMonth() ExpressionDataField { return p._dayOfMonth }
