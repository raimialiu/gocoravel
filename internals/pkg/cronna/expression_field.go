package cronna

import "errors"

type (
	ExpressionFieldConfig func(*ExpressionField)
	ExpressionField       struct {
		_part        ExpressionPart
		_minutes     []int
		_hours       []int
		_dayOfMonth  []int
		_monthOfYear []int
		_dayOfWeek   []int
	}
)

func NewExpressionField(parser Parser, expression string) (*ExpressionField, error) {
	if expression == "" {
		return nil, errors.New("expression is empty")
	}

	return parser.ParseV2(expression), nil
}
