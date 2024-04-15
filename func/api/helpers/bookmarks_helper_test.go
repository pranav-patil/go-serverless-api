package helpers

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pranav-patil/go-serverless-api/func/api/models"
	"github.com/pranav-patil/go-serverless-api/pkg/constant"
	dynamoMocks "github.com/pranav-patil/go-serverless-api/pkg/dynamodb/mocks"
	"github.com/pranav-patil/go-serverless-api/pkg/dynamodb/model"
	"github.com/pranav-patil/go-serverless-api/pkg/mockutil"
	"github.com/pranav-patil/go-serverless-api/pkg/s3"
	s3Mocks "github.com/pranav-patil/go-serverless-api/pkg/s3/mocks"
	"github.com/stretchr/testify/suite"
)

type BookmarksHelperTestSuite struct {
	suite.Suite
	ctrl               *gomock.Controller
	mockS3Client       *s3Mocks.MockS3Client
	mockDynamoDBClient *dynamoMocks.MockDynamoDBClient
	mockTimeNow        time.Time
}

var mockDistribution = model.UserBookmarks{
	Status:         constant.Pending,
	StartTimestamp: time.Now(),
	EndTimestamp:   time.Now(),
	UserId:         "1",
	LatestVersion:  "1.0.96",
}

func TestBookmarksHelperSuite(t *testing.T) {
	suite.Run(t, new(BookmarksHelperTestSuite))
}

func (s *BookmarksHelperTestSuite) SetupSuite() {
	s.ctrl = gomock.NewController(s.T())
	s.T().Setenv("BOOKMARKS_BUCKET", "TEST_S3_BUCKET")
}

func (s *BookmarksHelperTestSuite) SetupTest() {
	s.mockDynamoDBClient = dynamoMocks.NewMockDynamoDBClient(s.ctrl)
	s.mockS3Client = s3Mocks.NewMockS3Client(s.ctrl)

	s.mockTimeNow = time.Date(2009, time.November, 10, 23, 52, 34, 9, time.UTC)
	TimeNow = func() time.Time {
		return s.mockTimeNow
	}
}

func (s *BookmarksHelperTestSuite) TestGetBookmarksS3Object() {
	distribution := &model.UserBookmarks{UserId: "1"}
	mockDistribution.LatestVersion = "1.0.96"
	s.mockDynamoDBClient.EXPECT().GetRecordByKey(gomock.Eq(distribution)).Return(&mockDistribution, nil)

	s3Content := `{"bookmarks": [
			{ "url": "https://docs.ai21.com/docs/jurassic-2-models" },
			{ "url": "https://jalammar.github.io/illustrated-transformer/" }
		]}`

	s.mockS3Client.EXPECT().GetObject(gomock.Eq("TEST_S3_BUCKET"),
		gomock.Eq("Bookmarks/1/1.0.96")).Return([]byte(s3Content), nil)

	bookmarks, err := GetBookmarksS3Object(s.mockDynamoDBClient, s.mockS3Client, distribution.UserId)

	s.Nil(err)
	s.NotNil(bookmarks)
	s.Equal(s3Content, string(bookmarks))
}

func (s *BookmarksHelperTestSuite) TestGetBookmarksLatestFileName() {
	distribution := &model.UserBookmarks{UserId: "45", LatestVersion: "1.0.67"}

	version := GetUserBookmarksS3Path(distribution)

	s.Equal(version, "Bookmarks/45/1.0.67")
}

func (s *BookmarksHelperTestSuite) TestGetBookmarksLatestFileNameWhenDefault() {
	version := GetUserBookmarksS3Path(nil)
	s.Equal(version, "")
}

func (s *BookmarksHelperTestSuite) TestGetBookmarksLatestFileNameWhenDeleted() {
	distribution := &model.UserBookmarks{UserId: "45", LatestVersion: "1.0.78_DELETED"}

	version := GetUserBookmarksS3Path(distribution)
	s.Equal(version, "")
}

