package util

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
)

type RestyClient struct {
	restyClient *resty.Client
	basedURI    string
}

type RestClient interface {
	CallService(url, method, authToken string, headers map[string]string, requestBody interface{}) (int, map[string]any, error)
	Get(url, authToken, contentType string) (int, map[string]any, error)
	Post(url, authToken, contentType, content string) (int, map[string]any, error)
	Put(url, authToken, contentType, content string) (int, map[string]any, error)
	Delete(url, authToken string) (int, map[string]any, error)
}

func NewRestyClient(uri string, httpTimeoutSeconds int) (RestClient, error) {
	restyClient := resty.New()
	if restyClient == nil {
		return nil, fmt.Errorf("fail to create resty client")
	}

	restyClient.SetTimeout(time.Duration(httpTimeoutSeconds) * time.Second)
	return &RestyClient{restyClient, uri}, nil
}

func (rc *RestyClient) Get(url, authToken, contentType string) (code int, response map[string]any, err error) {
	headers := map[string]string{
		"Content-Type": contentType,
	}
	return rc.CallService(url, "GET", authToken, headers, nil)
}

func (rc *RestyClient) Post(url, authToken, contentType, content string) (code int, response map[string]any, err error) {
	headers := map[string]string{
		"Content-Type": contentType,
	}
	return rc.CallService(url, "POST", authToken, headers, content)
}

func (rc *RestyClient) Put(url, authToken, contentType, content string) (code int, response map[string]any, err error) {
	headers := map[string]string{
		"Content-Type": contentType,
	}
	return rc.CallService(url, "PUT", authToken, headers, content)
}

func (rc *RestyClient) Delete(url, authToken string) (code int, response map[string]any, err error) {
	headers := map[string]string{}
	return rc.CallService(url, "DELETE", authToken, headers, nil)
}

func (rc *RestyClient) CallService(url, method, authToken string, headers map[string]string,
	requestBody interface{}) (code int, response map[string]any, err error) {
	jsonMap := map[string]any{}

	method = strings.ToUpper(method)

	req := rc.restyClient.R().
		SetAuthToken(authToken).
		SetHeader("accept", "application/json").
		SetHeaders(headers).
		SetResult(map[string]interface{}{})

	if requestBody != nil {
		req = req.SetBody(requestBody)
	}

	resp, err := req.Execute(method, fmt.Sprint(rc.basedURI, url))
	if err != nil {
		log.Error().Msgf("%s %s failed: %v", method, url, err.Error())
		return -1, nil, err
	}

	if resp.StatusCode() >= 200 && resp.StatusCode() < 300 {
		jsonResponse := (*resp.Result().(*map[string]interface{}))
		return resp.StatusCode(), jsonResponse, nil
	}

	return resp.StatusCode(), jsonMap, errors.New(resp.String())
}
