package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	jira "github.com/ctreminiom/go-atlassian/v2/jira/v3"
	"github.com/stretchr/testify/require"
)

func TestFindMostRecentClosedIssue(t *testing.T) {
	for _, tc := range []struct {
		name  string
		pages []string
		want  string
		err   string
	}{
		{"same page", []string{`{"issues":[{"key":"similar","fields":{"summary":"target extra"}},{"key":"newest","fields":{"summary":"target"}},{"key":"older","fields":{"summary":"target"}}],"nextPageToken":"next"}`}, "newest", ""},
		{"later page", []string{`{"issues":[{"fields":{"summary":"similar"}}],"nextPageToken":"next"}`, `{"issues":[{"key":"exact","fields":{"summary":"target"}}]}`}, "exact", ""},
		{"exhausted", []string{`{"issues":[],"nextPageToken":"next"}`, `{"issues":[]}`}, "", ""},
		{"nil response", []string{`null`}, "", ""},
		{"later error", []string{`{"issues":[],"nextPageToken":"next"}`, `ERROR`}, "", "HTTP 500"},
		{"repeated token", []string{`{"issues":[],"nextPageToken":"next"}`, `{"issues":[],"nextPageToken":"next"}`}, "", "repeated"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			type key struct{}
			ctx := context.WithValue(context.Background(), key{}, "caller")
			calls := 0
			client, err := jira.New(&http.Client{Transport: priorityRoundTripper(func(req *http.Request) (*http.Response, error) {
				require.Equal(t, "caller", req.Context().Value(key{}))
				require.Equal(t, "POST", req.Method)
				require.Equal(t, "/rest/api/3/search/jql", req.URL.Path)
				var payload struct {
					MaxResults    int    `json:"maxResults"`
					NextPageToken string `json:"nextPageToken"`
					JQL           string `json:"jql"`
				}
				require.NoError(t, json.NewDecoder(req.Body).Decode(&payload))
				require.Equal(t, 50, payload.MaxResults)
				require.Contains(t, payload.JQL, "ORDER BY updated DESC")
				if calls == 0 {
					require.Empty(t, payload.NextPageToken)
				} else {
					require.Equal(t, "next", payload.NextPageToken)
				}
				require.Less(t, calls, len(tc.pages))
				body := tc.pages[calls]
				calls++
				status := 200
				if body == "ERROR" {
					status = 500
					body = `{"errorMessages":["failure"]}`
				}
				return &http.Response{Request: req, StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
			})}, "https://jira.example/")
			require.NoError(t, err)
			j := junit2jira{jiraClient: client, params: params{jiraProject: "TEST"}}
			issue, err := j.findMostRecentClosedIssue(ctx, "target")
			if tc.err != "" {
				require.ErrorContains(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}
			if tc.want == "" {
				require.Nil(t, issue)
			} else {
				require.NotNil(t, issue)
				require.Equal(t, tc.want, issue.Key)
			}
			require.Equal(t, len(tc.pages), calls)
		})
	}
}

func TestFindMostRecentClosedIssueCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client, err := jira.New(&http.Client{Transport: priorityRoundTripper(func(req *http.Request) (*http.Response, error) { return nil, req.Context().Err() })}, "https://jira.example/")
	require.NoError(t, err)
	j := junit2jira{jiraClient: client, params: params{jiraProject: "TEST"}}
	_, err = j.findMostRecentClosedIssue(ctx, "target")
	require.ErrorIs(t, err, context.Canceled)
}