func (s *BookmarksHelperTestSuite) TestIsDistributionPendingByTenantWhenStatusSuccess() {
	mockDistribution.Status = constant.Success
	isPending := IsDistributionPending(&mockDistribution)
	s.Equal(false, isPending)
}

func (s *BookmarksHelperTestSuite) TestIsDistributionPendingWhenNil() {
	isPending := IsDistributionPending(nil)
	s.Equal(false, isPending)
}

func (s *BookmarksHelperTestSuite) TestGetDistributionByTenantWhenStatusPending() {
	userId := "1"
	mockDistrib := &model.UserBookmarks{UserId: userId, Status: constant.Pending, LatestVersion: "1.0.96"}
	mockDistrib.StartTimestamp = time.Date(2009, time.November, 10, 23, 49, 32, 9, time.UTC)

	s.mockDynamoDBClient.EXPECT().GetRecordByKey(gomock.Eq(&model.UserBookmarks{UserId: "1"})).Return(mockDistrib, nil)

	resultDistrib := GetBookmarkByUser(s.mockDynamoDBClient, userId)
	s.Equal(constant.Pending, resultDistrib.Status)
}

func (s *BookmarksHelperTestSuite) TestGetDistributionByTenantWhenStatusPendingAndStartTimeDelayed() {
	userId := "1"
	mockDistrib := &model.UserBookmarks{UserId: userId, Status: constant.Pending, LatestVersion: "1.0.96"}
	mockDistrib.StartTimestamp = time.Date(2009, time.November, 10, 23, 1, 10, 3, time.UTC)

	s.mockDynamoDBClient.EXPECT().GetRecordByKey(gomock.Eq(&model.UserBookmarks{UserId: "1"})).Return(mockDistrib, nil)

	resultDistrib := GetBookmarkByUser(s.mockDynamoDBClient, userId)
	s.Equal(constant.Timeout, resultDistrib.Status)
}

func (s *BookmarksHelperTestSuite) TestAddOrUpdateDistributionWhenDistributionAdded() {
	userId := "1"
	distribution := model.UserBookmarks{UserId: userId}

	var capturedArgs []model.Entity

	s.mockDynamoDBClient.EXPECT().AddRecord(mockutil.AnyOfType(distribution)).DoAndReturn(func(p model.Entity) error {
		capturedArgs = append(capturedArgs, p)
		return nil
	})

	err := AddOrUpdateUserBookmarks(s.mockDynamoDBClient, nil, userId, "1.0.93", true)
	s.Nil(err)
	s.Equal(1, len(capturedArgs))

	actualDist := capturedArgs[0].(*model.UserBookmarks)
	s.Equal("1", actualDist.UserId)
	s.Equal("1.0.93", actualDist.LatestVersion)
	s.Equal(true, actualDist.ModifiedBookmarks)
}

func (s *BookmarksHelperTestSuite) TestAddOrUpdateDistributionWhenDistributionUpdated() {
	userId := "1"
	distribution := model.UserBookmarks{UserId: userId}

	var capturedArgs []model.Entity

	s.mockDynamoDBClient.EXPECT().UpdateRecordsByKey(mockutil.AnyOfType(distribution)).DoAndReturn(func(p model.Entity) error {
		capturedArgs = append(capturedArgs, p)
		return nil
	})

	distribution = model.UserBookmarks{
		UserId: userId,
	}
	err := AddOrUpdateUserBookmarks(s.mockDynamoDBClient, &distribution, userId, "1.0.78", true)
	s.Nil(err)
	s.Equal(1, len(capturedArgs))

	actualDist := capturedArgs[0].(*model.UserBookmarks)
	s.Equal("1", actualDist.UserId)
	s.Equal("1.0.78", actualDist.LatestVersion)
	s.Equal(true, actualDist.ModifiedBookmarks)
}

