package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/pranav-patil/go-serverless-api/func/api/helpers"
	"github.com/pranav-patil/go-serverless-api/func/api/middleware"
	"github.com/pranav-patil/go-serverless-api/func/api/models"
	"github.com/pranav-patil/go-serverless-api/pkg/constant"
	pkgDynamoDB "github.com/pranav-patil/go-serverless-api/pkg/dynamodb"
	dynamoMocks "github.com/pranav-patil/go-serverless-api/pkg/dynamodb/mocks"
	"github.com/pranav-patil/go-serverless-api/pkg/dynamodb/model"
	"github.com/pranav-patil/go-serverless-api/pkg/mockutil"
	pkgS3 "github.com/pranav-patil/go-serverless-api/pkg/s3"
	s3Mocks "github.com/pranav-patil/go-serverless-api/pkg/s3/mocks"
	apiService "github.com/pranav-patil/go-serverless-api/pkg/service"
	apiMocks "github.com/pranav-patil/go-serverless-api/pkg/service/mocks"
	pkgStepFunc "github.com/pranav-patil/go-serverless-api/pkg/stepfunc"
	stepFuncMocks "github.com/pranav-patil/go-serverless-api/pkg/stepfunc/mocks"
	"github.com/stretchr/testify/suite"
)

type DistributeBookmarksTestSuite struct {
	suite.Suite

	ctrl               *gomock.Controller
	recorder           *httptest.ResponseRecorder
	context            *gin.Context
	mockS3Client       *s3Mocks.MockS3Client
	mockDynamoDBClient *dynamoMocks.MockDynamoDBClient
	mockStepFuncClient *stepFuncMocks.MockStepFuncClient
	mockTimeNow        time.Time
}

func TestDistributeBookmarksSuite(t *testing.T) {
	suite.Run(t, new(DistributeBookmarksTestSuite))
}

func (s *DistributeBookmarksTestSuite) SetupSuite() {
	s.T().Setenv("BOOKMARKS_BUCKET", "test_bookmarks_bucket")
	s.T().Setenv("BOOKMARKS_SUMMARY_BUCKET", "test_package_bucket")
	s.T().Setenv("DISTRIBUTION_STATE_MACHINE_ARN", "test_distribution_state_machine_arn")
	s.ctrl = gomock.NewController(s.T())
}

func (s *DistributeBookmarksTestSuite) SetupTest() {
	s.recorder = httptest.NewRecorder()
	s.context = mockutil.MockGinContext(s.recorder)
	s.context.Set(middleware.UserIDCxt, "1")
	s.context.Set(middleware.JWTToken, "mock-jwt-token")

	s.mockS3Client = s3Mocks.NewMockS3Client(s.ctrl)
	NewS3Client = func() (pkgS3.S3Client, error) {
		return s.mockS3Client, nil
	}

	s.mockDynamoDBClient = dynamoMocks.NewMockDynamoDBClient(s.ctrl)
	NewDynamoDBClient = func() (pkgDynamoDB.DynamoDBClient, error) {
		return s.mockDynamoDBClient, nil
	}

	s.mockStepFuncClient = stepFuncMocks.NewMockStepFuncClient(s.ctrl)
	NewStepFunctionClient = func() (pkgStepFunc.StepFuncClient, error) {
		return s.mockStepFuncClient, nil
	}

	s.mockTimeNow = time.Date(2009, time.November, 10, 23, 52, 34, 9, time.UTC)
	TimeNow = func() time.Time {
		return s.mockTimeNow
	}
	helpers.TimeNow = func() time.Time {
		return s.mockTimeNow
	}
}

