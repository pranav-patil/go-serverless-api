package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/pranav-patil/go-serverless-api/pkg/dynamodb/model"
	"github.com/pranav-patil/go-serverless-api/pkg/env"
	"github.com/pranav-patil/go-serverless-api/pkg/mockutil"
	"github.com/pranav-patil/go-serverless-api/pkg/sizedwaitgroup"
	"github.com/pranav-patil/go-serverless-api/pkg/util"
	"github.com/rs/zerolog/log"
)

//go:generate mockgen -destination mocks/dynamodb_client_mock.go -package mocks . DynamoDBClient

type DynamoDBClient interface {
	CreateTableIfNotExists(entity model.Entity) (*types.TableDescription, error)
	CreateTableByKeysIfNotExists(tableName string, keySchema []types.KeySchemaElement,
		attrDefs []types.AttributeDefinition) (*types.TableDescription, error)
	TableExists(tableName string) (bool, error)
	ListTables() ([]string, error)
	AddRecord(entity model.Entity) error
	AddBatchRecords(entities []model.Entity) error
	GetAllRecords(entity model.Entity, filter *expression.ConditionBuilder,
		projection *expression.ProjectionBuilder) (interface{}, error)
	GetRecordByKey(entity model.Entity) (model.Entity, error)
	GetRecordsByKeyAndFields(entity model.Entity) (interface{}, error)
	GetRecordsByKeyAndFieldsLimit(entity model.Entity,
		limit int32, scanIndex bool) (interface{}, error)
	GetRecordsByKeyAndExprLimit(entity model.Entity, filter *expression.ConditionBuilder,
		projection *expression.ProjectionBuilder, limit int32, scanIndex bool) (interface{}, error)
	GetRecordsByPagination(entity model.Entity, pageLimit int32,
		lastEvaluatedKey map[string]types.AttributeValue,
		scanIndexForward bool) (interface{}, map[string]types.AttributeValue, error)
	UpdateRecordsByKey(entity model.Entity) error
	UpdateRecordsByParams(entity model.Entity, queryParams map[string]interface{}) error
	UpdateRecordsByExpression(entity model.Entity, expr expression.Expression) error
	DeleteRecordByKey(entity model.Entity) error
	DeleteRecordByKeyAndFields(entity model.Entity) error
	DeleteRecordByKeyAndExpression(entity model.Entity, condExp *expression.ConditionBuilder) error
	DeleteBatchRecords(entity model.Entity, inputFilter *expression.ConditionBuilder) (int, error)
	DeleteTable(tableName string) error
}

type dynamodbAPI struct {
	DynamoDB AWSDynamoDBClient
}

const (
	retryAttempt   int64 = 100
	maxBatchSize   int   = 25
	maxConcurrency int   = 40
)

func NewDynamoDBClient() (DynamoDBClient, error) {
	dynamodbAPI := new(dynamodbAPI)

	var cfg aws.Config
	var err error

	if env.IsLocalOrTestEnv() {
		cfg, err = mockutil.GetLocalStackConfig()
	} else {
		cfg, err = config.LoadDefaultConfig(context.TODO())
	}

	if err != nil {
		log.Error().Msgf("S3 LoadDefaultConfig Error: %v", err.Error())
		return dynamodbAPI, err
	}

	dynamodbAPI.DynamoDB = dynamodb.NewFromConfig(cfg)
	return dynamodbAPI, nil
}

