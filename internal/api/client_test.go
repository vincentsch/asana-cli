// Package api tests the HTTP client, pagination, and retry behavior.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vincentsch/asana-cli/internal/version"
	"github.com/vincentsch/rungrad/testutil"
)

func TestGetAddsAuthHeader(t *testing.T) {
	var gotAuth string
	var gotAgent string

	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAgent = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": []}`))
	}))

	client := newTestClient(srv)
	_, err := client.Get(context.Background(), "/workspaces", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("expected auth header, got %q", gotAuth)
	}
	if gotAgent != version.UserAgent() {
		t.Fatalf("expected user-agent %q, got %q", version.UserAgent(), gotAgent)
	}
}

func TestPostSendsJSONBody(t *testing.T) {
	var gotMethod string
	var gotContentType string
	var gotPayload struct {
		Data    map[string]string `json:"data"`
		Options struct {
			Fields []string `json:"fields"`
		} `json:"options"`
	}

	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"gid":"1"}}`))
	}))

	client := newTestClient(srv)
	_, err := client.Post(context.Background(), "/tasks", RequestBody{
		Data: map[string]string{"name": "Test"},
		Options: &RequestOptions{
			Fields: []string{"gid"},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST, got %s", gotMethod)
	}
	if gotContentType != "application/json" {
		t.Fatalf("expected content-type application/json, got %q", gotContentType)
	}
	if gotPayload.Data["name"] != "Test" {
		t.Fatalf("unexpected payload data: %#v", gotPayload.Data)
	}
	if len(gotPayload.Options.Fields) != 1 || gotPayload.Options.Fields[0] != "gid" {
		t.Fatalf("unexpected payload options: %#v", gotPayload.Options.Fields)
	}
}

func TestPutSendsJSONBody(t *testing.T) {
	var gotMethod string
	var gotContentType string
	var gotPayload struct {
		Data map[string]any `json:"data"`
	}

	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"gid":"1"}}`))
	}))

	client := newTestClient(srv)
	_, err := client.Put(context.Background(), "/tasks/1", RequestBody{
		Data: map[string]any{"completed": true},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Fatalf("expected PUT, got %s", gotMethod)
	}
	if gotContentType != "application/json" {
		t.Fatalf("expected content-type application/json, got %q", gotContentType)
	}
	if gotPayload.Data["completed"] != true {
		t.Fatalf("unexpected payload data: %#v", gotPayload.Data)
	}
}

func TestMutationPreviewCapturesRequestWithoutNetworkIO(t *testing.T) {
	var requests atomic.Int32
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))

	client := newTestClient(srv)
	client.SetMutationPreview(true)
	_, err := client.Post(context.Background(), "/goals/123/setMetric", RequestBody{
		Data: map[string]any{
			"current_number_value": 50,
			"target_number_value":  100,
		},
	})
	var preview *MutationPreview
	if !errors.As(err, &preview) {
		t.Fatalf("Post error = %v, want MutationPreview", err)
	}
	if preview.Method != http.MethodPost || preview.Path != "/goals/123/setMetric" {
		t.Fatalf("unexpected preview route: %#v", preview)
	}
	if preview.Body["current_number_value"] != 50 || preview.Body["target_number_value"] != 100 {
		t.Fatalf("unexpected preview body: %#v", preview.Body)
	}
	if requests.Load() != 0 {
		t.Fatalf("preview made %d network request(s)", requests.Load())
	}

	if err := client.Delete(context.Background(), "/tasks/456"); !errors.As(err, &preview) {
		t.Fatalf("Delete error = %v, want MutationPreview", err)
	}
	if preview.Method != http.MethodDelete || preview.Path != "/tasks/456" || len(preview.Body) != 0 {
		t.Fatalf("unexpected delete preview: %#v", preview)
	}
}

func TestPaginateAccumulatesPages(t *testing.T) {
	var calls int32
	var handlerErr error
	var handlerMu sync.Mutex

	recordErr := func(err error) {
		if err == nil {
			return
		}
		handlerMu.Lock()
		if handlerErr == nil {
			handlerErr = err
		}
		handlerMu.Unlock()
	}

	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := atomic.AddInt32(&calls, 1)
		if r.URL.Query().Get("limit") != defaultLimit {
			recordErr(fmt.Errorf("expected limit %s, got %s", defaultLimit, r.URL.Query().Get("limit")))
		}
		switch call {
		case 1:
			if r.URL.Query().Get("offset") != "" {
				recordErr(fmt.Errorf("expected empty offset, got %q", r.URL.Query().Get("offset")))
			}
			_, _ = w.Write([]byte(`{"data":[{"gid":"1","name":"Alpha"}],"next_page":{"offset":"next"}}`))
		case 2:
			if r.URL.Query().Get("offset") != "next" {
				recordErr(fmt.Errorf("expected offset next, got %q", r.URL.Query().Get("offset")))
			}
			_, _ = w.Write([]byte(`{"data":[{"gid":"2","name":"Beta"}]}`))
		default:
			recordErr(fmt.Errorf("unexpected call %d", call))
		}
	}))

	client := newTestClient(srv)
	workspaces, err := Paginate[Workspace](context.Background(), client, "/workspaces", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if handlerErr != nil {
		t.Fatal(handlerErr)
	}
	if len(workspaces) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(workspaces))
	}
	if workspaces[0].GID != "1" || workspaces[1].GID != "2" {
		t.Fatalf("unexpected pagination order: %#v", workspaces)
	}
}

