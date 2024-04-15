package mockutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/gin-gonic/gin"
)

func MockGinContext(w *httptest.ResponseRecorder) *gin.Context {
	gin.SetMode(gin.TestMode)

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = &http.Request{
		Header: make(http.Header),
		URL:    &url.URL{},
	}

	return ctx
}

func MockJSONRequest(c *gin.Context, method string, params gin.Params, content interface{}) {
	c.Request.Method = method
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = params

	if content == nil {
		return
	}

	jsonbytes, err := json.Marshal(content)
	if err != nil {
		panic(err)
	}

	c.Request.Body = io.NopCloser(bytes.NewBuffer(jsonbytes))
}

func MockJSONRequestWithQuery(c *gin.Context, method string, queryParams gin.Params, content interface{}) {
	if queryParams != nil {
		values := url.Values{}

		for _, p := range queryParams {
			values.Add(p.Key, p.Value)
		}

		query := values.Encode()

		u, err := url.Parse("/url?" + query)
		if err != nil {
			panic(err)
		}
		c.Request.URL = u
	}

	MockJSONRequest(c, method, nil, content)
}
