package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	jira "github.com/ctreminiom/go-atlassian/v2/jira/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePriorityThresholds(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    []int
		expectError bool
	}{
		{
			name:     "valid default thresholds",
			input:    priorityThresholdsString(defaultPriorityThresholds),
			expected: defaultPriorityThresholds,
		},
		{
			name:     "valid custom thresholds",
			input:    "10,50,100,200,400",
			expected: []int{10, 50, 100, 200, 400},
		},
		{
			name:     "valid with spaces",
			input:    " 4 , 16 , 64 , 128 , 256 ",
			expected: []int{4, 16, 64, 128, 256},
		},
		{
			name:        "invalid - too few values",
			input:       "4,16,64",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "invalid - too many values",
			input:       "4,16,64,128,256,512",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "invalid - non-numeric value",
			input:       "4,16,abc,128,256",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "negative value",
			input:       "-1,16,64,128,256",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "duplicate values",
			input:       "4,16,16,128,256",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "descending values",
			input:       "256,128,64,16,4",
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsePriorityThresholds(tt.input)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

type priorityRoundTripper func(*http.Request) (*http.Response, error)

func (f priorityRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestUpdatePriorityIfNeeded(t *testing.T) {
	for _, test := range []struct {
		name         string
		initialTotal int
		current      string
		dryRun       bool
		empty        bool
		wantUpdate   bool
	}{
		{name: "all pages and selected window", initialTotal: 4, current: "Undefined", wantUpdate: true},
		{name: "total grows", initialTotal: 1, current: "Undefined", wantUpdate: true},
		{name: "initial total is stale zero", current: "Undefined", wantUpdate: true},
		{name: "preserve Critical", initialTotal: 4, current: "Critical"},
		{name: "preserve Blocker", initialTotal: 4, current: "Blocker"},
		{name: "unknown priority", initialTotal: 4, current: "Custom"},
		{name: "dry run", initialTotal: 4, current: "Undefined", dryRun: true},
		{name: "no comments", current: "Undefined", empty: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			type contextKey struct{}
			ctx := context.WithValue(context.Background(), contextKey{}, "caller")
			var offsets []string
			updates := 0
			client, err := jira.New(&http.Client{Transport: priorityRoundTripper(func(req *http.Request) (*http.Response, error) {
				require.Equal(t, "caller", req.Context().Value(contextKey{}), "caller context must reach every Jira request")
				var body any
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/rest/api/3/issue/TEST-1":
					body = map[string]any{"fields": map[string]any{"priority": map[string]string{"name": test.current}, "comment": map[string]any{"total": test.initialTotal, "comments": []any{}}}}
				case req.Method == http.MethodGet && req.URL.Path == "/rest/api/3/issue/TEST-1/comment":
					offset := req.URL.Query().Get("startAt")
					offsets = append(offsets, offset)
					require.Equal(t, "50", req.URL.Query().Get("maxResults"))
					require.Equal(t, "created", req.URL.Query().Get("orderBy"))
					created := time.Now().AddDate(0, 0, -60).Format(time.RFC3339)
					if offset == "2" {
						created = time.Now().Add(-time.Hour).Format(time.RFC3339)
					} else {
						require.Equal(t, "0", offset)
					}
					body = map[string]any{"total": 4, "comments": []any{map[string]string{"created": created}, map[string]string{"created": created}}}
					if test.empty {
						body = map[string]any{"total": 0, "comments": []any{}}
					}
				case req.Method == http.MethodPut && req.URL.Path == "/rest/api/3/issue/TEST-1":
					updates++
					payload, err := io.ReadAll(req.Body)
					require.NoError(t, err)
					require.JSONEq(t, `{"fields":{"priority":{"name":"Minor"}}}`, string(payload))
					return &http.Response{Request: req, StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL)
				}
				data, err := json.Marshal(body)
				require.NoError(t, err)
				return &http.Response{Request: req, StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(string(data))), Header: http.Header{"Content-Type": {"application/json"}}}, nil
			})}, "https://jira.example/")
			require.NoError(t, err)
			j := junit2jira{params: params{enableAutoPriority: true, priorityWindowDays: 30, parsedPriorityThresholds: []int{2, 3, 4, 5, 6}, dryRun: test.dryRun}, jiraClient: client}
			require.NoError(t, j.updatePriorityIfNeeded(ctx, "TEST-1"))
			if test.empty {
				require.Equal(t, []string{"0"}, offsets)
			} else {
				require.Equal(t, []string{"0", "2"}, offsets)
			}
			if test.wantUpdate {
				require.Equal(t, 1, updates)
			} else {
				require.Zero(t, updates)
			}
		})
	}
}

func TestUpdatePriorityIfNeededCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client, err := jira.New(&http.Client{Transport: priorityRoundTripper(func(req *http.Request) (*http.Response, error) {
		require.ErrorIs(t, req.Context().Err(), context.Canceled)
		return nil, req.Context().Err()
	})}, "https://jira.example/")
	require.NoError(t, err)
	j := junit2jira{params: params{enableAutoPriority: true, priorityWindowDays: 30, parsedPriorityThresholds: defaultPriorityThresholds}, jiraClient: client}
	require.ErrorIs(t, j.updatePriorityIfNeeded(ctx, "TEST-1"), context.Canceled)
}

func TestRunRejectsInvalidPriorityConfiguration(t *testing.T) {
	for _, test := range []struct {
		name       string
		window     int
		thresholds string
		wantError  string
	}{
		{"invalid window", 0, "2,10,50,100,200", "priority window days must be positive"},
		{"invalid thresholds", 30, "2,10,10,100,200", "invalid priority thresholds"},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.ErrorContains(t, run(context.Background(), params{priorityWindowDays: test.window, priorityThresholds: test.thresholds}), test.wantError)
		})
	}
}
