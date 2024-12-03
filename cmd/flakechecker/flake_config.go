package main

import (
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"regexp"
)

// flakeDetectionPolicyConfig represents configuration used by flakechecker to evaluate failed tests.
//
// It contains the following fields:
// match_job_name - name of the job that should be evaluated by flakechecker. i.e. (branch should be evaluated, but main not)
// ratio_job_name - job name that should be used for ratio calculation. i.e. we take main branch test runs as base for evaluation of flake ratio
// test_name_regex - regex used to match test names. Some test names contain detailed information (i.e. version 4.4.4), but we want to use ratio for all tests in that group (i.e. 4.4.z). Using regex allow us to group tests differently.
// classname - class name of the test that should be isolated. With this option we can isolate single flake test from suite and isolate only that one from the rest.
// ratio_threshold - failure percentage that is allowed for this test. This information is usually fetched from historical executions and data collected in DB.
type flakeDetectionPolicyConfig struct {
	MatchJobName   string `yaml:"matchJobName"`
	RatioJobName   string `yaml:"ratioJobName"`
	TestNameRegex  string `yaml:"testNameRegex"`
	Classname      string `yaml:"classname"`
	RatioThreshold int    `yaml:"ratioThreshold"`
}

type flakeDetectionPolicy struct {
	config             *flakeDetectionPolicyConfig
	regexMatchJobName  *regexp.Regexp
	regexTestNameRegex *regexp.Regexp
}

func newFlakeDetectionPolicy(config *flakeDetectionPolicyConfig) (*flakeDetectionPolicy, error) {
	regexMatchJobName, err := regexp.Compile(fmt.Sprintf("^%s$", config.MatchJobName))
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("invalid flake config match job regex: %v", config.MatchJobName))
	}

	regexTestNameRegex, err := regexp.Compile(fmt.Sprintf("^%s$", config.TestNameRegex))
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("invalid flake config test name regex: %v", config.TestNameRegex))
	}

	return &flakeDetectionPolicy{
		config:             config,
		regexMatchJobName:  regexMatchJobName,
		regexTestNameRegex: regexTestNameRegex,
	}, nil
}

// newFlakeDetectionPolicyMust - is primarily  used in tests.
func newFlakeDetectionPolicyMust(config *flakeDetectionPolicyConfig) *flakeDetectionPolicy {
	policy, err := newFlakeDetectionPolicy(config)
	if err != nil {
		panic(err)
	}

	return policy
}

func (r *flakeDetectionPolicy) matchJobName(jobName string) (bool, error) {
	return r.regexMatchJobName.MatchString(jobName), nil
}

func (r *flakeDetectionPolicy) matchTestName(testName string) (bool, error) {
	return r.regexTestNameRegex.MatchString(testName), nil
}

func (r *flakeDetectionPolicy) matchClassname(classname string) (bool, error) {
	return classname == r.config.Classname, nil
}

func loadFlakeConfigFile(fileName string) ([]*flakeDetectionPolicy, error) {
	ymlConfigFile, err := os.Open(fileName)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("open flake config file: %s", fileName))
	}
	defer ymlConfigFile.Close()

	ymlConfigFileData, err := io.ReadAll(ymlConfigFile)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("read flake config file: %s", fileName))
	}

	flakeConfigs := make([]*flakeDetectionPolicyConfig, 0)
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
