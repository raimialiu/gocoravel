package cronna

import (
	"time"
)

// *->m * ->h *->day * -> month * -> day of the week
type (
	ExpressionConfig func(*ExpressionField)
	Expression       struct {
		_field          *ExpressionField
		_rawExpression  string
		_nextEvaluation *time.Time
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

func (e *Expression) NextTime(from time.Time) *time.Time {
	next := from.Truncate(time.Minute).Add(time.Minute)
	return &next
}

func (e *Expression) Matches(t time.Time) bool {
	return e._field._part.CheckIfTimeIsDue(t)
}

func (e *Expression) IsMinuteDue(t time.Time) bool {
	return e._field._part.IsMinuteDue(t)
}

func (e *Expression) IsHourDue(t time.Time) bool {
	return e._field._part.IsHourDue(t)
}

func (e *Expression) IsDayOfMonthDue(t time.Time) bool {
	return e._field._part.IsDayOfMonthDue(t)
}

func (e *Expression) IsMonthDue(t time.Time) bool {
	return e._field._part.IsMonthDue(t)
}

func (e *Expression) IsDayOfWeekDue(t time.Time) bool {
	return e._field._part.IsDayOfWeekDue(t)
}