func (api *dynamodbAPI) CreateTableIfNotExists(entity model.Entity) (*types.TableDescription, error) {
	keySchema := []types.KeySchemaElement{{
		AttributeName: aws.String("PK"),
		KeyType:       types.KeyTypeHash,
	}}

	keyAttributes := []types.AttributeDefinition{{
		AttributeName: aws.String("PK"),
		AttributeType: types.ScalarAttributeTypeS,
	}}

	sKey, err := getKeyValue(entity, model.SortKeyTag)
	if err == nil && sKey != "" {
		keySchema = append(keySchema, types.KeySchemaElement{
			AttributeName: aws.String("SK"),
			KeyType:       types.KeyTypeRange,
		})

		keyAttributes = append(keyAttributes, types.AttributeDefinition{
			AttributeName: aws.String("SK"),
			AttributeType: types.ScalarAttributeTypeS,
		})
	}

	return api.CreateTableByKeysIfNotExists(entity.GetTableName(), keySchema, keyAttributes)
}
func (api *dynamodbAPI) CreateTableByKeysIfNotExists(tableName string, keySchema []types.KeySchemaElement,
	attrDefs []types.AttributeDefinition) (*types.TableDescription, error) {
	var tableDesc *types.TableDescription

	exists, err := api.TableExists(tableName)
	if err != nil || exists {
		return tableDesc, err
	}

	tableInput := dynamodb.CreateTableInput{
		TableName:            aws.String(tableName),
		BillingMode:          types.BillingModePayPerRequest,
		AttributeDefinitions: attrDefs,
		KeySchema:            keySchema,
	}

	table, err := api.DynamoDB.CreateTable(context.TODO(), &tableInput)

	if err != nil {
		log.Error().Msgf("Couldn't create table %v: %v\n", tableName, err)
	} else {
		waiter := dynamodb.NewTableExistsWaiter(api.DynamoDB)
		err = waiter.Wait(context.TODO(),
			&dynamodb.DescribeTableInput{
				TableName: aws.String(tableName)},
			2*time.Minute,
			func(o *dynamodb.TableExistsWaiterOptions) {
				o.MaxDelay = 5 * time.Second
				o.MinDelay = 5 * time.Second
			})

		if err != nil {
			log.Error().Msgf("Wait for table exists failed: %v\n", err)
		}
		tableDesc = table.TableDescription
		log.Info().Msgf("Table %v created successfully.\n", tableName)
	}
	return tableDesc, err
}

func (api *dynamodbAPI) TableExists(tableName string) (bool, error) {
	exists := true
	_, err := api.DynamoDB.DescribeTable(
		context.TODO(), &dynamodb.DescribeTableInput{TableName: aws.String(tableName)},
	)
	if err != nil {
		var notFoundEx *types.ResourceNotFoundException
		if errors.As(err, &notFoundEx) {
			log.Warn().Msgf("Table %v does not exist.\n", tableName)
			err = nil
		} else {
			log.Error().Msgf("Couldn't determine existence of table %v: %v\n", tableName, err)
		}
		exists = false
	}
	return exists, err
}

func (api *dynamodbAPI) ListTables() ([]string, error) {
	tables, err := api.DynamoDB.ListTables(context.TODO(), &dynamodb.ListTablesInput{})
	if err != nil {
		return make([]string, 0), err
	}
	return tables.TableNames, err
}

func (api *dynamodbAPI) AddRecord(entity model.Entity) error {
	err := loadEntityKeys(entity)
	if err != nil {
		return err
	}

	item, err := attributevalue.MarshalMap(entity)

	if err != nil {
		return err
	}
	_, err = api.DynamoDB.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(entity.GetTableName()),
		Item:      item,
	})
	return err
}

func (api *dynamodbAPI) AddBatchRecords(entities []model.Entity) (err error) {
	totalEntities := len(entities)

	if totalEntities == 0 {
		return fmt.Errorf("entities to add are empty")
	}

	swg := sizedwaitgroup.New(maxConcurrency)
	wgDone := make(chan bool)
	fatalErrors := make(chan error)

	for i := 0; i < totalEntities; i += maxBatchSize {
		j := i + maxBatchSize

		if j > totalEntities {
			j = totalEntities
		}

		entityBatch := entities[i:j]
		swg.Add()

		go func() {
			defer swg.Done()
			err = api.processEntityBatch(entityBatch)
			if err != nil {
				fatalErrors <- err
			}
		}()
	}

	go func() {
		swg.Wait()
		close(wgDone)
	}()

	select {
	case <-wgDone:
		break
	case err = <-fatalErrors:
		log.Error().Msgf("error in entity batch processing goroutine: %v\n", err)
		close(fatalErrors)
	}

	return err
}

