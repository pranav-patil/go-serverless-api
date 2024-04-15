package handler

import (
	"encoding/json"
	"errors"
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

const (
	TestLatestVersion = "1.0.89"
)

type BookmarksTestSuite struct {
	suite.Suite

	ctrl               *gomock.Controller
	recorder           *httptest.ResponseRecorder
	context            *gin.Context
	mockS3Client       *s3Mocks.MockS3Client
	mockDynamoDBClient *dynamoMocks.MockDynamoDBClient
}

var bookmarksRequest = models.BookmarkList{
	BookmarkEntry: []models.BookmarkEntry{
		{URL: "https://docs.ai21.com/docs/jurassic-2-models"},
		{URL: "https://jalammar.github.io/illustrated-transformer/"},
	},
}

var inValidBookmarksRequest = models.BookmarkList{
	BookmarkEntry: []models.BookmarkEntry{
		{URL: "https://github.com/mlc-ai/mlc-llm"},
		{URL: "htps://www.run.ai/guides/gpu-deep-learning/best-gpu-for-deep-learning"},
		{URL: "https/www.simplilearn.com/keras-vs-tensorflow-vs-pytorch-article"},
	},
}

var mockdist = &model.UserBookmarks{
	Status:         constant.Success,
	StartTimestamp: time.Now(),
	EndTimestamp:   time.Now(),
	UserId:         "1",
	LatestVersion:  TestLatestVersion,
}

func TestBookmarksSuite(t *testing.T) {
	suite.Run(t, new(BookmarksTestSuite))
}

func (s *BookmarksTestSuite) SetupSuite() {
	s.T().Setenv("BOOKMARKS_BUCKET", "test_bucket")
	s.ctrl = gomock.NewController(s.T())
}

func (s *BookmarksTestSuite) SetupTest() {
	s.recorder = httptest.NewRecorder()
	s.context = mockutil.MockGinContext(s.recorder)
	s.context.Set(middleware.UserIDCxt, "1")

	s.mockDynamoDBClient = dynamoMocks.NewMockDynamoDBClient(s.ctrl)
	NewDynamoDBClient = func() (pkgDynamoDB.DynamoDBClient, error) {
		return s.mockDynamoDBClient, nil
	}

	s.mockS3Client = s3Mocks.NewMockS3Client(s.ctrl)
	NewS3Client = func() (pkgS3.S3Client, error) {
		return s.mockS3Client, nil
	}
}

func (s *BookmarksTestSuite) TestGetBookmarksFirstPage() {
	pathParams := []gin.Param{
		{Key: "limit", Value: "5"},
	}
	mockutil.MockJSONRequestWithQuery(s.context, "GET", pathParams, nil)
	addMocksForGetBookmarks(s)

	GetBookmarks(s.context)

	s.EqualValues(http.StatusOK, s.recorder.Code)

	var bookmarks models.BookmarkList
	bookmarksJSON := s.recorder.Body.String()
	err := json.Unmarshal([]byte(bookmarksJSON), &bookmarks)

	s.NoError(err)
	s.Equal(5, len(bookmarks.BookmarkEntry))
	s.Contains(bookmarks.BookmarkEntry, models.BookmarkEntry{URL: "https://keda.sh/docs/2.13/deploy/"})
	s.Contains(bookmarks.BookmarkEntry, models.BookmarkEntry{URL: "https://www.langchain.com/"})
}

func (s *BookmarksTestSuite) TestGetBookmarksNextPage() {
	pathParams := []gin.Param{
		{Key: "limit", Value: "3"},
		{Key: "cursor", Value: "aHR0cHM6Ly93d3cubGFuZ2NoYWluLmNvbS8="},
	}
	mockutil.MockJSONRequestWithQuery(s.context, "GET", pathParams, nil)
	addMocksForGetBookmarks(s)

	GetBookmarks(s.context)

	s.EqualValues(http.StatusOK, s.recorder.Code)

	var ipResponse models.BookmarksResponse
	bookmarksJSON := s.recorder.Body.String()
	err := json.Unmarshal([]byte(bookmarksJSON), &ipResponse)

	s.NoError(err)
	s.Equal(3, len(ipResponse.BookmarkList))
	s.Contains(ipResponse.BookmarkList, models.BookmarkEntry{URL: "https://zapier.com/blog/claude-ai/"})
	s.Contains(ipResponse.BookmarkList, models.BookmarkEntry{URL: "https://jalammar.github.io/illustrated-transformer/"})
	s.Equal("aHR0cHM6Ly96YXBpZXIuY29tL2Jsb2cvY2xhdWRlLWFpLw==", ipResponse.Next)
}

func (s *BookmarksTestSuite) TestGetBookmarksLastPage() {
	pathParams := []gin.Param{
		{Key: "limit", Value: "5"},
		{Key: "cursor", Value: "aHR0cHM6Ly96YXBpZXIuY29tL2Jsb2cvY2xhdWRlLWFpLw=="},
	}
	mockutil.MockJSONRequestWithQuery(s.context, "GET", pathParams, nil)
	addMocksForGetBookmarks(s)

	GetBookmarks(s.context)

	s.EqualValues(http.StatusOK, s.recorder.Code)

	var ipResponse models.BookmarksResponse
	bookmarksJSON := s.recorder.Body.String()
	err := json.Unmarshal([]byte(bookmarksJSON), &ipResponse)

	s.NoError(err)
	s.Equal(3, len(ipResponse.BookmarkList))
	s.Contains(ipResponse.BookmarkList, models.BookmarkEntry{URL: "https://karpenter.sh/"})
	s.Contains(ipResponse.BookmarkList, models.BookmarkEntry{URL: "https://github.com/openxla/xla"})
	s.Contains(ipResponse.BookmarkList, models.BookmarkEntry{URL: "https://deepgram.com/learn/visualizing-and-explaining-transformer-models-from-the-ground-up"})
	s.Equal("", ipResponse.Next)
}

func (s *BookmarksTestSuite) TestPutBookmarksAsJson() {
	mockutil.MockJSONRequest(s.context, "PUT", nil, bookmarksRequest)

	mockdist.Status = constant.Success
	mockdist.LatestVersion = TestLatestVersion
	distribution := &model.UserBookmarks{UserId: "1"}
	s.mockDynamoDBClient.EXPECT().GetRecordByKey(gomock.Eq(distribution)).Return(mockdist, nil)

	s.mockDynamoDBClient.EXPECT().UpdateRecordsByKey(mockutil.AnyOfType(distribution)).Return(nil)

	s.mockS3Client.EXPECT().DeleteObject(gomock.Eq("test_bucket"), gomock.Eq("Bookmarks/1/1.0.89")).Return(nil)

	s3Content := []byte(`{"bookmarks":[{"url":"https://docs.ai21.com/docs/jurassic-2-models"},{"url":"https://jalammar.github.io/illustrated-transformer/"}]}`)

	s.mockS3Client.EXPECT().PutObject(gomock.Eq("test_bucket"),
		gomock.Eq("Bookmarks/1/1.0.90"), gomock.Eq(JSON), gomock.Eq(pkgS3.GZip),
		&s3Content).Return(nil)

	PutBookmarks(s.context)

	s.EqualValues(http.StatusCreated, s.recorder.Code)
}

func (s *BookmarksTestSuite) TestPutBookmarksAsJsonWithS3Error() {
	mockutil.MockJSONRequest(s.context, "PUT", nil, bookmarksRequest)

	distribution := &model.UserBookmarks{UserId: "1"}
	mockdist.Status = constant.Success
	mockdist.LatestVersion = TestLatestVersion
	s.mockDynamoDBClient.EXPECT().GetRecordByKey(gomock.Eq(distribution)).Return(mockdist, nil)

	s3Content := []byte(`{"bookmarks":[{"url":"https://docs.ai21.com/docs/jurassic-2-models"},{"url":"https://jalammar.github.io/illustrated-transformer/"}]}`)

	s.mockS3Client.EXPECT().PutObject(gomock.Eq("test_bucket"),
		gomock.Eq("Bookmarks/1/1.0.90"), gomock.Eq(JSON), gomock.Eq(pkgS3.GZip),
		&s3Content).Return(errors.New("s3 error"))

	PutBookmarks(s.context)

	s.EqualValues(http.StatusInternalServerError, s.recorder.Code)
	s.Equal(`{"error":"internal server error"}`, s.recorder.Body.String())
}

func (s *BookmarksTestSuite) TestPutBookmarksWhenInValidBookmarks() {
	mockutil.MockJSONRequest(s.context, "PUT", nil, inValidBookmarksRequest)

	PutBookmarks(s.context)

	var invalidResponse models.InValidBookmarksResponse
	err := json.Unmarshal([]byte((s.recorder.Body.String())), &invalidResponse)

	s.NoError(err)
	s.EqualValues(http.StatusBadRequest, s.recorder.Code)
	s.EqualValues([]models.BookmarkEntry{
		{URL: "htps://www.run.ai/guides/gpu-deep-learning/best-gpu-for-deep-learning"},
		{URL: "https/www.simplilearn.com/keras-vs-tensorflow-vs-pytorch-article"}}, invalidResponse.BookmarkList)
}

func (s *BookmarksTestSuite) TestPutBookmarksWhenDistributionPending() {
	mockutil.MockJSONRequest(s.context, "PUT", nil, bookmarksRequest)

	distribution := &model.UserBookmarks{UserId: "1"}
	mockdist.Status = constant.Pending

	s.mockDynamoDBClient.EXPECT().GetRecordByKey(gomock.Eq(distribution)).Return(mockdist, nil)

	PutBookmarks(s.context)

	s.EqualValues(http.StatusForbidden, s.recorder.Code)
}

func addMocksForGetBookmarks(s *BookmarksTestSuite) {
	distribution := &model.UserBookmarks{UserId: "1"}
	s.mockDynamoDBClient.EXPECT().GetRecordByKey(gomock.Eq(distribution)).Return(mockdist, nil)

	s3Content := `{"bookmarks": [
			{ "url": "https://keda.sh/docs/2.13/deploy/" },
			{ "url": "https://botpenguin.com/glossary/transformer-architecture" },
			{ "url": "https://www.langchain.com/" },
			{ "url": "https://www.analyticsvidhya.com/blog/2022/11/top-6-interview-questions-on-transformer" },
			{ "url": "https://jalammar.github.io/illustrated-transformer/" },
			{ "url": "https://zapier.com/blog/claude-ai/" },
			{ "url": "https://karpenter.sh/" },
			{ "url": "https://github.com/openxla/xla" },
			{ "url": "https://deepgram.com/learn/visualizing-and-explaining-transformer-models-from-the-ground-up" }
		]}`

	s.mockS3Client.EXPECT().GetObject(gomock.Eq("test_bucket"),
		gomock.Eq("Bookmarks/1/1.0.89")).Return([]byte(s3Content), nil)
}
