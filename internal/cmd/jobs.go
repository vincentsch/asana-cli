// Package cmd provides job polling helper for async operations.
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/vincentsch/asana-cli/internal/api"
)

// waitForJob polls /jobs/{gid} until status is succeeded or failed.
// Polls every 1 second with exponential backoff up to 5 seconds.
// Returns the completed Job on success, or error on failure/timeout.
func waitForJob(ctx context.Context, client *api.Client, jobGID string) (*api.Job, error) {
	const (
		initialInterval = 1 * time.Second
		maxInterval     = 5 * time.Second
		maxDuration     = 5 * time.Minute
	)

	deadline := time.Now().Add(maxDuration)
	interval := initialInterval

	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("job %s timed out after %v", jobGID, maxDuration)
		}

		job, err := getJob(ctx, client, jobGID)
		if err != nil {
			return nil, err
		}

		switch job.Status {
		case "succeeded":
			return job, nil
		case "failed":
			return nil, fmt.Errorf("job %s failed", jobGID)
		case "not_started", "in_progress":
			// Continue polling
		default:
			return nil, fmt.Errorf("job %s has unknown status: %s", jobGID, job.Status)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
			// Increase interval with exponential backoff
			interval = time.Duration(float64(interval) * 1.5)
			if interval > maxInterval {
				interval = maxInterval
			}
		}
	}
}

func getJob(ctx context.Context, client *api.Client, jobGID string) (*api.Job, error) {
	query := url.Values{}
	query.Set("opt_fields", "gid,status,new_task.gid,new_task.name,new_project.gid,new_project.name")

	data, err := client.Get(ctx, "/jobs/"+jobGID, query)
	if err != nil {
		return nil, err
	}

	var resp api.Response[api.Job]
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, &api.ResponseError{Err: err}
	}

	return &resp.Data, nil
}