func (api *dynamodbAPI) GetAllRecords(entity model.Entity, filter *expression.ConditionBuilder,
	projection *expression.ProjectionBuilder) (interface{}, error) {
	expr, err := GenerateExpression(nil, filter, projection)
	if err != nil {
		return nil, err
	}

	scanInput := dynamodb.ScanInput{
		TableName:                 aws.String(entity.GetTableName()),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
	}

	result, err := api.DynamoDB.Scan(context.TODO(), &scanInput)
	var outRows interface{}

	if err == nil {
		outRows, err = convertItemsToSlice(entity, result.Count, result.Items)
	}

	return outRows, err
}

// Zero/Empty field values will not be used in the query.
func (api *dynamodbAPI) GetRecordByKey(entity model.Entity) (model.Entity, error) {
	response, err := api.DynamoDB.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(entity.GetTableName()),
		Key:       getKeys(entity),
	})
	if err != nil { // getItem operation failed
		return nil, err
	}

	if response.Item == nil { // no matching record found
		return nil, nil
	}

	err = attributevalue.UnmarshalMap(response.Item, entity)
	if err != nil {
		return nil, err
	}

	// otherwise the matching record is successfully retrieved
	return entity, nil
}

func (api *dynamodbAPI) GetRecordsByKeyAndFields(entity model.Entity) (interface{}, error) {
	return api.GetRecordsByKeyAndFieldsLimit(entity, -1, true)
}

// Get all records by Keys and other field values from the passed Entity.
// Zero/Empty field values will not be used in the query.
func (api *dynamodbAPI) GetRecordsByKeyAndFieldsLimit(entity model.Entity,
	limit int32, scanIndex bool) (interface{}, error) {
	queryParams, err := loadKeysAndConvertToMap(entity)
	if err != nil {
		return nil, err
	}

	keyCondition := GenKeyConditionBuilder(queryParams)
	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).Build()
	if err != nil {
		return nil, err
	}

	return api.getRecordsByExpression(entity, expr, limit, scanIndex)
}

func (api *dynamodbAPI) GetRecordsByKeyAndExprLimit(entity model.Entity, filter *expression.ConditionBuilder,
	projection *expression.ProjectionBuilder, limit int32, scanIndex bool) (interface{}, error) {
	queryParams, err := loadKeysAndConvertToMap(entity)
	if err != nil {
		return nil, err
	}

	expr, err := GenerateExpression(queryParams, filter, projection)
	if err != nil {
		return nil, err
	}

	return api.getRecordsByExpression(entity, expr, limit, scanIndex)
}

func (api *dynamodbAPI) getRecordsByExpression(entity model.Entity, expr expression.Expression,
	limit int32, scanIndex bool) (interface{}, error) {
	queryInput := dynamodb.QueryInput{
		TableName:                 aws.String(entity.GetTableName()),
		ConsistentRead:            aws.Bool(true),
		KeyConditionExpression:    expr.KeyCondition(),
		ProjectionExpression:      expr.Projection(),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ScanIndexForward:          aws.Bool(scanIndex),
	}

	if limit > 0 {
		queryInput.Limit = aws.Int32(limit)
	}

	result, err := api.DynamoDB.Query(context.TODO(), &queryInput)
	var outRows interface{}

	if err == nil {
		outRows, err = convertItemsToSlice(entity, result.Count, result.Items)
	}

	return outRows, err
}