func (s *DistributeBookmarksTestSuite) TestDistributeBookmarksWithInValidAppIds() {
	distributeRequest := models.DistributeBookmarksRequest{
		DeviceIDs: []int{24390168, 46747567, 67479298, 67787448},
	}

	mockutil.MockJSONRequest(s.context, "POST", nil, distributeRequest)

	distribution := &model.UserBookmarks{UserId: "1"}
	s.mockDynamoDBClient.EXPECT().GetRecordByKey(gomock.Eq(distribution)).Return(mockdist, nil)

	mockExtServiceAPI := apiMocks.NewMockExternalServiceAPI(s.ctrl)
	NewExtServiceAPI = func() (apiService.ExternalServiceAPI, error) {
		return mockExtServiceAPI, nil
	}

	deviceResponse := &apiService.DeviceListResponse{
		Devices: []apiService.Device{
			{Id: 46747567, InstanceId: "35546"},
			{Id: 54656674, InstanceId: "57683"},
			{Id: 67787448, InstanceId: "23678"},
			{Id: 98988343, InstanceId: "10457"},
		},
	}
	mockExtServiceAPI.EXPECT().GetUserDevices(gomock.Eq("mock-jwt-token"),
		gomock.Eq("1")).Return(deviceResponse, nil)

	DistributeBookmarks(s.context)

	s.EqualValues(http.StatusBadRequest, s.recorder.Code)

	var invalidResponse models.InvalidDistributeBookmarksResponse
	responseJSON := s.recorder.Body.String()
	err := json.Unmarshal([]byte(responseJSON), &invalidResponse)

	s.NoError(err)
	s.Equal(2, len(invalidResponse.DeviceIDs))
	s.EqualValues(invalidResponse.DeviceIDs, []int{24390168, 67479298})
}

func (s *DistributeBookmarksTestSuite) TestDistributeBookmarksWhenExtGetUserDevicesEmpty() {
	distributeRequest := models.DistributeBookmarksRequest{
		DeviceIDs: []int{56765767, 234234},
	}

	mockutil.MockJSONRequest(s.context, "POST", nil, distributeRequest)

	distribution := &model.UserBookmarks{UserId: "1"}
	s.mockDynamoDBClient.EXPECT().GetRecordByKey(gomock.Eq(distribution)).Return(mockdist, nil)

	mockExtServiceAPI := apiMocks.NewMockExternalServiceAPI(s.ctrl)
	NewExtServiceAPI = func() (apiService.ExternalServiceAPI, error) {
		return mockExtServiceAPI, nil
	}

	mockExtServiceAPI.EXPECT().GetUserDevices(gomock.Eq("mock-jwt-token"),
		gomock.Eq("1")).Return(&apiService.DeviceListResponse{}, nil)

	DistributeBookmarks(s.context)

	s.EqualValues(http.StatusBadRequest, s.recorder.Code)
	s.Equal(`{"error":"No devices found for the user"}`, s.recorder.Body.String())
}

func (s *DistributeBookmarksTestSuite) TestDistributeBookmarksWithValidAppIds() {
	distributeRequest := models.DistributeBookmarksRequest{
		DeviceIDs: []int{46747567, 67787448},
	}

	mockutil.MockJSONRequest(s.context, "POST", nil, distributeRequest)

	mockExtServiceAPI := apiMocks.NewMockExternalServiceAPI(s.ctrl)
	NewExtServiceAPI = func() (apiService.ExternalServiceAPI, error) {
		return mockExtServiceAPI, nil
	}

	deviceResponse := &apiService.DeviceListResponse{
		Devices: []apiService.Device{
			{Id: 46747567, InstanceId: "35546"},
			{Id: 54656674, InstanceId: "57683"},
			{Id: 67787448, InstanceId: "23678"},
			{Id: 98988343, InstanceId: "10457"},
		},
	}
	mockExtServiceAPI.EXPECT().GetUserDevices(gomock.Eq("mock-jwt-token"),
		gomock.Eq("1")).Return(deviceResponse, nil)

	s3Content := `{"bookmarks": [{ "url": "172.12.0.101/32" }]}`

	s.mockS3Client.EXPECT().GetObject(gomock.Eq("test_bookmarks_bucket"),
		gomock.Eq("Bookmarks/1/1.0.89")).Return([]byte(s3Content), nil)

	s.mockS3Client.EXPECT().PutObject(gomock.Eq("test_package_bucket"),
		gomock.Eq("Bookmarks/c4ca4238a0b923820dcc509a6f75849b/1.0.89"), gomock.Eq(JSON), gomock.Eq("none"),
		gomock.Any()).Return(nil)

	s.mockS3Client.EXPECT().NewSignedGetURL(gomock.Eq("test_package_bucket"),
		gomock.Eq("Bookmarks/c4ca4238a0b923820dcc509a6f75849b/1.0.89"), gomock.Eq(int64(300))).Return("CHECKSUM", nil)

	distribution := &model.UserBookmarks{UserId: "1"}
	s.mockDynamoDBClient.EXPECT().GetRecordByKey(gomock.Eq(distribution)).Return(mockdist, nil)

	s.mockDynamoDBClient.EXPECT().UpdateRecordsByKey(mockutil.AnyOfType(distribution)).Return(nil).MaxTimes(2)

	appDistribution := &model.BookmarkDistribution{UserId: "1"}
	s.mockDynamoDBClient.EXPECT().AddRecord(mockutil.AnyOfType(appDistribution)).Return(nil).MaxTimes(2)

	s.mockStepFuncClient.EXPECT().StartExecution(gomock.Eq("test_distribution_state_machine_arn"),
		gomock.Any(), gomock.AssignableToTypeOf(DownloadBookmarksInput{})).Return(nil).MaxTimes(2)

	DistributeBookmarks(s.context)

	s.EqualValues(http.StatusOK, s.recorder.Code)

	var distributionResponse models.DistributedBookmarksResponse
	responseJSON := s.recorder.Body.String()
	err := json.Unmarshal([]byte(responseJSON), &distributionResponse)

	s.NoError(err)
	s.Equal(2, len(distributionResponse.DistributionJobList))
	s.Contains(distributionResponse.DistributionJobList,
		models.WebCrawlerJob{
			ID:             "20091110235234",
			DeviceId:       46747567,
			PackageVersion: "1.0.89",
			State:          constant.Pending,
			StatusMessage:  "Distribution Process Triggered",
			StartTime:      s.mockTimeNow,
			EndTime:        time.Time{},
		})
}

