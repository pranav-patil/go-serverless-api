package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/pranav-patil/go-serverless-api/func/api/middleware"
	"github.com/pranav-patil/go-serverless-api/func/api/models"
	"github.com/pranav-patil/go-serverless-api/pkg/constant"
	pkgDynamoDB "github.com/pranav-patil/go-serverless-api/pkg/dynamodb"
	dynamoMocks "github.com/pranav-patil/go-serverless-api/pkg/dynamodb/mocks"
	"github.com/pranav-patil/go-serverless-api/pkg/dynamodb/model"
	"github.com/pranav-patil/go-serverless-api/pkg/mockutil"
	pkgS3 "github.com/pranav-patil/go-serverless-api/pkg/s3"
	s3Mocks "github.com/pranav-patil/go-serverless-api/pkg/s3/mocks"
	"github.com/stretchr/testify/suite"
)

type BookmarksUpdateTestSuite struct {
	suite.Suite

	ctrl               *gomock.Controller
	recorder           *httptest.ResponseRecorder
	context            *gin.Context
	mockS3Client       *s3Mocks.MockS3Client
	mockDynamoDBClient *dynamoMocks.MockDynamoDBClient
}

var bookmarksUpdateRequest = models.BookmarkList{
	BookmarkEntry: []models.BookmarkEntry{
		{URL: "https://docs.ai21.com/docs/jurassic-2-models"},
		{URL: "https://jalammar.github.io/illustrated-transformer/"},
	},
}

var distributionSuccess = &model.UserBookmarks{
	Status:         constant.Success,
	StartTimestamp: time.Now(),
	EndTimestamp:   time.Now(),
	UserId:         "1",
}

var distributionPending = &model.UserBookmarks{
	Status:         constant.Pending,
	StartTimestamp: time.Now(),
	EndTimestamp:   time.Time{},
	UserId:         "1",
}

func TestBookmarksUpdateSuite(t *testing.T) {
	suite.Run(t, new(BookmarksUpdateTestSuite))
}

func (s *BookmarksUpdateTestSuite) SetupSuite() {
	s.T().Setenv("BOOKMARKS_BUCKET", "test_bucket")
	s.ctrl = gomock.NewController(s.T())
}

func (s *BookmarksUpdateTestSuite) SetupTest() {
	s.recorder = httptest.NewRecorder()
	s.context = mockutil.MockGinContext(s.recorder)
	s.context.Set(middleware.UserIDCxt, "1")

	s.mockS3Client = s3Mocks.NewMockS3Client(s.ctrl)
	NewS3Client = func() (pkgS3.S3Client, error) {
		return s.mockS3Client, nil
	}

	s.mockDynamoDBClient = dynamoMocks.NewMockDynamoDBClient(s.ctrl)
	NewDynamoDBClient = func() (pkgDynamoDB.DynamoDBClient, error) {
		return s.mockDynamoDBClient, nil
	}
}

func (s *BookmarksUpdateTestSuite) TestPostBookmarksWhereExistingVersionExists() {
	mockutil.MockJSONRequest(s.context, "POST", nil, bookmarksUpdateRequest)

	distribution := &model.UserBookmarks{UserId: "1"}
	distributionSuccess.LatestVersion = TestLatestVersion
	s.mockDynamoDBClient.EXPECT().GetRecordByKey(gomock.Eq(distribution)).Return(distributionSuccess, nil)

	s.mockDynamoDBClient.EXPECT().UpdateRecordsByKey(mockutil.AnyOfType(distribution)).Return(nil)

	s3Content := `{"bookmarks": [
				{ "url": "https://docs.ai21.com/docs/jurassic-2-models" },
				{ "url": "198.52.101.0/24" },
				{ "url": "https://chat.openai.com" }
			]}`

	s.mockS3Client.EXPECT().GetObject(gomock.Eq("test_bucket"),
		gomock.Eq("Bookmarks/1/1.0.89")).Return([]byte(s3Content), nil)

	s.mockS3Client.EXPECT().PutObject(gomock.Eq("test_bucket"),
		gomock.Eq("Bookmarks/1/1.0.90"), gomock.Eq(JSON), gomock.Eq(pkgS3.GZip),
		gomock.Any()).Return(nil)

	s.mockS3Client.EXPECT().DeleteObject(gomock.Eq("test_bucket"),
		gomock.Eq("Bookmarks/1/1.0.89")).Return(nil)

	PostBookmarks(s.context)

	s.EqualValues(http.StatusCreated, s.recorder.Code)
}

