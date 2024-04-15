package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/pranav-patil/go-serverless-api/func/api/middleware"
	"github.com/pranav-patil/go-serverless-api/func/api/models"
	pkgDynamoDB "github.com/pranav-patil/go-serverless-api/pkg/dynamodb"
	dynamoMocks "github.com/pranav-patil/go-serverless-api/pkg/dynamodb/mocks"
	"github.com/pranav-patil/go-serverless-api/pkg/dynamodb/model"
	"github.com/pranav-patil/go-serverless-api/pkg/mockutil"
	pkgS3 "github.com/pranav-patil/go-serverless-api/pkg/s3"

	"github.com/pranav-patil/go-serverless-api/pkg/s3/mocks"
	"github.com/stretchr/testify/suite"
)

type BookmarksSearchTestSuite struct {
	suite.Suite
	recorder *httptest.ResponseRecorder
	context  *gin.Context
	ctrl     *gomock.Controller

	mockS3Client       *mocks.MockS3Client
	mockDynamoDBClient *dynamoMocks.MockDynamoDBClient
}

func TestBookmarksSearchTestSuite(t *testing.T) {
	suite.Run(t, new(BookmarksSearchTestSuite))
}

func (s *BookmarksSearchTestSuite) SetupSuite() {
	s.T().Setenv("BOOKMARKS_BUCKET", "test_bucket")
	s.ctrl = gomock.NewController(s.T())
}

func (s *BookmarksSearchTestSuite) SetupTest() {
	s.recorder = httptest.NewRecorder()
	s.context = mockutil.MockGinContext(s.recorder)
	s.context.Set(middleware.UserIDCxt, "1")

	s.mockDynamoDBClient = dynamoMocks.NewMockDynamoDBClient(s.ctrl)
	NewDynamoDBClient = func() (pkgDynamoDB.DynamoDBClient, error) {
		return s.mockDynamoDBClient, nil
	}
	s.mockS3Client = mocks.NewMockS3Client(s.ctrl)
	NewS3Client = func() (pkgS3.S3Client, error) {
		return s.mockS3Client, nil
	}
}

func (s *BookmarksSearchTestSuite) TestFindBookmarkEntryWithURLMatched() {
	pathParams := []gin.Param{
		{Key: "url", Value: "https://docs.ai21.com/docs/jurassic-2-models"},
	}
	mockutil.MockJSONRequest(s.context, "HEAD", pathParams, nil)

	s3Content := `{"bookmarks": [
		{ "url": "https://docs.ai21.com/docs/jurassic-2-models" },
		{ "url": "https://jalammar.github.io/illustrated-transformer/" },
		{ "url": "https://chat.openai.com" }
	]}`

	assertGetS3Helper(s.mockDynamoDBClient, s.mockS3Client, s3Content)

	FindBookmarkEntry(s.context)

	s.EqualValues(http.StatusNoContent, s.context.Writer.Status())
}

func (s *BookmarksSearchTestSuite) TestFindBookmarkEntryWithURLNotMatched() {
	pathParams := []gin.Param{
		{Key: "url", Value: "https://docs.ai21.com/docs/jurassic-2-models"},
	}
	mockutil.MockJSONRequest(s.context, "HEAD", pathParams, nil)

	s3Content := `{"bookmarks": [
		{ "url": "https://arxiv.org/abs/1706.03762" },
		{ "url": "https://jalammar.github.io/illustrated-transformer/" },
		{ "url": "https://chat.openai.com" }
	]}`

	assertGetS3Helper(s.mockDynamoDBClient, s.mockS3Client, s3Content)

	FindBookmarkEntry(s.context)
	s.EqualValues(http.StatusNotFound, s.recorder.Code)
	s.Equal(`{"error":"no entry found for the input bookmark url"}`, s.recorder.Body.String())
}

func (s *BookmarksSearchTestSuite) TestFindBookmarkEntryWithEmptyEntryList() {
	pathParams := []gin.Param{
		{Key: "url", Value: "https://arxiv.org/abs/1706.03762"},
	}
	mockutil.MockJSONRequest(s.context, "HEAD", pathParams, nil)

	s3Content := `{"bookmarks": []}`

	assertGetS3Helper(s.mockDynamoDBClient, s.mockS3Client, s3Content)

	FindBookmarkEntry(s.context)

	s.EqualValues(http.StatusNotFound, s.context.Writer.Status())
}

