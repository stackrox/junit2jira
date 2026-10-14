package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPolicyAnchorsAlternatives(t *testing.T) {
	p, err := newFlakeDetectionPolicy(flakeDetectionPolicyConfig{JobNameRegex: "foo|bar", TestNameRegex: "foo|bar"})
	require.NoError(t, err)
	for _, name := range []string{"foo", "bar"} {
		require.True(t, p.matchJobName(name))
		require.True(t, p.matchTestName(name))
	}
	for _, name := range []string{"foo-unlisted", "unlisted-bar"} {
		require.False(t, p.matchJobName(name))
		require.False(t, p.matchTestName(name))
	}
	p, err = newFlakeDetectionPolicy(flakeDetectionPolicyConfig{JobNameRegex: ".*", TestNameRegex: ".*"})
	require.NoError(t, err)
	require.True(t, p.matchJobName("anything"))
	require.True(t, p.matchTestName("anything"))
	_, err = newFlakeDetectionPolicy(flakeDetectionPolicyConfig{JobNameRegex: ".*", TestNameRegex: "["})
	require.ErrorContains(t, err, "invalid flake config test name regex")
}

func TestLoadFailedTestsNormalizedIdentities(t *testing.T) {
	for _, tc := range []struct{ name, attrs, class, test string }{
		{"missing class", `name="test"`, "fallback-classname", "test"},
		{"missing name", `classname="class"`, "class", "fallback-name"},
		{"both missing", "", "fallback-classname", "fallback-name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			xml := `<testsuites><testsuite name="outer"><testsuite name="inner"><testcase ` + tc.attrs + `><failure message="failed"/></testcase><testcase classname="runtime.MemStats"><failure/></testcase></testsuite></testsuite></testsuites>`
			require.NoError(t, os.WriteFile(filepath.Join(dir, "report.xml"), []byte(xml), 0600))
			failed, err := loadFailedTests(dir)
			require.NoError(t, err)
			require.Len(t, failed, 1)
			require.Equal(t, tc.class, failed[0].Classname)
			require.Equal(t, tc.test, failed[0].Name)
			policy, err := newFlakeDetectionPolicy(flakeDetectionPolicyConfig{JobNameRegex: "job", ClassName: tc.class, TestNameRegex: tc.test, RatioThreshold: 5})
			require.NoError(t, err)
			calls := 0
			client := &mockBigQueryClient{getRatioForTest: func(config flakeDetectionPolicyConfig, name string) (int, int, error) {
				calls++
				require.Equal(t, tc.class, config.ClassName)
				require.Equal(t, tc.test, name)
				return minHistoricalRuns, 1, nil
			}}
			p := flakeCheckerParams{jobName: "job"}
			require.NoError(t, p.checkFailedTests(client, failed, []*flakeDetectionPolicy{policy}))
			require.Equal(t, 1, calls)
		})
	}
	_, err := loadFailedTests(filepath.Join(t.TempDir(), "missing"))
	require.ErrorContains(t, err, "could not read files")
}