func (s *BookmarksUpdateTestSuite) TestPostBookmarksWhereNoVersionExists() {
	mockutil.MockJSONRequest(s.context, "POST", nil, bookmarksUpdateRequest)

	distribution := &model.UserBookmarks{UserId: "1"}
	s.mockDynamoDBClient.EXPECT().GetRecordByKey(gomock.Eq(distribution)).Return(nil, nil)

	s.mockDynamoDBClient.EXPECT().AddRecord(mockutil.AnyOfType(distribution)).Return(nil)

	s.mockS3Client.EXPECT().PutObject(gomock.Eq("test_bucket"),
		gomock.Eq("Bookmarks/1/1.0.1"), gomock.Eq(JSON), gomock.Eq(pkgS3.GZip),
		gomock.Any()).Return(nil)

	PostBookmarks(s.context)

	s.EqualValues(http.StatusCreated, s.recorder.Code)
}

func (s *BookmarksUpdateTestSuite) TestPostBookmarksWhenInValidBookmarks() {
	mockutil.MockJSONRequest(s.context, "POST", nil, inValidBookmarksRequest)

	PostBookmarks(s.context)

	var invalidResponse models.InValidBookmarksResponse
	bookmarksJSON := s.recorder.Body.String()
	err := json.Unmarshal([]byte(bookmarksJSON), &invalidResponse)

	s.NoError(err)
	s.Equal(2, len(invalidResponse.BookmarkList))
	s.EqualValues(http.StatusBadRequest, s.recorder.Code)
	s.EqualValues([]models.BookmarkEntry{
		{URL: "htps://www.run.ai/guides/gpu-deep-learning/best-gpu-for-deep-learning"},
		{URL: "https/www.simplilearn.com/keras-vs-tensorflow-vs-pytorch-article"}}, invalidResponse.BookmarkList)
}

func (s *BookmarksUpdateTestSuite) TestPostBookmarksWhenDistributionPending() {
	mockutil.MockJSONRequest(s.context, "POST", nil, bookmarksUpdateRequest)

	distribution := &model.UserBookmarks{UserId: "1"}
	distributionPending.LatestVersion = TestLatestVersion
	s.mockDynamoDBClient.EXPECT().GetRecordByKey(gomock.Eq(distribution)).Return(distributionPending, nil)

	PostBookmarks(s.context)

	s.EqualValues(http.StatusForbidden, s.recorder.Code)
}

func (s *BookmarksUpdateTestSuite) TestDeleteBookmarks() {
	mockutil.MockJSONRequest(s.context, "DELETE", nil, nil)

	distribution := &model.UserBookmarks{UserId: "1"}
	mockdist.Status = constant.Failed

	var distributionResult = &model.UserBookmarks{
		UserId:            "1",
		Status:            constant.Failed,
		LatestVersion:     "1.0.45",
		ModifiedBookmarks: false,
	}
	s.mockDynamoDBClient.EXPECT().GetRecordByKey(gomock.Eq(distribution)).Return(distributionResult, nil)

	s.mockDynamoDBClient.EXPECT().UpdateRecordsByKey(mockutil.AnyOfType(distribution)).Return(nil)

	s.mockS3Client.EXPECT().DeleteObject(gomock.Eq("test_bucket"),
		gomock.Eq("Bookmarks/1/1.0.45")).Return(nil)

	DeleteBookmarks(s.context)

	s.EqualValues(http.StatusAccepted, s.recorder.Code)
	s.Equal(`{"message":"Bookmarks deletion complete"}`, s.recorder.Body.String())
}

func (s *BookmarksUpdateTestSuite) TestDeleteBookmarksWhenDistVersionAlreadyDeleted() {
	mockutil.MockJSONRequest(s.context, "DELETE", nil, nil)

	distribution := &model.UserBookmarks{UserId: "1"}
	mockdist.Status = constant.Failed

	var distributionResult = &model.UserBookmarks{
		UserId:            "1",
		Status:            constant.Success,
		LatestVersion:     "1.0.45_DELETED",
		ModifiedBookmarks: false,
	}
	s.mockDynamoDBClient.EXPECT().GetRecordByKey(gomock.Eq(distribution)).Return(distributionResult, nil)

	DeleteBookmarks(s.context)

	s.EqualValues(http.StatusForbidden, s.recorder.Code)
	s.Equal(`{"error":"no bookmarks found to delete"}`, s.recorder.Body.String())
}