func (s *DistributeBookmarksTestSuite) TestDistributeBookmarksWhenBookmarksDeleted() {
	distributeRequest := models.DistributeBookmarksRequest{
		DeviceIDs: []int{},
	}

	mockutil.MockJSONRequest(s.context, "POST", nil, distributeRequest)

	mockExtServiceAPI := apiMocks.NewMockExternalServiceAPI(s.ctrl)
	NewExtServiceAPI = func() (apiService.ExternalServiceAPI, error) {
		return mockExtServiceAPI, nil
	}

	deviceResponse := &apiService.DeviceListResponse{
		Devices: []apiService.Device{
			{Id: 46747567, InstanceId: "35546"},
			{Id: 54656674, InstanceId: "57683"},
		},
	}
	mockExtServiceAPI.EXPECT().GetUserDevices(gomock.Eq("mock-jwt-token"),
		gomock.Eq("1")).Return(deviceResponse, nil)

	distribution := &model.UserBookmarks{UserId: "1"}
	mockDeletedDist := &model.UserBookmarks{
		Status:         constant.Pending,
		StartTimestamp: s.mockTimeNow.Add(-time.Minute * 20),
		EndTimestamp:   time.Time{},
		UserId:         "1",
		LatestVersion:  "1.0.89_DELETED",
	}
	s.mockDynamoDBClient.EXPECT().GetRecordByKey(gomock.Eq(distribution)).Return(mockDeletedDist, nil)

	s.mockDynamoDBClient.EXPECT().UpdateRecordsByKey(mockutil.AnyOfType(distribution)).Return(nil).MaxTimes(2)

	appDistribution := &model.BookmarkDistribution{UserId: "1"}
	s.mockDynamoDBClient.EXPECT().AddRecord(mockutil.AnyOfType(appDistribution)).Return(nil).MaxTimes(2)

	s.mockStepFuncClient.EXPECT().StartExecution(gomock.Eq("test_distribution_state_machine_arn"),
		gomock.Any(), gomock.AssignableToTypeOf(DownloadBookmarksInput{})).Return(nil).MaxTimes(2)

	DistributeBookmarks(s.context)

	s.EqualValues(http.StatusOK, s.recorder.Code)

	var distributionResponse models.DistributedBookmarksResponse
	responseJSON := s.recorder.Body.String()
	err := json.Unmarshal([]byte(responseJSON), &distributionResponse)

	s.NoError(err)
	s.Equal(2, len(distributionResponse.DistributionJobList))
	s.Contains(distributionResponse.DistributionJobList,
		models.WebCrawlerJob{
			ID:             "20091110235234",
			DeviceId:       46747567,
			PackageVersion: "1.0.89_DELETED",
			State:          constant.Pending,
			StatusMessage:  "Distribution Process Triggered",
			StartTime:      s.mockTimeNow,
			EndTime:        time.Time{},
		})
}