func (s *BookmarksSearchTestSuite) TestDeleteMatchingBookmarksWithSingleURLMatch() {
	testURL := "https://lmsys.org/blog/2023-03-30-vicuna/"
	testEntryList := models.BookmarkList{
		BookmarkEntry: []models.BookmarkEntry{
			{URL: "https://lmsys.org/blog/2023-03-30-vicuna/"},
			{URL: "https://huggingface.co/spaces/HuggingFaceH4/open_llm_leaderboard"},
			{URL: "https://chat.openai.com"},
		},
	}
	expected := models.BookmarkList{
		BookmarkEntry: []models.BookmarkEntry{
			{URL: "https://huggingface.co/spaces/HuggingFaceH4/open_llm_leaderboard"},
			{URL: "https://chat.openai.com"},
		},
	}

	deleteMatchingBookmarks(testURL, &testEntryList)

	s.EqualValues(expected, testEntryList)
}

func (s *BookmarksSearchTestSuite) TestDeleteMatchingBookmarksWithMultipleMatches() {
	testURL := "https://docs.ai21.com/docs/jurassic-2-models"
	testEntryList := models.BookmarkList{
		BookmarkEntry: []models.BookmarkEntry{
			{URL: "https://docs.ai21.com/docs/jurassic-2-models"},
			{URL: "https://docs.ai21.com/docs/jurassic-2-models"},
			{URL: "https://chat.openai.com"},
		},
	}
	expected := models.BookmarkList{
		BookmarkEntry: []models.BookmarkEntry{
			{URL: "https://chat.openai.com"},
		},
	}

	deleteMatchingBookmarks(testURL, &testEntryList)

	s.EqualValues(expected, testEntryList)
}

func (s *BookmarksSearchTestSuite) TestDeleteMatchingBookmarksWithURLNotMatched() {
	testURL := "https://www.maxmind.com/en/home"
	testEntryList := models.BookmarkList{
		BookmarkEntry: []models.BookmarkEntry{
			{URL: "https://docs.ai21.com/docs/jurassic-2-models"},
			{URL: "https://chat.openai.com"},
		},
	}
	expected := models.BookmarkList{
		BookmarkEntry: []models.BookmarkEntry{
			{URL: "https://docs.ai21.com/docs/jurassic-2-models"},
			{URL: "https://chat.openai.com"},
		},
	}

	deleteMatchingBookmarks(testURL, &testEntryList)

	s.EqualValues(expected, testEntryList)
}

func (s *BookmarksSearchTestSuite) TestDeleteMatchingBookmarksWithEmptyEntryList() {
	testURL := "https://www.maxmind.com/en/home"
	testEntryList := models.BookmarkList{}
	testEntryList.BookmarkEntry = make([]models.BookmarkEntry, 0) // an empty entry list uploaded by user

	expected := models.BookmarkList{}
	expected.BookmarkEntry = make([]models.BookmarkEntry, 0)

	deleteMatchingBookmarks(testURL, &testEntryList)

	s.EqualValues(expected, testEntryList)
}

func (s *BookmarksSearchTestSuite) TestDeleteMatchingBookmarksWithConsecutiveDeletes() {
	testEntryList := models.BookmarkList{
		BookmarkEntry: []models.BookmarkEntry{
			{URL: "https://docs.ai21.com/docs/jurassic-2-models"},
			{URL: "https://huggingface.co/spaces/HuggingFaceH4/open_llm_leaderboard"},
			{URL: "https://huggingface.co/tiiuae/falcon-180B"},
			{URL: "https://chat.openai.com"},
		},
	}
	expected := models.BookmarkList{
		BookmarkEntry: []models.BookmarkEntry{
			{URL: "https://chat.openai.com"},
		},
	}

	deleteMatchingBookmarks("https://huggingface.co/tiiuae/falcon-180B", &testEntryList)
	deleteMatchingBookmarks("https://huggingface.co/spaces/HuggingFaceH4/open_llm_leaderboard", &testEntryList)
	deleteMatchingBookmarks("https://docs.ai21.com/docs/jurassic-2-models", &testEntryList)

	s.EqualValues(expected, testEntryList)
}

func (s *BookmarksSearchTestSuite) TestFindAndDeleteBookmarkEntryWithURLMatched() {
	pathParams := []gin.Param{
		{Key: "url", Value: "https://bigscience.huggingface.co/blog/bloom"},
	}
	mockutil.MockJSONRequest(s.context, "DELETE", pathParams, nil)

	s3Content := `{"bookmarks": [
		{ "url": "https://bigscience.huggingface.co/blog/bloom" },
		{ "url": "https://jalammar.github.io/illustrated-transformer/" },
		{ "url": "https://chat.openai.com" }
	]}`

	assertGetS3Helper(s.mockDynamoDBClient, s.mockS3Client, s3Content)

	updatedContent := []byte(
		`{"bookmarks":[{"url":"https://jalammar.github.io/illustrated-transformer/"},{"url":"https://chat.openai.com"}]}`)

	s.mockS3Client.EXPECT().
		PutObject("test_bucket", "Bookmarks/1/1.0.90", "application/json", pkgS3.GZip, &updatedContent)

	s.mockDynamoDBClient.EXPECT().UpdateRecordsByKey(mockutil.AnyOfType(&model.UserBookmarks{})).Return(nil)

	s.mockS3Client.EXPECT().DeleteObject(gomock.Eq("test_bucket"), gomock.Eq("Bookmarks/1/1.0.89")).Return(nil)

	FindAndDeleteBookmarkEntry(s.context)

	s.EqualValues(http.StatusNoContent, s.context.Writer.Status())
}

