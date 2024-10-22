package main

import (
	"github.com/pkg/errors"
	"github.com/stackrox/junit2jira/pkg/testcase"
	"github.com/stretchr/testify/assert"
	"testing"
)

type biqQueryClientMock struct {
	getRatioForTest func(flakeTestConfig *flakeDetectionPolicy, testName string) (int, int, error)
}

func (c *biqQueryClientMock) GetRatioForTest(flakeTestConfig *flakeDetectionPolicy, testName string) (int, int, error) {
	if c.getRatioForTest != nil {
		return c.getRatioForTest(flakeTestConfig, testName)
	}

	// By default, fail. In most cases, we will not reach this part in tests.
	return 0, 0, errors.New("fail")
}

func getRatioForTestNoFailures(_ *flakeDetectionPolicy, _ string) (int, int, error) {
	return totalRunsLimit, 0, nil
}

func getRatioForTestAllFailures(_ *flakeDetectionPolicy, _ string) (int, int, error) {
	return totalRunsLimit, 50, nil
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
			bqClient:    &biqQueryClientMock{},
			failedTests: []testcase.TestCase{},
			flakeDetectionPolicies: []*flakeDetectionPolicy{
				newFlakeDetectionPolicyMust(&flakeDetectionPolicyConfig{MatchJobName: "test-job1-", TestNameRegex: "test name"}),
			},
			expectError:    true,
			expectErrorStr: errDescNoFailedTests,
		},
		"no config match - job name": {
			bqClient: &biqQueryClientMock{
				getRatioForTest: getRatioForTestNoFailures,
			},
			failedTests: []testcase.TestCase{{Name: "test", Classname: "class"}},
			flakeDetectionPolicies: []*flakeDetectionPolicy{
				newFlakeDetectionPolicyMust(&flakeDetectionPolicyConfig{MatchJobName: "test-job-1", TestNameRegex: "test", Classname: "class"}),
			},
			expectError:    true,
			expectErrorStr: errDescNoMatch,
		},
		"no config match - test name": {
			bqClient: &biqQueryClientMock{
				getRatioForTest: getRatioForTestNoFailures,
			},
			failedTests: []testcase.TestCase{{Name: "test", Classname: "class"}},
			flakeDetectionPolicies: []*flakeDetectionPolicy{
				newFlakeDetectionPolicyMust(&flakeDetectionPolicyConfig{MatchJobName: "test-job", TestNameRegex: "wrong-test", Classname: "class"}),
			},
			expectError:    true,
			expectErrorStr: errDescNoMatch,
		},
		"no config match - classname": {
			bqClient: &biqQueryClientMock{
				getRatioForTest: getRatioForTestNoFailures,
			},
			failedTests: []testcase.TestCase{{Name: "test", Classname: "class"}},
			flakeDetectionPolicies: []*flakeDetectionPolicy{
				newFlakeDetectionPolicyMust(&flakeDetectionPolicyConfig{MatchJobName: "test-job", TestNameRegex: "test", Classname: "wrong-class"}),
			},
			expectError:    true,
			expectErrorStr: errDescNoMatch,
		},
		"unable to fetch ratio": {
			bqClient: &biqQueryClientMock{
				getRatioForTest: func(_ *flakeDetectionPolicy, _ string) (int, int, error) {
					return 0, 0, errors.New("fail")
				},
			},
			failedTests: []testcase.TestCase{{Name: "test", Classname: "class"}},
			flakeDetectionPolicies: []*flakeDetectionPolicy{
				newFlakeDetectionPolicyMust(&flakeDetectionPolicyConfig{MatchJobName: "test-job", TestNameRegex: "test", Classname: "class", RatioThreshold: 1}),
			},
			expectError:    true,
			expectErrorStr: errDescGetRatio,
		},
		"total runs below limit": {
			bqClient: &biqQueryClientMock{
				getRatioForTest: func(_ *flakeDetectionPolicy, _ string) (int, int, error) {
					return totalRunsLimit - 1, 0, nil
				},
			},
			failedTests: []testcase.TestCase{{Name: "test", Classname: "class"}},
			flakeDetectionPolicies: []*flakeDetectionPolicy{
				newFlakeDetectionPolicyMust(&flakeDetectionPolicyConfig{MatchJobName: "test-job", TestNameRegex: "test", Classname: "class", RatioThreshold: 1}),
			},
			expectError:    true,
			expectErrorStr: errDescShortHistory,
		},
		"fail ratio below threshold": {
			bqClient: &biqQueryClientMock{
				getRatioForTest: getRatioForTestNoFailures,
			},
			failedTests: []testcase.TestCase{{Name: "test", Classname: "class"}},
			flakeDetectionPolicies: []*flakeDetectionPolicy{
				newFlakeDetectionPolicyMust(&flakeDetectionPolicyConfig{MatchJobName: "test-job", TestNameRegex: "test", Classname: "class", RatioThreshold: 1}),
			},
			expectError: false,
		},
		"fail ratio above threshold": {
			bqClient: &biqQueryClientMock{
				getRatioForTest: getRatioForTestAllFailures,
			},
			failedTests: []testcase.TestCase{{Name: "test", Classname: "class"}},
			flakeDetectionPolicies: []*flakeDetectionPolicy{
				newFlakeDetectionPolicyMust(&flakeDetectionPolicyConfig{MatchJobName: "test-job", TestNameRegex: "test", Classname: "class", RatioThreshold: 1}),
			},
			expectError:    true,
			expectErrorStr: errDescAboveThreshold,
		},
		"fail ratio below threshold - multiple tests": {
			bqClient: &biqQueryClientMock{
				getRatioForTest: getRatioForTestNoFailures,
			},
			failedTests: []testcase.TestCase{
				{Name: "test", Classname: "class"},
				{Name: "test-1", Classname: "class-1"},
			},
			flakeDetectionPolicies: []*flakeDetectionPolicy{
				newFlakeDetectionPolicyMust(&flakeDetectionPolicyConfig{MatchJobName: "test-job", TestNameRegex: "test", Classname: "class", RatioThreshold: 1}),
				newFlakeDetectionPolicyMust(&flakeDetectionPolicyConfig{MatchJobName: "test-job", TestNameRegex: "test-1", Classname: "class-1", RatioThreshold: 1}),
			},
			expectError: false,
		},
		"fail ratio above threshold - multiple tests": {
			bqClient: &biqQueryClientMock{
				getRatioForTest: getRatioForTestAllFailures,
			},
			failedTests: []testcase.TestCase{
				{Name: "test-ratio-below", Classname: "class"},
				{Name: "test-ratio-above", Classname: "class"},
			},
			flakeDetectionPolicies: []*flakeDetectionPolicy{
				newFlakeDetectionPolicyMust(&flakeDetectionPolicyConfig{MatchJobName: "test-job", TestNameRegex: "test-ratio-below", Classname: "class", RatioThreshold: 90}),
				newFlakeDetectionPolicyMust(&flakeDetectionPolicyConfig{MatchJobName: "test-job", TestNameRegex: "test-ratio-above", Classname: "class", RatioThreshold: 10}),
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
