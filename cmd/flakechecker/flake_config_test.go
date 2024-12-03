package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLoadFlakeConfigFile(t *testing.T) {
	samples := []struct {
		name     string
		fileName string

		expectError  bool
		expectConfig []*flakeDetectionPolicy
	}{
		{
			name:         "no config file",
			fileName:     "no_config.yml",
			expectError:  true,
			expectConfig: nil,
		},
		{
			name:        "valid config file",
			fileName:    "testdata/flake-config.yml",
			expectError: false,
			expectConfig: []*flakeDetectionPolicy{
				newFlakeDetectionPolicyMust(&flakeDetectionPolicyConfig{
					MatchJobName:   "pr-.*",
					RatioJobName:   "main-branch-tests",
					TestNameRegex:  "TestLoadFlakeConf.*",
					Classname:      "TestLoadFlakeConfigFile",
					RatioThreshold: 5,
				}),
				newFlakeDetectionPolicyMust(&flakeDetectionPolicyConfig{
					MatchJobName:   "pull-request-tests",
					RatioJobName:   "main-branch-tests",
					TestNameRegex:  "TestLoadFlakeConfigFile",
					Classname:      "TestLoadFlakeConfigFile",
					RatioThreshold: 10,
				}),
			},
		},
	}

	for _, sample := range samples {
		t.Run(sample.name, func(tt *testing.T) {
			config, err := loadFlakeConfigFile(sample.fileName)

			assert.Equal(tt, sample.expectError, err != nil)
			assert.Equal(tt, sample.expectConfig, config)
		})
	}
}