func (s *DistributeBookmarksTestSuite) TestDistributeBookmarksWhenDistributionPending() {
	distributeRequest := models.DistributeBookmarksRequest{
		DeviceIDs: []int{46747567, 67787448},
	}

	mockutil.MockJSONRequest(s.context, "POST", nil, distributeRequest)

	distPending := &model.UserBookmarks{
		Status:         constant.Pending,
		StartTimestamp: s.mockTimeNow.Add(-time.Second * 30),
		EndTimestamp:   time.Time{},
		UserId:         "1",
	}

	fmt.Println("StartTimestamp: ", distPending.StartTimestamp.Format("2006-01-02 15:04:05"))
	fmt.Println("EndTimestamp: ", distPending.EndTimestamp.Format("2006-01-02 15:04:05"))
	distribution := &model.UserBookmarks{UserId: "1"}
	s.mockDynamoDBClient.EXPECT().GetRecordByKey(gomock.Eq(distribution)).Return(distPending, nil)

	DistributeBookmarks(s.context)

	s.EqualValues(http.StatusForbidden, s.recorder.Code)
}

func (s *DistributeBookmarksTestSuite) TestGetBookmarksAndCreatePackage() {
	s3Content := `{"bookmarks": [{ "url": "172.12.0.101/32" }]}`

	s.mockS3Client.EXPECT().GetObject(gomock.Eq("test_bookmarks_bucket"),
		gomock.Eq("Bookmarks/1/1.0.89")).Return([]byte(s3Content), nil)

	s.mockS3Client.EXPECT().PutObject(gomock.Eq("test_package_bucket"),
		gomock.Eq("Bookmarks/1/1.0.89"), gomock.Eq(JSON), gomock.Eq("none"),
		gomock.Any()).Return(nil)

	_, err := getBookmarksAndCreatePackage(s.mockS3Client, "1.0.89", "Bookmarks/1/1.0.89", "Bookmarks/1/1.0.89")
	s.NoError(err)
}

func (s *DistributeBookmarksTestSuite) TestGetDistributeBookmarks() {
	mockutil.MockJSONRequest(s.context, "GET", nil, nil)
	mockStartTime := s.mockTimeNow.Add(-time.Duration(10) * 220 * time.Millisecond)

	distribution := &model.UserBookmarks{UserId: "1"}
	s.mockDynamoDBClient.EXPECT().GetRecordByKey(gomock.Eq(distribution)).Return(mockdist, nil)

	appDistributionSlice := []model.BookmarkDistribution{
		{
			Status:         constant.Success,
			StartTimestamp: mockStartTime,
			EndTimestamp:   s.mockTimeNow,
			UserId:         "1",
			DeviceId:       "24234436",
		},
		{
			Status:         constant.Pending,
			StartTimestamp: mockStartTime,
			EndTimestamp:   time.Time{},
			UserId:         "1",
			DeviceId:       "67787448",
		},
		{
			Status:         constant.Pending,
			StartTimestamp: mockStartTime,
			EndTimestamp:   time.Time{},
			UserId:         "1",
			DeviceId:       "190345342",
		},
	}

	deviceDistribution := &model.BookmarkDistribution{UserId: "1"}
	s.mockDynamoDBClient.EXPECT().GetRecordsByKeyAndFields(
		gomock.Eq(deviceDistribution)).Return(appDistributionSlice, nil)

	GetDistributedBookmarks(s.context)

	s.EqualValues(http.StatusOK, s.recorder.Code)

	var distResponse models.DistributedBookmarksResponse
	distResponseJSON := s.recorder.Body.String()
	err := json.Unmarshal([]byte(distResponseJSON), &distResponse)

	s.NoError(err)
	s.Equal(3, len(distResponse.DistributionJobList))
}

