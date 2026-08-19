// Package api implements the Asana HTTP client, retries, and pagination.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vincentsch/asana-cli/internal/version"
)

const (
	// DefaultBaseURL is the production Asana REST API endpoint.
	DefaultBaseURL = "https://app.asana.com/api/1.0"
	defaultLimit   = "100"
	maxRetries     = 3
)

// Client wraps HTTP requests to the Asana API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
	userAgent  string
	sleep      func(context.Context, time.Duration) error
	metaMu     sync.Mutex
	metadata   RequestMetadata
	preview    bool
}

// MutationPreview reports a mutation intercepted before network I/O.
type MutationPreview struct {
	Method string
	Path   string
	Body   map[string]any
}

// Error identifies an intentionally intercepted mutation.
func (p *MutationPreview) Error() string {
	return "mutation intercepted for dry-run preview"
}

// RequestMetadata is the secret-free diagnostic state collected across the
// requests made by one command invocation.
type RequestMetadata struct {
	Endpoint   string
	RequestID  string
	RequestIDs []string
	NextCursor string
	Paginated  bool
	Attempts   int
	WaitsMS    []int64
	RateLimit  map[string]string
}

// RequestBody wraps data and options for Asana POST/PUT requests.
type RequestBody struct {
	Data    any             `json:"data"`
	Options *RequestOptions `json:"options,omitempty"`
}

// RequestOptions specifies output field selection for POST/PUT.
type RequestOptions struct {
	Fields []string `json:"fields,omitempty"`
}

// FileUpload represents a file to upload via multipart form.
type FileUpload struct {
	FieldName string
	FileName  string
	Content   io.Reader
}

// NewClient constructs a client with default settings.
func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    DefaultBaseURL,
		token:      token,
		userAgent:  version.UserAgent(),
		sleep:      sleepWithContext,
	}
}

// SetBaseURL overrides the API base URL (primarily for tests).
func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

// SetHTTPClient overrides the HTTP client (primarily for tests).
func (c *Client) SetHTTPClient(client *http.Client) {
	if client == nil {
		return
	}
	c.httpClient = client
}

// SetMutationPreview enables or disables interception of mutation requests.
func (c *Client) SetMutationPreview(enabled bool) {
	c.metaMu.Lock()
	defer c.metaMu.Unlock()
	c.preview = enabled
}

// MutationPreviewEnabled reports whether mutation requests are being captured.
func (c *Client) MutationPreviewEnabled() bool {
	return c.previewEnabled()
}

// Get issues a GET request to the Asana API and returns the response body.
func (c *Client) Get(ctx context.Context, path string, query url.Values) ([]byte, error) {
	c.recordEndpoint(path)
	fullURL := strings.TrimRight(c.baseURL, "/") + "/" + strings.TrimLeft(path, "/")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}
	if len(query) > 0 {
		req.URL.RawQuery = query.Encode()
	}

	c.addHeaders(req)

	resp, err := c.doWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, parseError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &ResponseError{Err: err}
	}

	return body, nil
}

// Post issues a POST request with a JSON body.
func (c *Client) Post(ctx context.Context, path string, body RequestBody) ([]byte, error) {
	return c.doJSON(ctx, http.MethodPost, path, body)
}

// Put issues a PUT request with a JSON body.
func (c *Client) Put(ctx context.Context, path string, body RequestBody) ([]byte, error) {
	return c.doJSON(ctx, http.MethodPut, path, body)
}

// Delete issues a DELETE request and returns nil on success.
func (c *Client) Delete(ctx context.Context, path string) error {
	c.recordEndpoint(path)
	if c.previewEnabled() {
		return &MutationPreview{Method: http.MethodDelete, Path: normalizePath(path)}
	}
	fullURL := strings.TrimRight(c.baseURL, "/") + "/" + strings.TrimLeft(path, "/")

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, fullURL, nil)
	if err != nil {
		return err
	}

	c.addHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	c.recordResponse(resp)

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return parseError(resp)
	}

	return nil
}