func (s *BookmarksSearchTestSuite) TestFindAndDeleteBookmarkEntryWithIPv4NotMatched() {
	pathParams := []gin.Param{
		{Key: "url", Value: "https://www.maxmind.com/en/home"},
	}
	mockutil.MockJSONRequest(s.context, "DELETE", pathParams, nil)

	s3Content := `{"bookmarks": [
		{ "url": "https://falconllm.tii.ae/falcon.html" },
		{ "url": "https://chat.openai.com" }
	]}`

	assertGetS3Helper(s.mockDynamoDBClient, s.mockS3Client, s3Content)

	updatedContent := []byte(
		`{"bookmarks":[{"url":"https://falconllm.tii.ae/falcon.html"},{"url":"https://chat.openai.com"}]}`)

	s.mockS3Client.EXPECT().
		PutObject(
			gomock.Eq("test_bucket"),
			gomock.Eq("Bookmarks/1/1.0.90"),
			gomock.Eq("application/json"),
			gomock.Eq(pkgS3.GZip),
			gomock.Eq(&updatedContent))

	s.mockDynamoDBClient.EXPECT().UpdateRecordsByKey(mockutil.AnyOfType(&model.UserBookmarks{})).Return(nil)

	s.mockS3Client.EXPECT().DeleteObject(gomock.Eq("test_bucket"), gomock.Eq("Bookmarks/1/1.0.89")).Return(nil)

	FindAndDeleteBookmarkEntry(s.context)

	s.EqualValues(http.StatusNoContent, s.context.Writer.Status())
}

func (s *BookmarksSearchTestSuite) TestFindAndDeleteBookmarkEntryWithInescapableURL() {
	pathParams := []gin.Param{
		{Key: "url", Value: "https://arxiv.org/abs%%2F1706.03762"}, // covering but not covered by "https://docs.chef.io/resources/script/"
	}
	mockutil.MockJSONRequest(s.context, "DELETE", pathParams, nil)

	FindAndDeleteBookmarkEntry(s.context)

	s.EqualValues(http.StatusBadRequest, s.recorder.Code)
	s.Equal(`{"error":"invalid URL escape \"%%2\""}`, s.recorder.Body.String())
}

func (s *BookmarksSearchTestSuite) TestFindAndDeleteBookmarkEntryWithEmptyEntryList() {
	pathParams := []gin.Param{
		{Key: "url", Value: "https://bigscience.huggingface.co/blog/bloom"},
	}
	mockutil.MockJSONRequest(s.context, "DELETE", pathParams, nil)

	s3Content := `{"bookmarks": []}`

	assertGetS3Helper(s.mockDynamoDBClient, s.mockS3Client, s3Content)

	updatedContent := []byte(`{"bookmarks":[]}`)

	s.mockS3Client.EXPECT().
		PutObject(
			gomock.Eq("test_bucket"),
			gomock.Eq("Bookmarks/1/1.0.90"),
			gomock.Eq("application/json"),
			gomock.Eq("gzip"),
			gomock.Eq(&updatedContent))

	s.mockDynamoDBClient.EXPECT().UpdateRecordsByKey(mockutil.AnyOfType(&model.UserBookmarks{})).Return(nil)

	s.mockS3Client.EXPECT().DeleteObject(gomock.Eq("test_bucket"), gomock.Eq("Bookmarks/1/1.0.89")).Return(nil)

	FindAndDeleteBookmarkEntry(s.context)

	s.EqualValues(http.StatusNoContent, s.context.Writer.Status())
}

func assertGetS3Helper(mockDynamoDBClient *dynamoMocks.MockDynamoDBClient, mockS3Client *mocks.MockS3Client,
	s3Content string) {
	distribution := &model.UserBookmarks{UserId: "1"}
	mockdist.LatestVersion = "1.0.89"
	mockDynamoDBClient.EXPECT().GetRecordByKey(gomock.Eq(distribution)).Return(mockdist, nil)

	mockS3Client.EXPECT().
		GetObject(
			gomock.Eq("test_bucket"),
			gomock.Eq("Bookmarks/1/1.0.89")).
		Return([]byte(s3Content), nil)
}
