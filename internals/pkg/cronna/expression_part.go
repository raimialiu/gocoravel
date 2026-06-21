package cronna

import (
	"fmt"
	"slices"
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

func (p *ExpressionPart) CheckIfTimeIsDue(t time.Time) bool {
	return p.IsMinuteDue(t) &&
		p.IsHourDue(t) &&
		p.IsDayOfMonthDue(t) &&
		p.IsMonthDue(t) &&
		p.IsDayOfWeekDue(t)
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

func (p *ExpressionPart) IsMinuteDue(t time.Time) bool {
	return p.expressionPartIsDue(t.Minute(), p._minute.String(), 60)
}

func (p *ExpressionPart) IsHourDue(t time.Time) bool {
	return p.expressionPartIsDue(t.Minute(), p._hour.String(), 24)
}

func (p *ExpressionPart) IsDayOfMonthDue(t time.Time) bool {
	return p.expressionPartIsDue(t.Minute(), p._dayOfMonth.String(), 31)
}

func (p *ExpressionPart) IsMonthDue(t time.Time) bool {
	return p.expressionPartIsDue(t.Minute(), p._month.String(), 12)
}

func (p *ExpressionPart) IsDayOfWeekDue(t time.Time) bool {
	return p.expressionPartIsDue(t.Minute(), p._dayOfWeek.String(), 7)
}

func (P *ExpressionPart) expressionPartIsDue(time int, part string, timePart int) bool {
	if time == 0 {
		time = timePart
	}

	parser := &Parser{}
	parseResult := parser.PP(part, timePart)

	return slices.Contains(parseResult, time)
}

func (p *ExpressionPart) Minute() ExpressionDataField { return p._minute }

func (p *ExpressionPart) Hour() ExpressionDataField { return p._hour }

func (p *ExpressionPart) Month() ExpressionDataField { return p._month }

func (p *ExpressionPart) DayOfWeek() ExpressionDataField { return p._dayOfWeek }

func (p *ExpressionPart) DayOfMonth() ExpressionDataField { return p._dayOfMonth }
