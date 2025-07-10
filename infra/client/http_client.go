package client

import (
	"io"
	"net/http"
	"time"
)

// HTTPClient は HTTPリクエストを送信するためのクライアントです。
type HTTPClient struct {
	client *http.Client
}

// NewHTTPClient は新しいHTTPClientを生成します。
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Get はGETリクエストを送信します。
func (c *HTTPClient) Get(url string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// Post はPOSTリクエストを送信します。
func (c *HTTPClient) Post(url string, headers map[string]string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
