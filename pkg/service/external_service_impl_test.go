package service

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/suite"
)

const (
	MockAPIHost           string = "mock-api-host"
	MockJWT               string = "mock-jwt"
	MockAccountID         string = "999999"
	MockAccountIDNumber   uint   = 999999
	MockRestyError        string = "mock-resty-error"
	MockHTTPTimeoutInSecs int    = 1

	MockGetDevicesByUserIDPathURI  string = "http://" + MockAPIHost + GetDevicesByUserIDPath
	MockGetDevicesByUserIDResponse string = "mock-appl-list-response"
)

type ExternalServiceTestSuite struct {
	suite.Suite
	mockResty *resty.Client
}

func TestExternalServiceSuite(t *testing.T) {
	suite.Run(t, new(ExternalServiceTestSuite))
}

func (s *ExternalServiceTestSuite) SetupTest() {
	s.T().Setenv("API_HOST", "mock-api-host")

	mockResty, err := newRestyClient(5)
	s.Nil(err)
	s.NotNil(mockResty)
	s.mockResty = mockResty

	httpmock.ActivateNonDefault(s.mockResty.GetClient())
}

func (s *ExternalServiceTestSuite) TearDownTest() {
	httpmock.DeactivateAndReset()
}

func (s *ExternalServiceTestSuite) TestNewExternalService() {
	extServiceClient, err := NewExternalService()

	s.NotNil(extServiceClient)
	s.Nil(err)
}

func (s *ExternalServiceTestSuite) TestNewExtServiceAPIHostEnvNotSet() {
	s.T().Setenv("API_HOST", "")
	extServiceClient, err := NewExternalService()

	s.Nil(extServiceClient)
	s.Equal("environment variable API_HOST is not set", err.Error())
}

func (s *ExternalServiceTestSuite) TestGetDeviceList() {
	mockResponse := DeviceListResponse{
		Devices: []Device{
			{
				Id:         MockAccountIDNumber,
				InstanceId: "1",
				ProviderMetadatas: []ProviderMetadata{
					{
						Key:   "mock-key",
						Value: "mock-value",
					},
				},
			},
			{
				Id:         MockAccountIDNumber,
				InstanceId: "2",
				ProviderMetadatas: []ProviderMetadata{
					{
						Key:   "mock-key",
						Value: "mock-value",
					},
				},
			},
		},
	}

	httpmock.RegisterResponder("GET", MockGetDevicesByUserIDPathURI,
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusOK, mockResponse)
		},
	)

	extServiceClient := &ExternalService{s.mockResty, MockAPIHost}
	deviceListResponse, err := extServiceClient.GetUserDevices(MockJWT, MockAccountID)

	s.Nil(err)
	s.Equal(mockResponse, *deviceListResponse)
}

func (s *ExternalServiceTestSuite) TestGetDeviceListHttpTimeout() {
	s.mockResty.SetTimeout(time.Duration(MockHTTPTimeoutInSecs) * time.Second)

	httpmock.RegisterResponder("GET", MockGetDevicesByUserIDPathURI,
		func(req *http.Request) (*http.Response, error) {
			time.Sleep(time.Duration(MockHTTPTimeoutInSecs+2) * time.Second)
			return httpmock.NewJsonResponse(http.StatusOK, DeviceListResponse{})
		},
	)

	extServiceClient := &ExternalService{s.mockResty, MockAPIHost}
	devices, err := extServiceClient.GetUserDevices(MockJWT, MockAccountID)

	s.NotNil(err)
	s.Empty(devices)
}

func (s *ExternalServiceTestSuite) TestGetDeviceListRestyError() {
	httpmock.RegisterResponder("GET", MockGetDevicesByUserIDPathURI,
		func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf(MockRestyError)
		},
	)

	extServiceClient := &ExternalService{s.mockResty, MockAPIHost}

	deviceListResponse, err := extServiceClient.GetUserDevices(MockJWT, MockAccountID)

	s.Nil(deviceListResponse)
	s.ErrorContains(err, MockRestyError)
}

func (s *ExternalServiceTestSuite) TestGetDeviceListStatusNot200() {
	httpmock.RegisterResponder("GET", MockGetDevicesByUserIDPathURI,
		httpmock.NewStringResponder(http.StatusInternalServerError, MockGetDevicesByUserIDResponse))

	extServiceClient := &ExternalService{s.mockResty, MockAPIHost}
	deviceListResponse, err := extServiceClient.GetUserDevices(MockJWT, MockAccountID)

	s.Nil(deviceListResponse)

	exceptError := fmt.Errorf("fail to get device list from user '%s': %d - '%s'",
		MockAccountID, http.StatusInternalServerError, MockGetDevicesByUserIDResponse)
	s.Equal(exceptError.Error(), err.Error())
}

func MakeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
