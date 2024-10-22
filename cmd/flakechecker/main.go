package main

import (
	_ "embed"
	"flag"
	"fmt"
	"github.com/carlmjohnson/versioninfo"
	junit "github.com/joshdk/go-junit"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stackrox/junit2jira/pkg/testcase"
	"os"
)

const minHistoricalRuns = 30

const errDescAboveThreshold = "flake ratio for test is above allowed threshold"
const errDescGetRatio = "retrieving ratio for test failed"
const errDescNoFailedTests = "no failed tests to process"
const errDescNoMatch = "test does not match any allowed flakes"
const errDescShortHistory = "not enough historical test runs to compute flakiness"

type flakeCheckerParams struct {
	junitReportsDir string
	configFile      string
	jobName         string
}

func (p *flakeCheckerParams) checkFailedTests(bqClient biqQueryClient, failedTests []testcase.TestCase, flakeCheckerRecs []*flakeDetectionPolicy) error {
	if len(failedTests) == 0 {
		return errors.New(errDescNoFailedTests)
	}

	for _, failedTest := range failedTests {
		log.Infof("Checking failed test: %q / %q / %q", p.jobName, failedTest.Classname, failedTest.Name)
		flakeCheckerRec, err := findFlakeConfigForTest(flakeCheckerRecs, p.jobName, failedTest.Classname, failedTest.Name)
		if err != nil {
			return err
		}

		log.Infof("Match found: %q / %q / %q", flakeCheckerRec.config.JobNameRegex, flakeCheckerRec.config.ClassName, flakeCheckerRec.config.TestNameRegex)
		totalRuns, failRatio, err := bqClient.GetRatioForTest(flakeCheckerRec.config, failedTest.Name)
		if err != nil {
			return errors.Wrap(err, errDescGetRatio)
		}

		if totalRuns < minHistoricalRuns {
			return errors.Wrap(fmt.Errorf("%d", totalRuns), errDescShortHistory)
		}

		if failRatio > flakeCheckerRec.config.RatioThreshold {
			return errors.Wrap(fmt.Errorf("(%d > %d)", failRatio, flakeCheckerRec.config.RatioThreshold), errDescAboveThreshold)
		}

		log.Infof("Failed test: %q / %q / %q - will be suppressed because failure reate is below allowed threshold: (%d <= %d)", p.jobName, failedTest.Classname, failedTest.Name, failRatio, flakeCheckerRec.config.RatioThreshold)
	}

	return nil
}

func (p *flakeCheckerParams) run() error {
	testSuites, err := junit.IngestDir(p.junitReportsDir)
	if err != nil {
		return errors.Wrap(err, "could not read files")
	}

	failedTests, err := testcase.GetFailedTests(testSuites)
	if err != nil {
		return errors.Wrap(err, "could not find failed tests")
	}
	log.Infof("Found %d failed tests", len(failedTests))

	flakeConfigs, err := loadFlakeConfigFile(p.configFile)
	if err != nil {
		return errors.Wrapf(err, "unable to load config file (%s)", p.configFile)
	}

	bqClient, err := getNewBigQueryClient()
	if err != nil {
		return errors.Wrap(err, "unable to create BigQuery client")
	}

	if err = p.checkFailedTests(bqClient, failedTests, flakeConfigs); err != nil {
		return errors.Wrap(err, "check failed tests")
	}

	log.Info("All failed tests are within allowed flake thresholds")
	return nil
}

func main() {
	p := flakeCheckerParams{}
	flag.StringVar(&p.junitReportsDir, "junit-reports-dir", os.Getenv("ARTIFACT_DIR"), "Directory containing JUnit report XML files.")
	flag.StringVar(&p.configFile, "config-file", "", "Config file with allowed flakes.")
	flag.StringVar(&p.jobName, "job-name", "", "Name of CI job.")

	var debug bool
	flag.BoolVar(&debug, "debug", false, "Enable debug log level.")
	versioninfo.AddFlag(flag.CommandLine)
	flag.Parse()

	if debug {
		log.SetLevel(log.DebugLevel)
	}

	if err := p.run(); err != nil {
		log.Fatal(err)
	}
}
