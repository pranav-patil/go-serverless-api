package dynamodb

import "github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"

func GenKeyConditionBuilder(data map[string]interface{}) expression.KeyConditionBuilder {
	firstKey := true
	var keyCondition expression.KeyConditionBuilder
	for k, v := range data {
		kcb := expression.Key(k).Equal(expression.Value(v))
		if firstKey {
			keyCondition = kcb
			firstKey = false
		} else {
			keyCondition = keyCondition.And(kcb)
		}
	}

	return keyCondition
}

func GenConditionBuilder(data map[string]interface{}) expression.ConditionBuilder {
	firstKey := true
	var condBuilder expression.ConditionBuilder
	for k, v := range data {
		cb := expression.Name(k).Equal(expression.Value(v))
		if firstKey {
			condBuilder = cb
			firstKey = false
		} else {
			condBuilder = condBuilder.And(cb)
		}
	}

	return condBuilder
}

func GenUpdateBuilder(data map[string]interface{}) expression.UpdateBuilder {
	firstKey := true
	var updateBuilder expression.UpdateBuilder
	for k, v := range data {
		if firstKey {
			updateBuilder = expression.Set(expression.Name(k), expression.Value(v))
			firstKey = false
		} else {
			updateBuilder = updateBuilder.Set(expression.Name(k), expression.Value(v))
		}
	}

	return updateBuilder
}

func GenerateExpression(data map[string]interface{}, filter *expression.ConditionBuilder,
	projection *expression.ProjectionBuilder) (expression.Expression, error) {
	expBuilder := expression.NewBuilder()

	if filter != nil {
		expBuilder = expBuilder.WithFilter(*filter)
	}
	if projection != nil {
		expBuilder = expBuilder.WithProjection(*projection)
	}
	if data == nil {
		return expBuilder.Build()
	}
	return expBuilder.WithKeyCondition(GenKeyConditionBuilder(data)).Build()
}