func TestRetryOn429RespectsRetryAfter(t *testing.T) {
	var calls int32
	var slept []time.Duration

	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := atomic.AddInt32(&calls, 1)
		if call == 1 {
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"errors":[{"message":"Rate limited"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"data": []}`))
	}))

	client := newTestClient(srv)
	client.sleep = func(_ context.Context, d time.Duration) error {
		slept = append(slept, d)
		return nil
	}

	_, err := client.Get(context.Background(), "/workspaces", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected 2 calls, got %d", got)
	}
	if len(slept) != 1 || slept[0] != 2*time.Second {
		t.Fatalf("expected sleep of 2s, got %#v", slept)
	}
	meta := client.RequestMetadata()
	if meta.Endpoint != "/workspaces" || meta.Attempts != 2 {
		t.Fatalf("unexpected request metadata: %#v", meta)
	}
	if len(meta.WaitsMS) != 1 || meta.WaitsMS[0] != 2000 {
		t.Fatalf("unexpected retry metadata: %#v", meta.WaitsMS)
	}
}

func TestRequestMetadataUsesSafeHeaderAllowlist(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Asana-Request-Id", "req-safe")
		w.Header().Set("X-RateLimit-Limit", "150")
		w.Header().Set("Authorization", "Bearer response-secret")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))

	client := newTestClient(srv)
	if _, err := Paginate[Workspace](context.Background(), client, "/workspaces", nil); err != nil {
		t.Fatalf("Paginate returned an error: %v", err)
	}
	meta := client.RequestMetadata()
	if meta.RequestID != "req-safe" || len(meta.RequestIDs) != 1 || meta.RequestIDs[0] != "req-safe" {
		t.Fatalf("unexpected request ids: %#v", meta)
	}
	if !meta.Paginated || meta.NextCursor != "" || meta.RateLimit["X-RateLimit-Limit"] != "150" {
		t.Fatalf("unexpected metadata: %#v", meta)
	}
	if _, exists := meta.RateLimit["Authorization"]; exists {
		t.Fatalf("metadata retained an unsafe header: %#v", meta.RateLimit)
	}
}

func TestRetryOn5xxUsesBackoffForGet(t *testing.T) {
	var calls int32
	var slept []time.Duration

	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := atomic.AddInt32(&calls, 1)
		if call < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errors":[{"message":"Server error"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"data": []}`))
	}))

	client := newTestClient(srv)
	client.sleep = func(_ context.Context, d time.Duration) error {
		slept = append(slept, d)
		return nil
	}

	_, err := client.Get(context.Background(), "/workspaces", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("expected 3 calls, got %d", got)
	}
	if len(slept) != 2 || slept[0] != time.Second || slept[1] != 2*time.Second {
		t.Fatalf("expected backoff [1s 2s], got %#v", slept)
	}
}

func TestNoRetryOn5xxForPost(t *testing.T) {
	var calls int32

	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"errors":[{"message":"Server error"}]}`))
	}))

	client := newTestClient(srv)
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/workspaces", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	resp, err := client.doWithRetry(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500 response, got %d", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected 1 call, got %d", got)
	}
}

func TestRetryOn429ReplaysRequestBody(t *testing.T) {
	var calls int32
	var bodies []string

	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		bodies = append(bodies, string(body))

		call := atomic.AddInt32(&calls, 1)
		if call == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"errors":[{"message":"Rate limited"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"data": []}`))
	}))

	client := newTestClient(srv)
	client.sleep = func(_ context.Context, d time.Duration) error {
		return nil
	}

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/workspaces", io.NopCloser(strings.NewReader("payload")))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	resp, err := client.doWithRetry(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected 2 calls, got %d", got)
	}
	if len(bodies) != 2 || bodies[0] != "payload" || bodies[1] != "payload" {
		t.Fatalf("expected replayed body, got %#v", bodies)
	}
}

func TestParseErrorIncludesRequestID(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Asana-Request-Id", "req-123")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors":[{"message":"Not Found"}]}`))
	}))

	client := newTestClient(srv)
	_, err := client.Get(context.Background(), "/workspaces", nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.RequestID != "req-123" {
		t.Fatalf("expected request id, got %q", apiErr.RequestID)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", apiErr.StatusCode)
	}
	if len(apiErr.Errors) != 1 || apiErr.Errors[0].Message != "Not Found" {
		t.Fatalf("unexpected error payload: %#v", apiErr.Errors)
	}
}

func TestTimeoutReturnsNetError(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte(`{"data": []}`))
	}))

	client := newTestClient(srv)
	client.httpClient.Timeout = 50 * time.Millisecond

	_, err := client.Get(context.Background(), "/workspaces", nil)
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if !netErr.Timeout() {
			t.Fatalf("expected timeout error, got %v", err)
		}
		return
	}
	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "Timeout") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func newTestClient(srv *httptest.Server) *Client {
	client := NewClient("test-token")
	client.baseURL = srv.URL
	client.httpClient = srv.Client()
	client.sleep = func(context.Context, time.Duration) error { return nil }
	return client
}

func ExampleAPIError_Error() {
	err := &APIError{StatusCode: http.StatusNotFound, RequestID: "abc"}
	fmt.Println(err.Error())
	// Output:
	// API error (HTTP 404): Not Found
	// Request ID: abc
}