func (api *dynamodbAPI) GetRecordsByPagination(entity model.Entity, pageLimit int32, lastEvalKey map[string]types.AttributeValue,
	scanIndexForward bool) (result interface{}, lastKey map[string]types.AttributeValue, err error) {
	queryParams, err := loadKeysAndConvertToMap(entity)
	if err != nil {
		return nil, nil, err
	}

	keyCondition := GenKeyConditionBuilder(queryParams)
	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).Build()
	if err != nil {
		return nil, nil, err
	}

	queryInput := &dynamodb.QueryInput{
		TableName:                 aws.String(entity.GetTableName()),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
		Limit:                     aws.Int32(pageLimit),
		ScanIndexForward:          aws.Bool(scanIndexForward),
		ConsistentRead:            aws.Bool(false),
	}

	if lastEvalKey != nil {
		queryInput.ExclusiveStartKey = lastEvalKey
	}

	p := dynamodb.NewQueryPaginator(api.DynamoDB, queryInput)
	if err != nil {
		return nil, nil, err
	}

	var collectiveResult []map[string]types.AttributeValue
	ctxt := context.TODO()
	var singlePage *dynamodb.QueryOutput

	for {
		if !p.HasMorePages() {
			fmt.Println("no more records in the partition")
			lastEvalKey = nil
			break
		}
		singlePage, err = p.NextPage(ctxt)
		if err != nil {
			return nil, lastEvalKey, err
		}
		pendingItems := int(pageLimit) - len(collectiveResult)
		if int(singlePage.Count) >= pendingItems {
			collectiveResult = append(collectiveResult, singlePage.Items[:pendingItems]...)
			lastEvalKey = buildExclusiveStartKey(singlePage.Items[pendingItems-1])
			break
		}
		collectiveResult = append(collectiveResult, singlePage.Items...)
	}

	result, err = convertItemsToSlice(entity, int32(len(collectiveResult)), collectiveResult)
	return result, lastEvalKey, err
}

func (api *dynamodbAPI) UpdateRecordsByKey(entity model.Entity) error {
	queryParams := make(map[string]interface{})

	partitionKey, err := getKeyValue(entity, model.PartitionKeyTag)
	if err != nil {
		return err
	}
	queryParams["PK"] = partitionKey

	sortKey, err := getKeyValue(entity, model.SortKeyTag)
	if err != nil {
		return err
	}
	if sortKey != "" {
		queryParams["SK"] = sortKey
	}

	return api.UpdateRecordsByParams(entity, queryParams)
}

// The records will be updated using the values of fields within the Entity, except the
// Partition key & Sort key and the corresponding attributes which form the keys.
// The Query Parameters is used to filter records based on additional criteria.
func (api *dynamodbAPI) UpdateRecordsByParams(entity model.Entity, queryParams map[string]interface{}) error {
	updateMap, err := util.StructToMap(entity, model.DynamoDBTag, false, model.PartitionKeyTag, model.SortKeyTag)
	if err != nil {
		return err
	}
	delete(updateMap, "PK")
	delete(updateMap, "SK")

	exprBuilder := expression.NewBuilder().WithUpdate(GenUpdateBuilder(updateMap))

	if len(queryParams) > 0 {
		exprBuilder = exprBuilder.WithCondition(GenConditionBuilder(queryParams))
	}

	updateExpr, err := exprBuilder.Build()
	if err != nil {
		return err
	}

	return api.UpdateRecordsByExpression(entity, updateExpr)
}

func (api *dynamodbAPI) UpdateRecordsByExpression(entity model.Entity, expr expression.Expression) error {
	_, err := api.DynamoDB.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
		TableName:                 aws.String(entity.GetTableName()),
		Key:                       getKeys(entity),
		ConditionExpression:       expr.Condition(),
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})

	return err
}

// Deletes single record with keys.
func (api *dynamodbAPI) DeleteRecordByKey(entity model.Entity) error {
	_, err := api.DynamoDB.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
		TableName: aws.String(entity.GetTableName()),
		Key:       getKeys(entity),
	})
	return err
}

// Deletes single record with keys and field values. Zero/Empty field values will not be used in the query.
func (api *dynamodbAPI) DeleteRecordByKeyAndFields(entity model.Entity) error {
	queryParams, err := loadKeysAndConvertToMap(entity)
	if err != nil {
		return err
	}

	conditionExpression := GenConditionBuilder(queryParams)
	return api.DeleteRecordByKeyAndExpression(entity, &conditionExpression)
}