// PostMultipart issues a POST request with multipart/form-data.
// fields is a map of form field names to values.
// file is optional file data with FieldName, FileName, and Content.
func (c *Client) PostMultipart(ctx context.Context, path string, fields map[string]string, file *FileUpload) ([]byte, error) {
	c.recordEndpoint(path)
	if c.previewEnabled() {
		body := make(map[string]any, len(fields)+1)
		for key, value := range fields {
			body[key] = value
		}
		if file != nil {
			body[file.FieldName] = file.FileName
		}
		return nil, &MutationPreview{Method: http.MethodPost, Path: normalizePath(path), Body: body}
	}
	fullURL := strings.TrimRight(c.baseURL, "/") + "/" + strings.TrimLeft(path, "/")

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Write string fields
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, &ResponseError{Err: err}
		}
	}

	// Write file if provided
	if file != nil && file.Content != nil {
		part, err := writer.CreateFormFile(file.FieldName, file.FileName)
		if err != nil {
			return nil, &ResponseError{Err: err}
		}
		if _, err := io.Copy(part, file.Content); err != nil {
			return nil, &ResponseError{Err: err}
		}
	}

	if err := writer.Close(); err != nil {
		return nil, &ResponseError{Err: err}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, &buf)
	if err != nil {
		return nil, err
	}

	c.addHeaders(req)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// No retry for multipart uploads (body not replayable)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	c.recordResponse(resp)

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, parseError(resp)
	}

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &ResponseError{Err: err}
	}

	return payload, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, body RequestBody) ([]byte, error) {
	c.recordEndpoint(path)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, &ResponseError{Err: err}
	}
	if c.previewEnabled() {
		return nil, &MutationPreview{Method: method, Path: normalizePath(path), Body: previewBody(body.Data)}
	}
	fullURL := strings.TrimRight(c.baseURL, "/") + "/" + strings.TrimLeft(path, "/")

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	c.addHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, parseError(resp)
	}

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &ResponseError{Err: err}
	}

	return payload, nil
}

// Paginate fetches all pages for a cursor-based endpoint.
func Paginate[T any](ctx context.Context, client *Client, path string, query url.Values) ([]T, error) {
	values := cloneValues(query)
	values.Set("limit", defaultLimit)

	all := make([]T, 0)
	for {
		payload, err := client.Get(ctx, path, values)
		if err != nil {
			return nil, err
		}

		var page Response[[]T]
		if err := json.Unmarshal(payload, &page); err != nil {
			return nil, &ResponseError{Err: err}
		}

		all = append(all, page.Data...)
		if page.NextPage == nil || page.NextPage.Offset == "" {
			client.recordPagination("")
			return all, nil
		}

		client.recordPagination(page.NextPage.Offset)
		values.Set("offset", page.NextPage.Offset)
	}
}

func (c *Client) addHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
}

func (c *Client) doWithRetry(req *http.Request) (*http.Response, error) {
	request, err := ensureReplayableRequest(req)
	if err != nil {
		return nil, err
	}

	retry5xx := req.Method == http.MethodGet || req.Method == http.MethodHead
	backoff := time.Second

	var lastResp *http.Response
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			request, err = cloneRequest(request)
			if err != nil {
				return nil, &ResponseError{Err: err}
			}
		}

		resp, err := c.httpClient.Do(request)
		if err != nil {
			return nil, err
		}
		lastResp = resp
		c.recordResponse(resp)

		switch {
		case resp.StatusCode == http.StatusTooManyRequests:
			if attempt == maxRetries {
				return resp, nil
			}
			wait := retryAfter(resp.Header.Get("Retry-After"))
			c.recordWait(wait)
			drainAndClose(resp)
			if err := c.sleep(req.Context(), wait); err != nil {
				return nil, err
			}
			continue
		case retry5xx && isRetryableStatus(resp.StatusCode):
			if attempt == maxRetries {
				return resp, nil
			}
			drainAndClose(resp)
			c.recordWait(backoff)
			if err := c.sleep(req.Context(), backoff); err != nil {
				return nil, err
			}
			backoff *= 2
			continue
		default:
			return resp, nil
		}
	}

	return lastResp, nil
}

