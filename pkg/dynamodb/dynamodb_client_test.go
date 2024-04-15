package dynamodb

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/golang/mock/gomock"
	"github.com/pranav-patil/go-serverless-api/pkg/constant"

	"github.com/pranav-patil/go-serverless-api/pkg/dynamodb/mocks"
	"github.com/pranav-patil/go-serverless-api/pkg/dynamodb/model"
	"github.com/stretchr/testify/suite"
)

const MockDeviceDistributionTableName = "Device_Distribution"

var (
	appDistribution = &model.BookmarkDistribution{
		Status:         constant.Pending,
		StartTimestamp: time.Now(),
		EndTimestamp:   time.Now(),
		UserId:         "23434",
		DeviceId:       "23424234234",
	}
)

type UserBookmarkEntry struct {
	PK            string `dynamodbav:"PK"`
	SK            string `dynamodbav:"SK"`
	UserId        string `dynamodbav:"userId,omitempty" partitionKey:"UID"`
	BookmarkEntry string `dynamodbav:"bookmarkEntry,omitempty"`
	FirstAddr     uint32 `dynamodbav:"firstAddr,omitempty" sortKey:"FAD"` // First address of CIDR block
	LastAddr      uint32 `dynamodbav:"lastAddr,omitempty" sortKey:"LAD"`  // Last address of CIDR block
}

func (userBookmarkEntry *UserBookmarkEntry) GetTableName() string {
	return "IPFiltering_Bookmarks"
}

func (userBookmarkEntry *UserBookmarkEntry) String() string {
	return fmt.Sprintf("UserId: %v\n\tFirst Address: %v\n\tLast Address: %v\n",
		userBookmarkEntry.UserId, userBookmarkEntry.FirstAddr, userBookmarkEntry.LastAddr)
}

type DynamoDBClientTestSuite struct {
	suite.Suite

	ctrl               *gomock.Controller
	mockDynamoDBClient *mocks.MockAWSDynamoDBClient
	api                dynamodbAPI
}

func TestDynamoDBClientSuite(t *testing.T) {
	suite.Run(t, new(DynamoDBClientTestSuite))
}

func (s *DynamoDBClientTestSuite) SetupSuite() {
	s.ctrl = gomock.NewController(s.T())
}

func (s *DynamoDBClientTestSuite) SetupTest() {
	s.mockDynamoDBClient = mocks.NewMockAWSDynamoDBClient(s.ctrl)
	s.api = dynamodbAPI{DynamoDB: s.mockDynamoDBClient}
}

func (s *DynamoDBClientTestSuite) TestTableExists() {
	tableName := MockDeviceDistributionTableName

	ctx := context.TODO()
	input := dynamodb.DescribeTableInput{TableName: aws.String(tableName)}

	s.mockDynamoDBClient.EXPECT().DescribeTable(ctx, &input).Return(&dynamodb.DescribeTableOutput{}, nil).Times(1)

	created, err := s.api.TableExists(tableName)

	s.NoError(err)
	s.Equal(true, created)
}

func (s *DynamoDBClientTestSuite) TestListTables() {
	ctx := context.TODO()
	input := dynamodb.ListTablesInput{}
	expectedTables := []string{"Table1", "Table2"}

	s.mockDynamoDBClient.EXPECT().ListTables(ctx, &input).Return(&dynamodb.ListTablesOutput{
		TableNames:             expectedTables,
		LastEvaluatedTableName: aws.String("Table2"),
	}, nil).Times(1)

	tables, err := s.api.ListTables()

	s.NoError(err)
	s.Equal(expectedTables, tables)
}

func (s *DynamoDBClientTestSuite) TestAddRecord() {
	ctx := context.TODO()

	s.mockDynamoDBClient.EXPECT().PutItem(ctx,
		gomock.AssignableToTypeOf(&dynamodb.PutItemInput{})).Return(
		&dynamodb.PutItemOutput{}, nil).Times(1)

	err := s.api.AddRecord(appDistribution)
	s.NoError(err)
}

func (s *DynamoDBClientTestSuite) TestAddBatchRecords() {
	ctx := context.TODO()

	s.mockDynamoDBClient.EXPECT().BatchWriteItem(ctx,
		gomock.AssignableToTypeOf(&dynamodb.BatchWriteItemInput{})).Return(
		&dynamodb.BatchWriteItemOutput{}, nil).Times(1)

	addressBookmarks := []model.Entity{
		&UserBookmarkEntry{
			UserId:        "1",
			BookmarkEntry: "https://www.run.ai/guides/gpu-deep-learning/best-gpu-for-deep-learning",
			FirstAddr:     23,
			LastAddr:      65,
		},
		&UserBookmarkEntry{
			UserId:        "1",
			BookmarkEntry: "https://www.simplilearn.com/keras-vs-tensorflow-vs-pytorch-article",
			FirstAddr:     78,
			LastAddr:      12,
		},
	}

	err := s.api.AddBatchRecords(addressBookmarks)
	s.NoError(err)
}

