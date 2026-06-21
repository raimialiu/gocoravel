package cronna

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

	if len(parts) > 0 {
		for _, partConfig := range parts {
			partConfig(part)
		}
	}

	return part
}

func (p *ExpressionPart) Minute() ExpressionDataField { return p._minute }

func (p *ExpressionPart) Hour() ExpressionDataField { return p._hour }

func (p *ExpressionPart) Month() ExpressionDataField { return p._month }

func (p *ExpressionPart) DayOfWeek() ExpressionDataField { return p._dayOfWeek }

func (p *ExpressionPart) DayOfMonth() ExpressionDataField { return p._dayOfMonth }