func (api *dynamodbAPI) DeleteRecordByKeyAndExpression(entity model.Entity, condExp *expression.ConditionBuilder) error {
	expr, err := expression.NewBuilder().WithCondition(*condExp).Build()
	if err != nil {
		return err
	}

	_, err = api.DynamoDB.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
		TableName:                 aws.String(entity.GetTableName()),
		Key:                       getKeys(entity),
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	return err
}

func (api *dynamodbAPI) DeleteBatchRecords(entity model.Entity, inputFilter *expression.ConditionBuilder) (int, error) {
	queryParams, err := loadKeysAndConvertToMap(entity)
	if err != nil {
		return 0, err
	}

	filter := GenConditionBuilder(queryParams)
	if inputFilter != nil {
		filter = filter.And(*inputFilter)
	}
	projection := expression.NamesList(expression.Name("PK"), expression.Name("SK"))
	expr, err := GenerateExpression(queryParams, &filter, &projection)
	if err != nil {
		return 0, err
	}

	p := dynamodb.NewScanPaginator(api.DynamoDB, &dynamodb.ScanInput{
		TableName:                 aws.String(entity.GetTableName()),
		ProjectionExpression:      expr.Projection(),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}, func(o *dynamodb.ScanPaginatorOptions) {
		o.Limit = int32(maxBatchSize)
	})

	var count int

	for p.HasMorePages() {
		var result *dynamodb.ScanOutput
		result, err = p.NextPage(context.TODO())
		if err != nil {
			return 0, err
		}

		var writeReqs []types.WriteRequest
		tableName := entity.GetTableName()
		count = int(result.Count)

		for _, item := range result.Items {
			writeReqs = append(writeReqs, types.WriteRequest{DeleteRequest: &types.DeleteRequest{Key: item}})
		}

		_, err = api.DynamoDB.BatchWriteItem(context.TODO(),
			&dynamodb.BatchWriteItemInput{
				RequestItems: map[string][]types.WriteRequest{
					tableName: writeReqs,
				},
			})
		if err != nil {
			return count, err
		}
	}

	return count, err
}

func (api *dynamodbAPI) DeleteTable(tableName string) error {
	_, err := api.DynamoDB.DeleteTable(context.TODO(), &dynamodb.DeleteTableInput{
		TableName: aws.String(tableName)})
	return err
}

func convertItemsToSlice(entity model.Entity, count int32,
	records []map[string]types.AttributeValue) (outRows interface{}, err error) {
	entityType := reflect.TypeOf(entity)
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	slice := reflect.MakeSlice(reflect.SliceOf(entityType), 0, 1)
	sliceRef := reflect.New(slice.Type())
	sliceRef.Elem().Set(slice)
	outRows = sliceRef.Interface()

	if count > 0 {
		err = attributevalue.UnmarshalListOfMaps(records, outRows)
	}

	// Slice address changes from original slice due to memory allocation
	// and copying the old data to new memory.
	outRows = (reflect.ValueOf(outRows).Elem()).Interface()
	return outRows, err
}

func getKeys(entity model.Entity) map[string]types.AttributeValue {
	pKey, err := getKeyValue(entity, model.PartitionKeyTag)
	if err != nil {
		log.Error().Msgf("%v Get PartitionKey Error: %v", entity.GetTableName(), err.Error())
		return map[string]types.AttributeValue{}
	}

	partitionKey, err := attributevalue.Marshal(pKey)
	if err != nil {
		log.Error().Msgf("%v Get PartitionKey Error: %v", entity.GetTableName(), err.Error())
		return map[string]types.AttributeValue{}
	}

	keyMap := map[string]types.AttributeValue{"PK": partitionKey}

	sKey, err := getKeyValue(entity, model.SortKeyTag)
	if err != nil {
		log.Error().Msgf("%v Get SortKey Error: %v", entity.GetTableName(), err.Error())
		return keyMap
	}

	if sKey != "" {
		sortKey, err := attributevalue.Marshal(sKey)
		if err != nil {
			log.Error().Msgf("%v  Get SortKey Error: %v", entity.GetTableName(), err.Error())
		}

		keyMap["SK"] = sortKey
	}

	return keyMap
}