func (s *DynamoDBClientTestSuite) TestGetRecordsByKeyAndFieldsLimitWithPKKey() {
	distribution := new(model.UserBookmarks)
	distribution.UserId = "70138"
	ctx := context.TODO()

	queryOutput := &dynamodb.QueryOutput{
		Count: 1,
		Items: []map[string]types.AttributeValue{
			{
				"UserId":                  &types.AttributeValueMemberS{Value: "70138"},
				"LastDistributionVersion": &types.AttributeValueMemberS{Value: "1.0.34"},
				"Enabled":                 &types.AttributeValueMemberBOOL{Value: false},
			},
		},
	}

	s.mockDynamoDBClient.EXPECT().Query(ctx,
		gomock.AssignableToTypeOf(&dynamodb.QueryInput{})).Return(queryOutput, nil).Times(1)

	result, err := s.api.GetRecordsByKeyAndFieldsLimit(distribution, -1, true)
	outRows := result.([]model.UserBookmarks)

	s.NoError(err)
	s.Equal(1, len(outRows))
}

func (s *DynamoDBClientTestSuite) TestGetRecordsByKeyAndFieldsLimitWithPKSKKeys() {
	addEntry := new(UserBookmarkEntry)
	addEntry.UserId = "70138"
	ctx := context.TODO()

	queryOutput := &dynamodb.QueryOutput{
		Count: 1,
		Items: []map[string]types.AttributeValue{
			{
				"UserId":        &types.AttributeValueMemberS{Value: "70138"},
				"BookmarkEntry": &types.AttributeValueMemberS{Value: "1.0.34"},
				"FirstAddr":     &types.AttributeValueMemberN{Value: "345435"},
				"LastAddr":      &types.AttributeValueMemberN{Value: "67567"},
				"Tags":          &types.AttributeValueMemberN{Value: "TopSecure"},
			},
			{
				"UserId":        &types.AttributeValueMemberS{Value: "70138"},
				"BookmarkEntry": &types.AttributeValueMemberS{Value: "172.0.1.23"},
				"FirstAddr":     &types.AttributeValueMemberN{Value: "678554"},
				"LastAddr":      &types.AttributeValueMemberN{Value: "567566"},
				"Tags":          &types.AttributeValueMemberN{Value: "Mild"},
			},
		},
	}

	s.mockDynamoDBClient.EXPECT().Query(ctx,
		gomock.AssignableToTypeOf(&dynamodb.QueryInput{})).Return(queryOutput, nil).Times(1)

	result, err := s.api.GetRecordsByKeyAndFieldsLimit(addEntry, -1, true)
	outRows := result.([]UserBookmarkEntry)

	s.NoError(err)
	s.Equal(2, len(outRows))
}

func (s *DynamoDBClientTestSuite) TestGetRecordsByKeyAndExprLimit() {
	distribution := new(model.UserBookmarks)
	distribution.UserId = "70138"
	ctx := context.TODO()

	queryOutput := &dynamodb.QueryOutput{
		Count: 1,
		Items: []map[string]types.AttributeValue{
			{
				"UserId":                  &types.AttributeValueMemberS{Value: "70138"},
				"LastDistributionVersion": &types.AttributeValueMemberS{Value: "1.0.34"},
				"Enabled":                 &types.AttributeValueMemberBOOL{Value: false},
			},
		},
	}

	s.mockDynamoDBClient.EXPECT().Query(ctx,
		gomock.AssignableToTypeOf(&dynamodb.QueryInput{})).Return(queryOutput, nil).Times(1)

	distCond := expression.GreaterThanEqual(expression.Name("LastDistributionVersion"), expression.Value("1.0.34"))

	result, err := s.api.GetRecordsByKeyAndExprLimit(distribution, &distCond, nil, -1, true)
	outRows := result.([]model.UserBookmarks)

	s.NoError(err)
	s.Equal(1, len(outRows))
}

func (s *DynamoDBClientTestSuite) TestGetRecordsByKeyAndExprLimitWhenResultIsEmpty() {
	distribution := new(model.UserBookmarks)
	distribution.UserId = "70138"
	ctx := context.TODO()

	queryOutput := &dynamodb.QueryOutput{
		Count: 0,
		Items: []map[string]types.AttributeValue{},
	}

	s.mockDynamoDBClient.EXPECT().Query(ctx,
		gomock.AssignableToTypeOf(&dynamodb.QueryInput{})).Return(queryOutput, nil).Times(1)

	distCond := expression.GreaterThanEqual(expression.Name("LastDistributionVersion"), expression.Value("1.0.34"))

	result, err := s.api.GetRecordsByKeyAndExprLimit(distribution, &distCond, nil, -1, true)
	outRows := result.([]model.UserBookmarks)

	s.NoError(err)
	s.Equal(0, len(outRows))
}