func (s *BookmarksHelperTestSuite) TestConvertCSVAndValidateBookmarks() {
	testCases := []struct {
		testName                 string
		inputContents            string
		expectedValidBookmarks   []models.BookmarkEntry
		expectedInvalidBookmarks []models.BookmarkEntry
	}{
		{"All Valid Bookmark Bookmarks",
			`https://docs.ai21.com/docs/jurassic-2-models,
			https://jalammar.github.io/illustrated-transformer/,
			https://karpenter.sh/,
			https://github.com/openxla/xla`,
			[]models.BookmarkEntry{
				{URL: "https://docs.ai21.com/docs/jurassic-2-models"},
				{URL: "https://jalammar.github.io/illustrated-transformer/"},
				{URL: "https://karpenter.sh/"},
				{URL: "https://github.com/openxla/xla"}},
			[]models.BookmarkEntry{}},
		{"Some Valid and some invalid Bookmark Bookmarks",
			`komodor.com/learn/how-to-fix-crashloopbackoff-kubernetes-error/,
			https://jalammar.github.io/illustrated-transformer/,
			https://karpenter.sh/,
			httpswww.langchain.com/`,
			[]models.BookmarkEntry{{URL: "https://jalammar.github.io/illustrated-transformer/"},
				{URL: "https://karpenter.sh/"}},
			[]models.BookmarkEntry{{URL: "komodor.com/learn/how-to-fix-crashloopbackoff-kubernetes-error/"},
				{URL: "httpswww.langchain.com/"}}},
		{"All Invalid Bookmark Bookmarks",
			`htt//emprovisetech.blogspot.com/,
			httpswww.langchain.com/`,
			[]models.BookmarkEntry{},
			[]models.BookmarkEntry{{URL: "htt//emprovisetech.blogspot.com/"}, {URL: "httpswww.langchain.com/"}}},
	}

	for _, testCase := range testCases {
		s.Run(testCase.testName, func() {
			s.T().Log(testCase.inputContents)
			valid, invalid, _ := ConvertCSVAndValidateBookmarks(testCase.inputContents)
			s.EqualValues(testCase.expectedValidBookmarks, valid)
			s.EqualValues(testCase.expectedInvalidBookmarks, invalid)
		})
	}
}

func (s *BookmarksHelperTestSuite) TestValidateBookmarks() {
	testCases := []struct {
		testName                 string
		inputBookmarks           []models.BookmarkEntry
		expectedValidBookmarks   []models.BookmarkEntry
		expectedInvalidBookmarks []models.BookmarkEntry
	}{
		{"All Valid Bookmark Bookmarks",
			[]models.BookmarkEntry{
				{URL: "https://docs.ai21.com/docs/jurassic-2-models"},
				{URL: "https://jalammar.github.io/illustrated-transformer/"},
				{URL: "https://akgeni.medium.com/llama-concepts-explained-summary-a87f0bd61964"},
				{URL: "https://github.com/openxla/xla"}},

			[]models.BookmarkEntry{
				{URL: "https://docs.ai21.com/docs/jurassic-2-models"},
				{URL: "https://jalammar.github.io/illustrated-transformer/"},
				{URL: "https://akgeni.medium.com/llama-concepts-explained-summary-a87f0bd61964"},
				{URL: "https://github.com/openxla/xla"}},
			[]models.BookmarkEntry{}},

		{"Some Valid and some invalid Bookmark Bookmarks",
			[]models.BookmarkEntry{
				{URL: "komodor.com/learn/how-to-fix-crashloopbackoff-kubernetes-error/"},
				{URL: "https://jalammar.github.io/illustrated-transformer/"},
				{URL: "https://karpenter.sh/"},
				{URL: "httpswww.langchain.com/"}},

			[]models.BookmarkEntry{{URL: "https://jalammar.github.io/illustrated-transformer/"},
				{URL: "https://karpenter.sh/"}},
			[]models.BookmarkEntry{{URL: "komodor.com/learn/how-to-fix-crashloopbackoff-kubernetes-error/"},
				{URL: "httpswww.langchain.com/"}}},

		{"All Invalid Bookmark Bookmarks",
			[]models.BookmarkEntry{{URL: "htt//emprovisetech.blogspot.com/"}, {URL: "httpswww.langchain.com/"}},
			[]models.BookmarkEntry{},
			[]models.BookmarkEntry{{URL: "htt//emprovisetech.blogspot.com/"}, {URL: "httpswww.langchain.com/"}}},
	}

	for _, testCase := range testCases {
		s.Run(testCase.testName, func() {
			s.T().Log(testCase.inputBookmarks)
			valid, invalid := ValidateBookmarks(testCase.inputBookmarks)
			s.EqualValues(testCase.expectedValidBookmarks, valid)
			s.EqualValues(testCase.expectedInvalidBookmarks, invalid)
		})
	}
}

