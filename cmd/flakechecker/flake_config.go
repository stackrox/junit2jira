package main

import (
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// flakeDetectionPolicyConfig represents configuration used by flakechecker to evaluate failed tests.
type flakeDetectionPolicyConfig struct {
	// JobNameRegex is a regular expression for the name of the CI job that should be evaluated by flakechecker.
	// (i.e. CI jobs for PRs should be evaluated, but not CI jobs for commits already merged to "main" branch)
	JobNameRegex string `yaml:"jobNameRegex"`
	// ClassName is class name of the test that should be isolated. Usually class name for Groovy tests,
	// package name for golang tests, etc.
	ClassName string `yaml:"className"`
	// TestNameRegex is a regular expression used to match test names. Some test names contain detailed information
	// (i.e. version 4.4.4), but we want to use ratio for all tests in that group (i.e. 4.4.z).
	// Using a regex allow us to group tests as needed.
	TestNameRegex string `yaml:"testNameRegex"`
	// TestNameRegex is CI job name that should be used for ratio calculation.
	// i.e. we take CI runs for commits on "main" branch as input for evaluation of flake ratio.
	RatioJobName string `yaml:"ratioJobName"`
	// RatioThreshold is the maximum failure percentage that is used to distinguish a flaky test from
	// a completely broken test. This information is usually fetched from historical executions and data
	// collected in DB. If measured flakiness exceeds this threshold, we no longer want to suppress test failure,
	// because we suspect it might have regressed above what we consider acceptable.
	RatioThreshold int `yaml:"ratioThreshold"`
}

type flakeDetectionPolicy struct {
	config                flakeDetectionPolicyConfig
	compiledJobNameRegex  *regexp.Regexp
	compiledTestNameRegex *regexp.Regexp
}

func newFlakeDetectionPolicy(config flakeDetectionPolicyConfig) (*flakeDetectionPolicy, error) {
	compiledJobNameRegex, err := regexp.Compile(fmt.Sprintf("^%s$", config.JobNameRegex))
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("invalid flake config match job regex: %v", config.JobNameRegex))
	}

	compiledTestNameRegex, err := regexp.Compile(fmt.Sprintf("^%s$", config.TestNameRegex))
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("invalid flake config test name regex: %v", config.TestNameRegex))
	}

	return &flakeDetectionPolicy{
		config:                config,
		compiledJobNameRegex:  compiledJobNameRegex,
		compiledTestNameRegex: compiledTestNameRegex,
	}, nil
}

func (r *flakeDetectionPolicy) matchJobName(jobName string) bool {
	return r.compiledJobNameRegex.MatchString(jobName)
}

func (r *flakeDetectionPolicy) matchClassName(classname string) bool {
	return classname == r.config.ClassName
}

func (r *flakeDetectionPolicy) matchTestName(testName string) bool {
	return r.compiledTestNameRegex.MatchString(testName)
}

func findFlakeConfigForTest(flakeCheckerRecs []*flakeDetectionPolicy, jobName string, className string, testName string) (*flakeDetectionPolicy, error) {
	for _, flakeCheckerRec := range flakeCheckerRecs {
		if flakeCheckerRec.matchJobName(jobName) && flakeCheckerRec.matchClassName(className) && flakeCheckerRec.matchTestName(testName) {
			return flakeCheckerRec, nil
		}
	}

	return nil, errors.Wrap(errors.Errorf("%q / %q / %q", jobName, className, testName), errDescNoMatch)
}

func loadFlakeConfigFile(fileName string) ([]*flakeDetectionPolicy, error) {
	ymlConfigFile, err := os.Open(fileName)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("open flake config file: %s", fileName))
	}
	defer ymlConfigFile.Close() //nolint:errcheck

	ymlConfigFileData, err := io.ReadAll(ymlConfigFile)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("read flake config file: %s", fileName))
	}

	flakeConfigs := make([]flakeDetectionPolicyConfig, 0)
	err = yaml.Unmarshal(ymlConfigFileData, &flakeConfigs)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("parse flake config file: %s", fileName))
	}

	detectionPolicies := make([]*flakeDetectionPolicy, 0, len(flakeConfigs))
	for _, flakeConfig := range flakeConfigs {
		detectionPolicy, errNewPolicy := newFlakeDetectionPolicy(flakeConfig)
		if errNewPolicy != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("create flake detection policy from config: %v", flakeConfig))
		}

		detectionPolicies = append(detectionPolicies, detectionPolicy)
	}

	return detectionPolicies, nil
}
