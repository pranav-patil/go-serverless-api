package model

type Entity interface {
	GetTableName() string
	String() string
}

// Define PartitionKey and SortKey with Fieldnames using below tags in Entity types
const (
	DynamoDBTag     = "dynamodbav"
	PartitionKeyTag = "partitionKey"
	SortKeyTag      = "sortKey"
)
