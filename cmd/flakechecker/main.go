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

const totalRunsLimit = 30

const errDescNoMatch = "there is no match in allowed flake tests"
const errDescAboveThreshold = "allowed flake ratio for test is above threshold"
const errDescShortHistory = "total runs for test is under history count threshold"
const errDescGetRatio = "get ratio for test failed"

type flakeCheckerParams struct {
	junitReportsDir string
	configFile      string

	jobName      string
	orchestrator string

	dryRun bool
}

func main() {
	var debug bool
	var err error

	p := flakeCheckerParams{}
	flag.StringVar(&p.junitReportsDir, "junit-reports-dir", os.Getenv("ARTIFACT_DIR"), "Dir that contains jUnit reports XML files")
	flag.StringVar(&p.configFile, "config-file", "", "Config file with defined failure ratios")

	flag.StringVar(&p.jobName, "job-name", "", "Name of CI job.")
	flag.StringVar(&p.orchestrator, "orchestrator", "", "orchestrator name (such as GKE or OpenShift), if any.")

	flag.BoolVar(&p.dryRun, "dry-run", false, "When set to true issues will NOT be created.")
	flag.BoolVar(&debug, "debug", false, "Enable debug log level")
	versioninfo.AddFlag(flag.CommandLine)
	flag.Parse()

	if debug {
		log.SetLevel(log.DebugLevel)
	}

	err = p.run()
	if err != nil {
		log.Fatal(err)
	}
}

type recentFlakyTestInfo struct {
	JobName      string
	FilteredName string
	Classname    string
	TotalAll     int
	FailRatio    int
}

func (p *flakeCheckerParams) checkFailedTests(bqClient biqQueryClient, failedTests []testcase.TestCase, flakeCheckerRecs []*flakeDetectionPolicy) error {
	for _, failedTest := range failedTests {
		found := false
		log.Infof("Checking failed test: %q / %q / %q", p.jobName, failedTest.Name, failedTest.Classname)
		for _, flakeCheckerRec := range flakeCheckerRecs {
			match, err := flakeCheckerRec.matchJobName(p.jobName)
			if err != nil {
				return err
			}

			if !match {
				continue
			}

			match, err = flakeCheckerRec.matchTestName(failedTest.Name)
			if err != nil {
				return err
			}

			if !match {
				continue
			}

			match, err = flakeCheckerRec.matchClassname(failedTest.Classname)
			if err != nil {
				return err
			}

			if !match {
				continue
			}

			found = true
			log.Infof("Match found: %q / %q / %q", flakeCheckerRec.config.MatchJobName, flakeCheckerRec.config.TestNameRegex, flakeCheckerRec.config.Classname)
			totalRuns, failRatio, err := bqClient.GetRatioForTest(flakeCheckerRec, failedTest.Name)
			if err != nil {
				return errors.Wrap(err, errDescGetRatio)
			}

			if totalRuns < totalRunsLimit {
				return errors.Wrap(fmt.Errorf("%d", totalRuns), errDescShortHistory)
			}

			if failRatio > flakeCheckerRec.config.RatioThreshold {
				return errors.Wrap(fmt.Errorf("(%d > %d)", failRatio, flakeCheckerRec.config.RatioThreshold), errDescAboveThreshold)
			}

			log.Infof("Ratio is below threshold: (%d <= %d)", failRatio, flakeCheckerRec.config.RatioThreshold)
		}

		if !found {
			return errors.Wrap(errors.New(failedTest.Name), errDescNoMatch)
		}
	}

	return nil
}

func (p *flakeCheckerParams) run() error {
	testSuites, err := junit.IngestDir(p.junitReportsDir)
	if err != nil {
		log.Fatalf("could not read files: %s", err)
	}

	failedTests, err := testcase.GetFailedTests(testSuites)
	if err != nil {
		return errors.Wrap(err, "could not find failed tests")
	}

	if len(failedTests) == 0 {
		log.Info("No failed tests to process")
		return nil
	}

	log.Infof("Found %d failed tests", len(failedTests))

	flakeConfigs, err := loadFlakeConfigFile(p.configFile)
	if err != nil {
		log.Fatalf("unable to load config file (%s): %s", p.configFile, err)
	}

	bqClient, err := getNewBigQueryClient()
	if err != nil {
		log.Fatalf("unable to create BigQuery client: %s", err)
	}

	if err = p.checkFailedTests(bqClient, failedTests, flakeConfigs); err != nil {
		log.Fatal(err)
	}

	log.Info("All failed tests are within allowed flake thresholds")

	return nil
}