func (s *DynamoDBClientTestSuite) TestUpdateRecordsByKey() {
	ctx := context.TODO()
	input := dynamodb.UpdateItemInput{}

	s.mockDynamoDBClient.EXPECT().UpdateItem(ctx,
		gomock.AssignableToTypeOf(&input)).Return(&dynamodb.UpdateItemOutput{}, nil).Times(1)

	distribution := &model.UserBookmarks{
		UserId:            "12900",
		LatestVersion:     "1.0.67",
		ModifiedBookmarks: true,
	}
	err := s.api.UpdateRecordsByKey(distribution)

	s.NoError(err)
}

func (s *DynamoDBClientTestSuite) TestDeleteTable() {
	tableName := MockDeviceDistributionTableName

	ctx := context.TODO()
	input := dynamodb.DeleteTableInput{TableName: aws.String(tableName)}

	s.mockDynamoDBClient.EXPECT().DeleteTable(ctx, &input).Return(&dynamodb.DeleteTableOutput{}, nil).Times(1)

	err := s.api.DeleteTable(tableName)

	s.NoError(err)
}

func (s *DynamoDBClientTestSuite) TestGetKeyValue() {
	testCases := []struct {
		testName string
		entity   model.Entity
		keyTag   string
		keyValue string
	}{
		{"UserBookmark entity",
			&UserBookmarkEntry{
				UserId:        "23424",
				BookmarkEntry: "10.0.0.1/32",
				FirstAddr:     34,
				LastAddr:      56,
			},
			model.SortKeyTag,
			"FAD#34#LAD#56"},

		{"UserBookmark with default values entity",
			&UserBookmarkEntry{
				UserId:        "58678",
				BookmarkEntry: "",
				FirstAddr:     0,
				LastAddr:      0,
			},
			model.SortKeyTag,
			""},

		{"Distribution entity",
			&model.UserBookmarks{
				UserId:      "23424",
				SyncEnabled: true,
			},
			model.PartitionKeyTag,
			"UID#23424"},

		{"Device Distribution entity",
			&model.BookmarkDistribution{
				UserId:         "1",
				DeviceId:       "44364564",
				Status:         constant.Success,
				StatusMessage:  "",
				StartTimestamp: time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC),
				EndTimestamp:   time.Time{},
			},
			model.SortKeyTag,
			"DID#44364564",
		},
	}

	for _, testCase := range testCases {
		s.Run(testCase.testName, func() {
			s.T().Log(testCase.testName)
			actualKey, err := getKeyValue(testCase.entity, testCase.keyTag)
			s.NoError(err)
			s.EqualValues(testCase.keyValue, actualKey)
		})
	}
}

func (s *DynamoDBClientTestSuite) TestLoadKeysAndConvertToMap() {
	testCases := []struct {
		testName    string
		entity      model.Entity
		expectedMap map[string]interface{}
	}{
		{"UserBookmark entity",
			&UserBookmarkEntry{
				UserId:        "3850",
				BookmarkEntry: "1.0",
				FirstAddr:     43567,
				LastAddr:      86,
			},
			map[string]interface{}{
				"PK":            "UID#3850",
				"SK":            "FAD#43567#LAD#86",
				"bookmarkEntry": "1.0",
			}},

		{"UserBookmark entity with empty sort keys",
			&UserBookmarkEntry{
				UserId: "8293",
			},
			map[string]interface{}{
				"PK": "UID#8293",
			}},

		{"Distribution entity",
			&model.UserBookmarks{
				UserId:      "453234",
				SyncEnabled: true,
			},
			map[string]interface{}{
				"PK":          "UID#453234",
				"syncEnabled": true,
			}},

		{"Device Distribution entity",
			&model.BookmarkDistribution{
				UserId:         "67",
				DeviceId:       "674546",
				Status:         constant.Success,
				StatusMessage:  "",
				StartTimestamp: time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC),
				EndTimestamp:   time.Time{},
			},
			map[string]interface{}{
				"PK":      "UID#67",
				"SK":      "DID#674546",
				"status":  "Success",
				"startTs": time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC),
			}},
	}

	for _, testCase := range testCases {
		s.Run(testCase.testName, func() {
			s.T().Log(testCase.testName)
			actualMap, err := loadKeysAndConvertToMap(testCase.entity)
			s.NoError(err)
			s.Equal(len(testCase.expectedMap), len(actualMap))
			s.EqualValues(testCase.expectedMap, actualMap)
		})
	}
}
