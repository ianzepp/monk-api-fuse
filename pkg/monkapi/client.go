package monkapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client handles communication with the Monk File API
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new Monk API client with connection pooling
func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
			Timeout: 30 * time.Second,
		},
	}
}

// post performs a POST request to the API
func (c *Client) post(ctx context.Context, endpoint string, body interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			return nil, &APIError{
				StatusCode: resp.StatusCode,
				ErrorCode:  errResp.ErrorCode,
				Message:    errResp.Error,
			}
		}
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// List retrieves directory listing from the File API
// Use pick parameter to reduce bandwidth (e.g., "entries" for 60% reduction)
func (c *Client) List(ctx context.Context, path string, opts ListOptions, pick string) (*ListResponse, error) {
	req := map[string]interface{}{
		"path":         path,
		"file_options": opts,
	}

	endpoint := "/api/file/list"
	if pick != "" {
		endpoint += "?pick=" + url.QueryEscape(pick)
	}

	respBody, err := c.post(ctx, endpoint, req)
	if err != nil {
		return nil, err
	}

	// Unwrap the API response
	var wrapper APIWrapper
	if err := json.Unmarshal(respBody, &wrapper); err != nil {
		return nil, fmt.Errorf("unmarshal wrapper: %w", err)
	}

	var result ListResponse
	if err := json.Unmarshal(wrapper.Data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal list response: %w", err)
	}

	return &result, nil
}

// Stat retrieves file/directory metadata from the File API
// Use pick parameter to reduce bandwidth (e.g., "file_metadata" for 40-50% reduction)
func (c *Client) Stat(ctx context.Context, path string, pick string) (*StatResponse, error) {
	req := map[string]interface{}{
		"path": path,
	}

	endpoint := "/api/file/stat"
	if pick != "" {
		endpoint += "?pick=" + url.QueryEscape(pick)
	}

	respBody, err := c.post(ctx, endpoint, req)
	if err != nil {
		return nil, err
	}

	// Unwrap the API response
	var wrapper APIWrapper
	if err := json.Unmarshal(respBody, &wrapper); err != nil {
		return nil, fmt.Errorf("unmarshal wrapper: %w", err)
	}

	var result StatResponse
	if err := json.Unmarshal(wrapper.Data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal stat response: %w", err)
	}

	return &result, nil
}

// Retrieve retrieves file content from the File API
// Use pick parameter to reduce bandwidth (e.g., "content" for 80% reduction)
func (c *Client) Retrieve(ctx context.Context, path string, opts RetrieveOptions, pick string) (*RetrieveResponse, error) {
	req := map[string]interface{}{
		"path":         path,
		"file_options": opts,
	}

	endpoint := "/api/file/retrieve"
	if pick != "" {
		endpoint += "?pick=" + url.QueryEscape(pick)
	}

	respBody, err := c.post(ctx, endpoint, req)
	if err != nil {
		return nil, err
	}

	// Unwrap the API response
	var wrapper APIWrapper
	if err := json.Unmarshal(respBody, &wrapper); err != nil {
		return nil, fmt.Errorf("unmarshal wrapper: %w", err)
	}

	var result RetrieveResponse
	if err := json.Unmarshal(wrapper.Data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal retrieve response: %w", err)
	}

	return &result, nil
}

// APIError represents an error from the Monk API
type APIError struct {
	StatusCode int
	ErrorCode  string
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d (%s): %s", e.StatusCode, e.ErrorCode, e.Message)
}

// Store stores file content to the File API
func (c *Client) Store(ctx context.Context, path string, content interface{}, opts StoreOptions, pick string) (*StoreResponse, error) {
	req := map[string]interface{}{
		"path":         path,
		"content":      content,
		"file_options": opts,
	}

	endpoint := "/api/file/store"
	if pick != "" {
		endpoint += "?pick=" + url.QueryEscape(pick)
	}

	respBody, err := c.post(ctx, endpoint, req)
	if err != nil {
		return nil, err
	}

	// Unwrap the API response
	var wrapper APIWrapper
	if err := json.Unmarshal(respBody, &wrapper); err != nil {
		return nil, fmt.Errorf("unmarshal wrapper: %w", err)
	}

	var result StoreResponse
	if err := json.Unmarshal(wrapper.Data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal store response: %w", err)
	}

	return &result, nil
}

// IsNotFound returns true if the error is a 404 not found
func IsNotFound(err error) bool {
	apiErr, ok := err.(*APIError)
	return ok && apiErr.StatusCode == 404
}
