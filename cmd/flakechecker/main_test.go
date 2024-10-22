package main

import (
	"github.com/pkg/errors"
	"github.com/stackrox/junit2jira/pkg/testcase"
	"github.com/stretchr/testify/assert"
	"testing"
)

type mockBigQueryClient struct {
	getRatioForTest func(config flakeDetectionPolicyConfig, testName string) (int, int, error)
}

func (c *mockBigQueryClient) GetRatioForTest(config flakeDetectionPolicyConfig, testName string) (int, int, error) {
	if c.getRatioForTest != nil {
		return c.getRatioForTest(config, testName)
	}

	// By default, fail. In most cases, we will not reach this part in tests.
	return 0, 0, errors.New("fail")
}

func getRatioForTestNoFailures(_ flakeDetectionPolicyConfig, _ string) (int, int, error) {
	return minHistoricalRuns, 0, nil
}

func getRatioForTestAllFailures(_ flakeDetectionPolicyConfig, _ string) (int, int, error) {
	return minHistoricalRuns, 50, nil
}

func TestCheckFailedTests(t *testing.T) {
	p := flakeCheckerParams{jobName: "test-job"}

	samples := map[string]struct {
		bqClient               biqQueryClient
		failedTests            []testcase.TestCase
		flakeDetectionPolicies []*flakeDetectionPolicy
		expectError            bool
		expectErrorStr         string
	}{
		"no failed tests": {
			bqClient:    &mockBigQueryClient{},
			failedTests: []testcase.TestCase{},
			flakeDetectionPolicies: []*flakeDetectionPolicy{
				newFlakeDetectionPolicyMust(flakeDetectionPolicyConfig{JobNameRegex: "test-job1-", TestNameRegex: "test name"}),
			},
			expectError:    true,
			expectErrorStr: errDescNoFailedTests,
		},
		"no config match - job name": {
			bqClient: &mockBigQueryClient{
				getRatioForTest: getRatioForTestNoFailures,
			},
			failedTests: []testcase.TestCase{{Classname: "class", Name: "test"}},
			flakeDetectionPolicies: []*flakeDetectionPolicy{
				newFlakeDetectionPolicyMust(flakeDetectionPolicyConfig{JobNameRegex: "test-job-1", ClassName: "class", TestNameRegex: "test"}),
			},
			expectError:    true,
			expectErrorStr: errDescNoMatch,
		},
		"no config match - test name": {
			bqClient: &mockBigQueryClient{
				getRatioForTest: getRatioForTestNoFailures,
			},
			failedTests: []testcase.TestCase{{Classname: "class", Name: "test"}},
			flakeDetectionPolicies: []*flakeDetectionPolicy{
				newFlakeDetectionPolicyMust(flakeDetectionPolicyConfig{JobNameRegex: "test-job", ClassName: "class", TestNameRegex: "wrong-test"}),
			},
			expectError:    true,
			expectErrorStr: errDescNoMatch,
		},
		"no config match - class name": {
			bqClient: &mockBigQueryClient{
				getRatioForTest: getRatioForTestNoFailures,
			},
			failedTests: []testcase.TestCase{{Classname: "class", Name: "test"}},
			flakeDetectionPolicies: []*flakeDetectionPolicy{
				newFlakeDetectionPolicyMust(flakeDetectionPolicyConfig{JobNameRegex: "test-job", ClassName: "wrong-class", TestNameRegex: "test"}),
			},
			expectError:    true,
			expectErrorStr: errDescNoMatch,
		},
		"unable to fetch ratio": {
			bqClient: &mockBigQueryClient{
				getRatioForTest: func(_ flakeDetectionPolicyConfig, _ string) (int, int, error) {
					return 0, 0, errors.New("fail")
				},
			},
			failedTests: []testcase.TestCase{{Classname: "class", Name: "test"}},
			flakeDetectionPolicies: []*flakeDetectionPolicy{
				newFlakeDetectionPolicyMust(flakeDetectionPolicyConfig{JobNameRegex: "test-job", ClassName: "class", TestNameRegex: "test", RatioThreshold: 1}),
			},
			expectError:    true,
			expectErrorStr: errDescGetRatio,
		},
		"total runs below limit": {
			bqClient: &mockBigQueryClient{
				getRatioForTest: func(_ flakeDetectionPolicyConfig, _ string) (int, int, error) {
					return minHistoricalRuns - 1, 0, nil
				},
			},
			failedTests: []testcase.TestCase{{Classname: "class", Name: "test"}},
			flakeDetectionPolicies: []*flakeDetectionPolicy{
				newFlakeDetectionPolicyMust(flakeDetectionPolicyConfig{JobNameRegex: "test-job", ClassName: "class", TestNameRegex: "test", RatioThreshold: 1}),
			},
			expectError:    true,
			expectErrorStr: errDescShortHistory,
		},
		"fail ratio below threshold": {
			bqClient: &mockBigQueryClient{
				getRatioForTest: getRatioForTestNoFailures,
			},
			failedTests: []testcase.TestCase{{Classname: "class", Name: "test"}},
			flakeDetectionPolicies: []*flakeDetectionPolicy{
				newFlakeDetectionPolicyMust(flakeDetectionPolicyConfig{JobNameRegex: "test-job", ClassName: "class", TestNameRegex: "test", RatioThreshold: 1}),
			},
			expectError: false,
		},
		"fail ratio above threshold": {
			bqClient: &mockBigQueryClient{
				getRatioForTest: getRatioForTestAllFailures,
			},
			failedTests: []testcase.TestCase{{Classname: "class", Name: "test"}},
			flakeDetectionPolicies: []*flakeDetectionPolicy{
				newFlakeDetectionPolicyMust(flakeDetectionPolicyConfig{JobNameRegex: "test-job", ClassName: "class", TestNameRegex: "test", RatioThreshold: 1}),
			},
			expectError:    true,
			expectErrorStr: errDescAboveThreshold,
		},
		"fail ratio below threshold - multiple tests": {
			bqClient: &mockBigQueryClient{
				getRatioForTest: getRatioForTestNoFailures,
			},
			failedTests: []testcase.TestCase{
				{Classname: "class", Name: "test"},
				{Classname: "class-1", Name: "test-1"},
			},
			flakeDetectionPolicies: []*flakeDetectionPolicy{
				newFlakeDetectionPolicyMust(flakeDetectionPolicyConfig{JobNameRegex: "test-job", ClassName: "class", TestNameRegex: "test", RatioThreshold: 1}),
				newFlakeDetectionPolicyMust(flakeDetectionPolicyConfig{JobNameRegex: "test-job", ClassName: "class-1", TestNameRegex: "test-1", RatioThreshold: 1}),
			},
			expectError: false,
		},
		"fail ratio above threshold - multiple tests": {
			bqClient: &mockBigQueryClient{
				getRatioForTest: getRatioForTestAllFailures,
			},
			failedTests: []testcase.TestCase{
				{Classname: "class", Name: "test-ratio-below"},
				{Classname: "class", Name: "test-ratio-above"},
			},
			flakeDetectionPolicies: []*flakeDetectionPolicy{
				newFlakeDetectionPolicyMust(flakeDetectionPolicyConfig{JobNameRegex: "test-job", ClassName: "class", TestNameRegex: "test-ratio-below", RatioThreshold: 90}),
				newFlakeDetectionPolicyMust(flakeDetectionPolicyConfig{JobNameRegex: "test-job", ClassName: "class", TestNameRegex: "test-ratio-above", RatioThreshold: 10}),
			},
			expectError:    true,
			expectErrorStr: errDescAboveThreshold,
		},
	}

	for sampleName, sample := range samples {
		t.Run(sampleName, func(tt *testing.T) {
			err := p.checkFailedTests(sample.bqClient, sample.failedTests, sample.flakeDetectionPolicies)

			if sample.expectError {
				assert.ErrorContains(tt, err, sample.expectErrorStr)
			} else {
				assert.NoError(tt, err)
			}
		})
	}
}
