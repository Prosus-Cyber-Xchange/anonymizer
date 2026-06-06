package driver

import (
	"context"
	"io"
	"net/http"
	"strings"

	"anonymizer-service-v2/e2e/specifications"
)

type HTTPDriver struct {
	baseURL string
	client  *http.Client
}

func NewHTTPDriver(baseURL string, client *http.Client) *HTTPDriver {
	return &HTTPDriver{baseURL: baseURL, client: client}
}

func (d *HTTPDriver) Anonymize(ctx context.Context, body string, headers map[string][]string) (specifications.Response, error) {
	return d.do(ctx, http.MethodPost, "/api/v1/anonymize", body, headers)
}

func (d *HTTPDriver) AnonymizeBatch(ctx context.Context, body string, headers map[string][]string) (specifications.Response, error) {
	return d.do(ctx, http.MethodPost, "/api/v1/anonymize/batch", body, headers)
}

func (d *HTTPDriver) Health(ctx context.Context) (specifications.Response, error) {
	return d.do(ctx, http.MethodGet, "/health", "", nil)
}

func (d *HTTPDriver) do(ctx context.Context, method, path, body string, headers map[string][]string) (specifications.Response, error) {
	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, d.baseURL+path, reqBody)
	if err != nil {
		return specifications.Response{}, err
	}
	for k, vs := range headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return specifications.Response{}, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return specifications.Response{}, err
	}
	return specifications.Response{
		StatusCode: resp.StatusCode,
		Body:       string(b),
		Headers:    resp.Header,
	}, nil
}
