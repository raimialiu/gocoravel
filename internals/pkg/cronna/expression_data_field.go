package cronna

type (
	ExpressionDataField string
)

func NewExpressionDataField(value string) ExpressionDataField {
	return ExpressionDataField(value)
}

func (expressionDataField ExpressionDataField) String() string {
	return string(expressionDataField)
}
