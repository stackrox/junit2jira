package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func newFlakeDetectionPolicyMust(config flakeDetectionPolicyConfig) *flakeDetectionPolicy {
	policy, err := newFlakeDetectionPolicy(config)
	if err != nil {
		panic(err)
	}

	return policy
}

func TestLoadFlakeConfigFile(t *testing.T) {
	samples := []struct {
		name     string
		fileName string

		expectError    bool
		expectErrorStr string
		expectConfig   []*flakeDetectionPolicy
	}{
		{
			name:           "no config file",
			fileName:       "no_config.yml",
			expectError:    true,
			expectErrorStr: "open flake config file: no_config.yml: open no_config.yml: no such file or directory",
			expectConfig:   nil,
		},
		{
			name:        "valid config file",
			fileName:    "testdata/flake-config.yml",
			expectError: false,
			expectConfig: []*flakeDetectionPolicy{
				newFlakeDetectionPolicyMust(flakeDetectionPolicyConfig{
					JobNameRegex:   "pr-.*",
					ClassName:      "TestLoadFlakeConfigFile",
					TestNameRegex:  "TestLoadFlakeConf.*",
					RatioJobName:   "main-branch-tests",
					RatioThreshold: 5,
				}),
				newFlakeDetectionPolicyMust(flakeDetectionPolicyConfig{
					JobNameRegex:   "pull-request-tests",
					ClassName:      "TestLoadFlakeConfigFile",
					TestNameRegex:  "TestLoadFlakeConfigFile",
					RatioJobName:   "main-branch-tests",
					RatioThreshold: 10,
				}),
			},
		},
	}

	for _, sample := range samples {
		t.Run(sample.name, func(tt *testing.T) {
			config, err := loadFlakeConfigFile(sample.fileName)

			if sample.expectError {
				assert.EqualError(tt, err, sample.expectErrorStr)
			} else {
				assert.NoError(tt, err)
			}
			assert.Equal(tt, sample.expectConfig, config)
		})
	}
}
