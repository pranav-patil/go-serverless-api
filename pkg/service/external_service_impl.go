package service

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
)

type ExternalService struct {
	restyClient *resty.Client

	apiHost string
}

type ProviderMetadata struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Device struct {
	Id                uint               `json:"ID"`
	InstanceId        string             `json:"instanceId"`
	ProviderMetadatas []ProviderMetadata `json:"providerMetadata"`
}

type DeviceListResponse struct {
	Devices []Device `json:"devices"`
}

const (
	ExtServiceHTTPTimeoutSeconds int    = 30
	GetDevicesByUserIDPath       string = "/api/user/devices"
)

func newRestyClient(httpTimeoutSeconds int) (*resty.Client, error) {
	restyClient := resty.New()
	if restyClient == nil {
		return nil, fmt.Errorf("fail to create resty client")
	}

	restyClient.SetTimeout(time.Duration(httpTimeoutSeconds) * time.Second)
	return restyClient, nil
}

func NewExternalService() (ExternalServiceAPI, error) {
	apiHost, ok := os.LookupEnv("API_HOST")
	if !ok || apiHost == "" {
		return nil, fmt.Errorf("environment variable API_HOST is not set")
	}

	restyClient, err := newRestyClient(ExtServiceHTTPTimeoutSeconds)
	if err != nil {
		return nil, err
	}

	return &ExternalService{restyClient, apiHost}, nil
}

func (p *ExternalService) GetUserDevices(jwt, accountId string) (*DeviceListResponse, error) {
	deviceServiceURI := "http://" + p.apiHost + GetDevicesByUserIDPath
	log.Debug().Msgf("Device service api URI: %s", deviceServiceURI)

	resp, err := p.restyClient.R().
		SetAuthToken(jwt).
		SetHeaders(map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
			"api-version":  "v1",
			"X-USER-ID":    accountId,
		}).
		SetResult(&DeviceListResponse{}).
		Get(deviceServiceURI)

	if err != nil {
		log.Error().Msgf("Get DeviceList by UserId failed: %v", err.Error())
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		errMsg := fmt.Sprintf("fail to get device list from user '%s': %d - '%s'", accountId, resp.StatusCode(), resp.String())
		log.Error().Msg(errMsg)
		return nil, fmt.Errorf("%s", errMsg)
	}

	deviceListResponse, ok := resp.Result().(*DeviceListResponse)
	if !ok {
		return nil, fmt.Errorf("fail to convert to response")
	}

	return deviceListResponse, nil
}
