// Package cmd tests goal command behavior.
package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/api"
	"github.com/vincentsch/rungrad/testutil"
)

func TestGoalListPremiumError(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"errors":[{"message":"Payment required"}]}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, _ := newGoalListTestCmd()
	if err := cmd.Flags().Set("workspace-gid", "w1"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	err := runGoalList(cmd, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "goals require a premium workspace") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGoalMetricSetEndpointSelection(t *testing.T) {
	cases := []struct {
		name       string
		setTarget  bool
		expectPath string
		checkBody  func(*testing.T, api.RequestBody)
	}{
		{
			name:       "current-only",
			setTarget:  false,
			expectPath: "/goals/123/setMetricCurrentValue",
			checkBody: func(t *testing.T, body api.RequestBody) {
				payload, ok := body.Data.(map[string]any)
				if !ok {
					t.Fatalf("unexpected body type %T", body.Data)
				}
				current, ok := payload["current_number_value"].(float64)
				if !ok || current != 12 {
					t.Fatalf("unexpected current value: %#v", payload)
				}
			},
		},
		{
			name:       "with-target",
			setTarget:  true,
			expectPath: "/goals/123/setMetric",
			checkBody: func(t *testing.T, body api.RequestBody) {
				payload, ok := body.Data.(map[string]any)
				if !ok {
					t.Fatalf("unexpected body type %T", body.Data)
				}
				current, ok := payload["current_number_value"].(float64)
				if !ok || current != 12 {
					t.Fatalf("unexpected current value: %#v", payload)
				}
				target, ok := payload["target_number_value"].(float64)
				if !ok || target != 25 {
					t.Fatalf("unexpected target value: %#v", payload)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var sawPath string
			var gotBody api.RequestBody

			srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sawPath = r.URL.Path
				if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
					t.Fatalf("failed to decode body: %v", err)
				}
				_, _ = w.Write([]byte(`{"data":{}}`))
			}))

			apiClient = newTestClient(srv)
			defer func() { apiClient = nil }()

			cmd, _ := newGoalMetricSetTestCmd()
			if err := cmd.Flags().Set("current-value", "12"); err != nil {
				t.Fatalf("failed to set flag: %v", err)
			}
			if tc.setTarget {
				if err := cmd.Flags().Set("target-value", "25"); err != nil {
					t.Fatalf("failed to set flag: %v", err)
				}
			}

			if err := runGoalMetricSet(cmd, []string{"123"}); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if sawPath != tc.expectPath {
				t.Fatalf("expected path %s, got %s", tc.expectPath, sawPath)
			}
			if tc.checkBody != nil {
				tc.checkBody(t, gotBody)
			}
		})
	}
}

func TestGoalMetricSetRejectsMetadataWithoutTarget(t *testing.T) {
	cmd, _ := newGoalMetricSetTestCmd()
	if err := cmd.Flags().Set("current-value", "5"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}
	if err := cmd.Flags().Set("unit", "currency"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	err := runGoalMetricSet(cmd, []string{"123"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "--unit, --precision, and --initial-value require --target-value") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func newGoalListTestCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&buf)
	cmd.Flags().String("workspace", "", "")
	cmd.Flags().String("workspace-gid", "", "")
	cmd.Flags().String("team", "", "")
	cmd.Flags().String("team-gid", "", "")
	cmd.Flags().String("time-period", "", "")
	cmd.Flags().Int("limit", 0, "")
	return cmd, &buf
}

func newGoalMetricSetTestCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&buf)
	cmd.Flags().Float64("current-value", 0, "")
	cmd.Flags().Float64("target-value", 0, "")
	cmd.Flags().Float64("initial-value", 0, "")
	cmd.Flags().String("unit", "", "")
	cmd.Flags().Int("precision", 0, "")
	cmd.Flags().Bool("dry-run", false, "")
	return cmd, &buf
}