func getKeyValue(entity model.Entity, keyTag string) (string, error) {
	keyMap, err := util.StructToMap(entity, keyTag, true)
	if err != nil {
		return "", err
	}
	if len(keyMap) == 0 {
		// Partition Key must be present, while Sort key is Optional
		if keyTag == model.PartitionKeyTag {
			return "", fmt.Errorf("%v key for %v is empty", keyTag, entity.GetTableName())
		} else {
			return "", nil
		}
	}

	keys := make([]string, 0, len(keyMap))
	for k := range keyMap {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	var keyBuilder strings.Builder

	for i, key := range keys {
		if fmt.Sprint(keyMap[key]) != "" {
			if i > 0 {
				keyBuilder.WriteString("#")
			}
			keyBuilder.WriteString(fmt.Sprint(key, "#", keyMap[key]))
		}
	}

	key := keyBuilder.String()
	if key == "" {
		return "", fmt.Errorf("%v key for %v is empty", keyTag, entity.GetTableName())
	}
	return key, nil
}

func loadEntityKeys(entity model.Entity) error {
	partitionKey, err := getKeyValue(entity, model.PartitionKeyTag)
	if err != nil {
		return err
	}

	util.SetStructField(entity, "PK", partitionKey)

	sortKey, err := getKeyValue(entity, model.SortKeyTag)
	if err != nil {
		return err
	}
	if sortKey != "" {
		util.SetStructField(entity, "SK", sortKey)
	}
	return nil
}

// It loads the PK and SK key fields using PartitionKeyTag and SortKeyTag tagged fields from the Entity
func loadKeysAndConvertToMap(entity model.Entity) (map[string]interface{}, error) {
	err := loadEntityKeys(entity)
	if err != nil {
		return nil, err
	}

	queryParams, err := util.StructToMap(entity, model.DynamoDBTag, true, model.PartitionKeyTag, model.SortKeyTag)
	if err != nil {
		return nil, err
	}
	return queryParams, err
}

func buildExclusiveStartKey(lastEvaluatedItem map[string]types.AttributeValue) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"PK": lastEvaluatedItem["PK"],
		"SK": lastEvaluatedItem["SK"],
	}
}

func (api *dynamodbAPI) processEntityBatch(batchSlice []model.Entity) (err error) {
	var writeReqs []types.WriteRequest

	for i := range batchSlice {
		err = loadEntityKeys(batchSlice[i])
		if err != nil {
			return err
		}

		var item map[string]types.AttributeValue
		item, err = attributevalue.MarshalMap(batchSlice[i])
		if err != nil {
			return err
		}
		writeReqs = append(writeReqs, types.WriteRequest{PutRequest: &types.PutRequest{Item: item}})
	}

	tableName := batchSlice[0].GetTableName()
	requestItems := map[string][]types.WriteRequest{tableName: writeReqs}
	var result *dynamodb.BatchWriteItemOutput

	for i := 0; i == 0 || err != nil; i++ {
		result, err = api.DynamoDB.BatchWriteItem(context.TODO(),
			&dynamodb.BatchWriteItemInput{
				RequestItems: requestItems,
			})

		if err == nil || i+1 > int(retryAttempt) {
			if len(result.UnprocessedItems) != 0 {
				err = fmt.Errorf("unable to process %v items after %v retries", result.UnprocessedItems, i)
			}
			return err
		}

		time.Sleep(time.Second * time.Duration(1))
		if result != nil && len(result.UnprocessedItems) != 0 {
			requestItems = result.UnprocessedItems
		}
	}
	return err
}
