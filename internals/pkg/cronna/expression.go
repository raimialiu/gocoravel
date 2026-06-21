package cronna

import (
	"slices"
	"time"
)

// *->m * ->h *->day * -> month * -> day of the week
type (
	ExpressionConfig func(*ExpressionField)
	Expression       struct {
		_field         *ExpressionField
		_rawExpression string
	}
)

func NewExpression(rawExpression string) *Expression {
	expression := &Expression{}
	parser := &Parser{}
	expressionField, err := NewExpressionField(*parser, rawExpression)
	if err != nil {
		panic(err)
	}

	expression._field = expressionField
	expression._rawExpression = rawExpression

	return expression
}

func (e *Expression) Matches(time time.Time) bool {
	dueSec := e._IsMinuteDue(time) &&
		e._IsHourDue(time) &&
		e._IsDayOfMonthDue(time) &&
		e._IsMonthDue(time) &&
		e._IsDayOfWeekDue(time)
}

func (e *Expression) _IsMinuteDue(t time.Time) bool {
	value := e._field._part.Minute().String()
	if value == "*" {
		return true
	}

	return slices.Contains(e._field._minutes, t.Minute())
}

func (e *Expression) _IsHourDue(t time.Time) bool {
	value := e._field._part.Hour().String()
	if value == "*" {
		return true
	}

	return slices.Contains(e._field._hours, t.Hour())
}

func (e *Expression) _IsDayOfMonthDue(t time.Time) bool {
	value := e._field._part.DayOfMonth().String()
	if value == "*" {
		return true
	}

	return slices.Contains(e._field._dayOfMonth, t.Day())
}

func (e *Expression) _IsMonthDue(t time.Time) bool {
	value := e._field._part.Month().String()
	if value == "*" {
		return true
	}

	return slices.Contains(e._field._monthOfYear, int(t.Month()))
}

func (e *Expression) _IsDayOfWeekDue(t time.Time) bool {
	value := e._field._part.DayOfWeek().String()
	if value == "*" {
		return true
	}

	return slices.Contains(e._field._dayOfWeek, int(t.Weekday()))
}