// RequestMetadata returns a copy of the current command's diagnostic request
// metadata. Only an allowlist of response headers is retained.
func (c *Client) RequestMetadata() RequestMetadata {
	if c == nil {
		return RequestMetadata{}
	}
	c.metaMu.Lock()
	defer c.metaMu.Unlock()
	meta := c.metadata
	meta.RequestIDs = append([]string(nil), c.metadata.RequestIDs...)
	meta.WaitsMS = append([]int64(nil), c.metadata.WaitsMS...)
	if c.metadata.RateLimit != nil {
		meta.RateLimit = make(map[string]string, len(c.metadata.RateLimit))
		for key, value := range c.metadata.RateLimit {
			meta.RateLimit[key] = value
		}
	}
	return meta
}

func (c *Client) recordEndpoint(path string) {
	c.metaMu.Lock()
	defer c.metaMu.Unlock()
	c.metadata.Endpoint = normalizePath(path)
}

func (c *Client) previewEnabled() bool {
	c.metaMu.Lock()
	defer c.metaMu.Unlock()
	return c.preview
}

func normalizePath(path string) string {
	return "/" + strings.TrimLeft(path, "/")
}

func previewBody(data any) map[string]any {
	if data == nil {
		return nil
	}
	if body, ok := data.(map[string]any); ok {
		return body
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return map[string]any{"data": fmt.Sprint(data)}
	}
	var body map[string]any
	if err := json.Unmarshal(encoded, &body); err != nil {
		return map[string]any{"data": string(encoded)}
	}
	return body
}

func (c *Client) recordResponse(resp *http.Response) {
	c.metaMu.Lock()
	defer c.metaMu.Unlock()
	c.metadata.Attempts++
	requestID := resp.Header.Get("X-Asana-Request-Id")
	if requestID == "" {
		requestID = resp.Header.Get("X-Request-Id")
	}
	if requestID != "" {
		c.metadata.RequestID = requestID
		if len(c.metadata.RequestIDs) == 0 || c.metadata.RequestIDs[len(c.metadata.RequestIDs)-1] != requestID {
			c.metadata.RequestIDs = append(c.metadata.RequestIDs, requestID)
		}
	}
	for _, header := range []string{"X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"} {
		if value := resp.Header.Get(header); value != "" {
			if c.metadata.RateLimit == nil {
				c.metadata.RateLimit = map[string]string{}
			}
			c.metadata.RateLimit[header] = value
		}
	}
}

func (c *Client) recordWait(wait time.Duration) {
	c.metaMu.Lock()
	defer c.metaMu.Unlock()
	c.metadata.WaitsMS = append(c.metadata.WaitsMS, wait.Milliseconds())
}

func (c *Client) recordPagination(nextCursor string) {
	c.metaMu.Lock()
	defer c.metaMu.Unlock()
	c.metadata.Paginated = true
	c.metadata.NextCursor = nextCursor
}

func parseError(resp *http.Response) *APIError {
	var payload ErrorResponse
	body, err := io.ReadAll(resp.Body)
	if err == nil && len(body) > 0 {
		_ = json.Unmarshal(body, &payload)
	}

	return &APIError{
		StatusCode: resp.StatusCode,
		RequestID:  resp.Header.Get("X-Asana-Request-Id"),
		Errors:     payload.Errors,
	}
}

func retryAfter(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Second
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if parsed, err := http.ParseTime(value); err == nil {
		wait := time.Until(parsed)
		if wait > 0 {
			return wait
		}
	}

	return time.Second
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func ensureReplayableRequest(req *http.Request) (*http.Request, error) {
	if req.Body == nil || req.Body == http.NoBody || req.GetBody != nil {
		return req, nil
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, &ResponseError{Err: err}
	}
	_ = req.Body.Close()

	req.Body = io.NopCloser(bytes.NewReader(body))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	req.ContentLength = int64(len(body))

	return req, nil
}

func cloneRequest(req *http.Request) (*http.Request, error) {
	clone := req.Clone(req.Context())
	if req.Body == nil || req.Body == http.NoBody {
		return clone, nil
	}
	if req.GetBody == nil {
		return nil, errors.New("request body cannot be replayed")
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, err
	}
	clone.Body = body
	return clone, nil
}

func cloneValues(values url.Values) url.Values {
	clone := url.Values{}
	for key, list := range values {
		clone[key] = append([]string(nil), list...)
	}
	return clone
}

func isRetryableStatus(status int) bool {
	switch status {
	case http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func drainAndClose(resp *http.Response) {
	if resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}
