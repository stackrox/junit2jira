# junit2jira

Utility tools for handling test failures

### Build
```shell
go build  -o . ./...
```

### Test
```shell
go test ./...
```

### Usage

This repo provides two cli tools:
- junit2jira
- flakechecker

### junit2jira

`junit2jira` supports conversion of test failures to jira issues. It also posts Slack messages for new failures and imports test results into DB.

*Usage*

```shell
Usage of junit2jira:
  -base-link string
    	Link to source code at the exact version under test.
  -build-id string
    	Build job run ID.
  -build-link string
    	Link to build job.
  -build-tag string
    	Built tag or revision.
  -csv-output string
    	Convert XML to a CSV file (use dash [-] for stdout)
  -debug
    	Enable debug log level
  -dry-run
    	When set to true issues will NOT be created.
  -html-output string
    	Generate HTML report to this file (use dash [-] for stdout)
  -jira-url string
    	Url of JIRA instance (default "https://issues.redhat.com/")
  -job-name string
    	Name of CI job.
  -junit-reports-dir string
    	Dir that contains jUnit reports XML files
  -orchestrator string
    	Orchestrator name (such as GKE or OpenShift), if any.
  -slack-output string
    	Generate JSON output in slack format (use dash [-] for stdout)
  -threshold int
    	Number of reported failures that should cause single issue creation. (default 10)
  -timestamp string
    	Timestamp of CI test. (default "2023-09-04T17:50:36+02:00")
  -v	short alias for -version
  -version
    	print version information and exit
```

*Example usage*
```shell
JIRA_TOKEN="..." junit2jira \
  -jira-url "https://..." \
  -junit-reports-dir "..." \
  -base-link "https://..." \
  -build-id "$BUILD_ID|GITHUB_RUN_ID" \
  -build-link "https://..." \
  -build-tag "$STACKROX_BUILD_TAG|$GITHUB_SHA" \
  -job-name "$JOB_NAME|$GITHUB_WORKFLOW" \
  -orchestrator "$ORCHESTRATOR_FLAVOR" \
  -timestamp $(date --rfc-3339=seconds)
  -csv-output -
```

### flakechecker

`flakechecker` helps prevent unnecessary CI pipeline failures by suppressing known flaky tests that are within the allowed failure thresholds.

`flakechecker` relies on several components:
- collected test results from `junit2jira`: we generate a table of flaky tests, including their failure ratios for the last 30 executions.
- flaky test configuration: we define and provide a `flakechecker` configuration with allowed failure ratio thresholds for known flaky tests.
- CI pipeline integration script: `flakechecker` is executed as the last step in a CI pipeline, and provided results allow the CI pipeline script to report success or failure.

 The `flakechecker` expects at least one failed test. It will return an error if it is executed on test results without any failures.

`flakechecker` decision making:
- it checks if a failed test in a CI pipeline is listed as flaky in the provided configuration.
- if the test is not found in the flaky tests config -> it will cause the CI pipeline to fail. (test not found)
- if the test is found in the configuration, `flakechecker` will fetch information about the fail ratio for that test from the database. If we have fewer than 30 executions for that test -> it will cause the CI pipeline to fail. (insufficient historical test results)
- if the test's failure ratio in the database exceeds the threshold defined in the config -> it will cause the CI pipeline to fail. (flake ratio is above the allowed threshold)
- if a flaky test's failure ratio is below the defined threshold -> it will report the test as a success in the CI pipeline. (test suppression)

The `flakechecker` will apply this logic for each failed test in the CI pipeline.

*Usage*

```
Usage of flakechecker:
  -config-file string
        Config file with allowed flakes.
  -debug
        Enable debug log level.
  -job-name string
        Name of CI job.
  -junit-reports-dir string
        Directory containing JUnit report XML files.
  -v    short alias for -version
  -version
        print version information and exit
```

*Example usage*
```
flakechecker --config-file flake-config.yml --job-name "${JOB_NAME}" -junit-reports-dir "${ARTIFACT_DIR}"
```