func (s *DistributeBookmarksTestSuite) TestGetDistributeBookmarksWhenDistributionEmpty() {
	mockutil.MockJSONRequest(s.context, "GET", nil, nil)

	distribution := &model.UserBookmarks{UserId: "1"}
	s.mockDynamoDBClient.EXPECT().GetRecordByKey(gomock.Eq(distribution)).Return(mockdist, nil)

	emptyDistribSlice := []model.BookmarkDistribution{}
	deviceDistribution := &model.BookmarkDistribution{UserId: "1"}
	s.mockDynamoDBClient.EXPECT().GetRecordsByKeyAndFields(
		gomock.Eq(deviceDistribution)).Return(emptyDistribSlice, nil)

	GetDistributedBookmarks(s.context)

	s.EqualValues(http.StatusNotFound, s.recorder.Code)
	s.Equal(`{"error":"no device distributions found"}`, s.recorder.Body.String())
}

func (s *DistributeBookmarksTestSuite) TestGetDistributeBookmarksWhenPendingIsDelayed() {
	mockutil.MockJSONRequest(s.context, "GET", nil, nil)
	mockStartTime := s.mockTimeNow.Add(-time.Duration(30) * time.Minute)

	appDistributionSlice := []model.BookmarkDistribution{
		{
			Status:         constant.Success,
			StartTimestamp: mockStartTime,
			EndTimestamp:   s.mockTimeNow,
			UserId:         "1",
			DeviceId:       "24234436",
		},
		{
			Status:         constant.Pending,
			StartTimestamp: mockStartTime,
			EndTimestamp:   time.Time{},
			UserId:         "1",
			DeviceId:       "67787448",
		},
		{
			Status:         constant.Pending,
			StartTimestamp: mockStartTime,
			EndTimestamp:   time.Time{},
			UserId:         "1",
			DeviceId:       "190345342",
		},
	}

	deviceDistribution := &model.BookmarkDistribution{UserId: "1"}
	s.mockDynamoDBClient.EXPECT().GetRecordsByKeyAndFields(
		gomock.Eq(deviceDistribution)).Return(appDistributionSlice, nil)

	testUpdatedItem1 := &model.BookmarkDistribution{
		Status:         constant.Timeout,
		StartTimestamp: mockStartTime,
		EndTimestamp:   s.mockTimeNow,
		UserId:         "1",
		DeviceId:       "67787448",
		StatusMessage:  "Setting to timeout after 30 mins",
	}
	s.mockDynamoDBClient.EXPECT().UpdateRecordsByKey(gomock.Eq(testUpdatedItem1)).Return(nil)

	testUpdatedItem2 := &model.BookmarkDistribution{
		Status:         constant.Timeout,
		StartTimestamp: mockStartTime,
		EndTimestamp:   s.mockTimeNow,
		UserId:         "1",
		DeviceId:       "190345342",
		StatusMessage:  "Setting to timeout after 30 mins",
	}
	s.mockDynamoDBClient.EXPECT().UpdateRecordsByKey(gomock.Eq(testUpdatedItem2)).Return(nil)

	distribution := &model.UserBookmarks{UserId: "1"}
	mockPendingDist := &model.UserBookmarks{
		Status:         constant.Pending,
		StartTimestamp: mockStartTime,
		EndTimestamp:   time.Time{},
		UserId:         "1",
		LatestVersion:  TestLatestVersion,
	}
	s.mockDynamoDBClient.EXPECT().GetRecordByKey(gomock.Eq(distribution)).Return(mockPendingDist, nil)

	mockFailedDist := &model.UserBookmarks{
		Status:         constant.Timeout,
		StartTimestamp: mockStartTime,
		EndTimestamp:   s.mockTimeNow,
		UserId:         "1",
		LatestVersion:  TestLatestVersion,
	}
	s.mockDynamoDBClient.EXPECT().UpdateRecordsByKey(gomock.Eq(mockFailedDist)).Return(nil)

	GetDistributedBookmarks(s.context)

	s.EqualValues(http.StatusOK, s.recorder.Code)

	var distResponse models.DistributedBookmarksResponse
	distResponseJSON := s.recorder.Body.String()
	err := json.Unmarshal([]byte(distResponseJSON), &distResponse)

	s.NoError(err)
	s.Equal(3, len(distResponse.DistributionJobList))
}
