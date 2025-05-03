package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPClient is a wrapper around the standard http client with added functionality
type HTTPClient struct {
	client         *http.Client
	baseURL        string
	defaultHeaders map[string]string
	maxRetries     int
	retryDelay     time.Duration
}

// HTTPClientOption is a function that configures the HTTPClient
type HTTPClientOption func(*HTTPClient)

// WithTimeout sets the timeout for the HTTP client
func WithTimeout(timeout time.Duration) HTTPClientOption {
	return func(c *HTTPClient) {
		c.client.Timeout = timeout
	}
}

// WithBaseURL sets the base URL for the HTTP client
func WithBaseURL(baseURL string) HTTPClientOption {
	return func(c *HTTPClient) {
		c.baseURL = baseURL
	}
}

// WithDefaultHeaders sets default headers for the HTTP client
func WithDefaultHeaders(headers map[string]string) HTTPClientOption {
	return func(c *HTTPClient) {
		c.defaultHeaders = headers
	}
}

// WithRetries configures retry behavior
func WithRetries(maxRetries int, retryDelay time.Duration) HTTPClientOption {
	return func(c *HTTPClient) {
		c.maxRetries = maxRetries
		c.retryDelay = retryDelay
	}
}

// NewHTTPClient creates a new HTTPClient with the given options
func NewHTTPClient(options ...HTTPClientOption) *HTTPClient {
	client := &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		defaultHeaders: map[string]string{
			"Content-Type": "application/json",
		},
		maxRetries: 3,
		retryDelay: 500 * time.Millisecond,
	}

	for _, option := range options {
		option(client)
	}

	return client
}

// Request represents an HTTP request
type Request struct {
	Method      string
	Path        string
	QueryParams map[string]string
	Headers     map[string]string
	Body        interface{}
	Context     context.Context
}

// Response represents an HTTP response
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// Error represents an HTTP error
type Error struct {
	StatusCode int
	Message    string
	Response   *Response
}

func (e *Error) Error() string {
	return fmt.Sprintf("HTTP error %d: %s", e.StatusCode, e.Message)
}

// Do executes an HTTP request and returns the response
func (c *HTTPClient) Do(req *Request) (*Response, error) {
	if req.Context == nil {
		req.Context = context.Background()
	}

	var err error
	var resp *http.Response
	var httpReq *http.Request

	url := req.Path
	if c.baseURL != "" {
		url = c.baseURL + url
	}

	// Add query parameters if any
	if len(req.QueryParams) > 0 {
		url += "?"
		i := 0
		for k, v := range req.QueryParams {
			if i > 0 {
				url += "&"
			}
			url += fmt.Sprintf("%s=%s", k, v)
			i++
		}
	}

	// Create request with body if needed
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq, err = http.NewRequestWithContext(req.Context, req.Method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	for k, v := range c.defaultHeaders {
		httpReq.Header.Set(k, v)
	}

	// Set request-specific headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Perform the request with retries
	var respBody []byte
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-req.Context.Done():
				return nil, req.Context.Err()
			case <-time.After(c.retryDelay * time.Duration(attempt)):
				// Continue with retry after delay
			}
		}

		resp, err = c.client.Do(httpReq)
		if err != nil {
			if attempt == c.maxRetries {
				return nil, fmt.Errorf("failed after %d retries: %w", c.maxRetries, err)
			}
			continue
		}

		// Read response body
		respBody, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			if attempt == c.maxRetries {
				return nil, fmt.Errorf("failed to read response body after %d retries: %w", c.maxRetries, err)
			}
			continue
		}

		// No need to retry if we got here successfully
		break
	}

	response := &Response{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       respBody,
	}

	// Check for error status codes
	if resp.StatusCode >= 400 {
		return response, &Error{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("request failed with status code %d", resp.StatusCode),
			Response:   response,
		}
	}

	return response, nil
}

// Get performs a GET request
func (c *HTTPClient) Get(path string, queryParams map[string]string, headers map[string]string) (*Response, error) {
	return c.Do(&Request{
		Method:      http.MethodGet,
		Path:        path,
		QueryParams: queryParams,
		Headers:     headers,
	})
}

// Post performs a POST request
func (c *HTTPClient) Post(path string, body interface{}, headers map[string]string) (*Response, error) {
	return c.Do(&Request{
		Method:  http.MethodPost,
		Path:    path,
		Body:    body,
		Headers: headers,
	})
}

// Put performs a PUT request
func (c *HTTPClient) Put(path string, body interface{}, headers map[string]string) (*Response, error) {
	return c.Do(&Request{
		Method:  http.MethodPut,
		Path:    path,
		Body:    body,
		Headers: headers,
	})
}

// Delete performs a DELETE request
func (c *HTTPClient) Delete(path string, headers map[string]string) (*Response, error) {
	return c.Do(&Request{
		Method:  http.MethodDelete,
		Path:    path,
		Headers: headers,
	})
}

// Patch performs a PATCH request
func (c *HTTPClient) Patch(path string, body interface{}, headers map[string]string) (*Response, error) {
	return c.Do(&Request{
		Method:  http.MethodPatch,
		Path:    path,
		Body:    body,
		Headers: headers,
	})
}

// DecodeJSON decodes the response body into the target
func (r *Response) DecodeJSON(target interface{}) error {
	if r.Body == nil {
		return errors.New("empty response body")
	}
	return json.Unmarshal(r.Body, target)
}

// String returns the response body as a string
func (r *Response) String() string {
	return string(r.Body)
}

// IsSuccess returns true if the status code is between 200 and 299
func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode <= 299
}