func (s *BookmarksHelperTestSuite) TestGetIncrementedVersion() {
	distribution := &model.UserBookmarks{LatestVersion: "1.0.67"}

	distVersion, err := GetIncrementedVersion(distribution)
	s.Nil(err)
	s.Equal("1.0.68", distVersion)
}

func (s *BookmarksHelperTestSuite) TestGetIncrementedVersionWhenDistributionNil() {
	distVersion, err := GetIncrementedVersion(nil)
	s.Nil(err)
	s.Equal("1.0.1", distVersion)
}

func (s *BookmarksHelperTestSuite) TestGetIncrementedVersionWhenDistVersionIsDeleted() {
	distribution := &model.UserBookmarks{LatestVersion: "1.0.67_DELETED"}

	distVersion, err := GetIncrementedVersion(distribution)
	s.Nil(err)
	s.Equal("1.0.68", distVersion)
}

func (s *BookmarksHelperTestSuite) TestAddBookmarksInS3Bucket() {
	distribution := &model.UserBookmarks{UserId: "1", LatestVersion: "1.0.96"}

	s.mockDynamoDBClient.EXPECT().UpdateRecordsByKey(mockutil.AnyOfType(distribution)).Return(nil)

	s3Content := []byte(
		`{"bookmarks":[{"url":"https://jalammar.github.io/illustrated-transformer/"},{"url":"https://chat.openai.com"}]}`)

	s.mockS3Client.EXPECT().PutObject(gomock.Eq("TEST_S3_BUCKET"),
		gomock.Eq("Bookmarks/1/1.0.97"), gomock.Eq("application/json"), gomock.Eq(s3.GZip),
		&s3Content).Return(nil)

	s.mockS3Client.EXPECT().DeleteObject(gomock.Eq("TEST_S3_BUCKET"),
		gomock.Eq("Bookmarks/1/1.0.96")).Return(nil)

	err := AddBookmarksInS3Bucket(s.mockDynamoDBClient, s.mockS3Client,
		distribution, "1", "TEST_S3_BUCKET", "application/json", s3.GZip, &s3Content)

	s.Nil(err)
}

func (s *BookmarksHelperTestSuite) TestAddBookmarksInS3BucketWhenPreviousDistributionIsNil() {
	distribution := &model.UserBookmarks{UserId: "1", LatestVersion: "1.0.1"}

	s.mockDynamoDBClient.EXPECT().AddRecord(mockutil.AnyOfType(distribution)).Return(nil)

	s3Content := []byte(`{"bookmarks":[{"url":"https://jalammar.github.io/illustrated-transformer/"}]}`)

	s.mockS3Client.EXPECT().PutObject(gomock.Eq("TEST_S3_BUCKET"),
		gomock.Eq("Bookmarks/1/1.0.1"), gomock.Eq("application/json"), gomock.Eq(s3.GZip),
		&s3Content).Return(nil)

	err := AddBookmarksInS3Bucket(s.mockDynamoDBClient, s.mockS3Client,
		nil, "1", "TEST_S3_BUCKET", "application/json", s3.GZip, &s3Content)

	s.Nil(err)
}
