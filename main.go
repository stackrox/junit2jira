package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"github.com/andygrunwald/go-jira"
	"github.com/hashicorp/go-multierror"
	"github.com/joshdk/go-junit"
	"github.com/pkg/errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"
	"unicode"
)

const jql = `project in (ROX)
AND issuetype = Bug
AND status != Closed
AND labels = CI_Failure
AND summary ~ %q
ORDER BY created DESC`

func main() {
	p := params{}
	flag.StringVar(&p.csvOutput, "csv-output", "", "Convert XML to a CSV file (use dash [-] for stdout)")
	flag.StringVar(&p.jiraUrl, "jira-url", "https://issues.redhat.com/", "Url of JIRA instance")
	flag.StringVar(&p.junitReportsDir, "junit-reports-dir", os.Getenv("ARTIFACT_DIR"), "Dir that contains jUnit reports XML files")
	flag.BoolVar(&p.dryRun, "dry-run", false, "When set to true issues will NOT be created.")
	flag.IntVar(&p.threshold, "threshold", 10, "Number of reported failures that should cause single issue creation.")
	flag.StringVar(&p.timestamp, "timestamp", time.Now().Format(time.RFC3339), "Timestamp of CI test.")
	flag.StringVar(&p.BaseLink, "base-link", "", "Link to source code at the exact version under test.")
	flag.StringVar(&p.BuildId, "build-id", "", "Build job run ID.")
	flag.StringVar(&p.BuildLink, "build-link", "", "Link to build job.")
	flag.StringVar(&p.BuildTag, "build-tag", "", "Built tag or revision.")
	flag.StringVar(&p.JobName, "job-name", "", "Name of CI job.")
	flag.StringVar(&p.Orchestrator, "orchestrator", "", "Orchestrator name (such as GKE or OpenShift), if any.")

	flag.Parse()

	err := run(p)
	if err != nil {
		log.Fatal(err)
	}
}

type junit2jira struct {
	params
	jiraClient *jira.Client
}

func run(p params) error {
	transport := http.DefaultTransport

	tp := jira.PATAuthTransport{
		Token:     os.Getenv("JIRA_TOKEN"),
		Transport: transport,
	}

	jiraClient, err := jira.NewClient(tp.Client(), p.jiraUrl)
	if err != nil {
		return errors.Wrapf(err, "could not create client for %s", p.jiraUrl)
	}

	j := &junit2jira{
		params:     p,
		jiraClient: jiraClient,
	}

	testSuites, err := junit.IngestDir(p.junitReportsDir)
	if err != nil {
		log.Fatalf("coud not read files: %s", err)
	}

	err = j.createCsv(testSuites)
	if err != nil {
		log.Fatalf("coud create CSV: %s", err)
	}

	failedTests, err := j.findFailedTests(testSuites)
	if err != nil {
		return errors.Wrap(err, "could not find failed tests")
	}

	err = j.createIssuesOrComments(failedTests)
	if err != nil {
		return errors.Wrap(err, "could not create issues or comments")
	}
	return nil
}

func (j junit2jira) createCsv(testSuites []junit.Suite) error {
	if j.csvOutput == "" {
		return nil
	}
	out := os.Stdout
	if j.csvOutput != "-" {
		file, err := os.Create(j.csvOutput)
		if err != nil {
			return fmt.Errorf("could not create file %s: %w", j.csvOutput, err)
		}
		out = file
		defer file.Close()
	}
	return junit2csv(testSuites, j.params, out)
}

func (j junit2jira) createIssuesOrComments(failedTests []testCase) error {
	var result error
	for _, tc := range failedTests {
		err := j.createIssueOrComment(tc)
		if err != nil {
			result = multierror.Append(result, err)
		}
	}
	return result
}

func (j junit2jira) createIssueOrComment(tc testCase) error {
	summary, err := tc.summary()
	if err != nil {
		return fmt.Errorf("could not get summary: %w", err)
	}
	description, err := tc.description()
	if err != nil {
		return fmt.Errorf("could not get description: %w", err)
	}
	log.Println("Searching for ", summary)
	search, response, err := j.jiraClient.Issue.Search(fmt.Sprintf(jql, summary), nil)
	if err != nil {
		logError(err, response)
		return fmt.Errorf("could not search: %w", err)
	}

	issue := findMatchingIssue(search, summary)

	if issue == nil {
		log.Println("Issue not found. Creating new issue...")
		log.Println(summary)
		log.Println(description)
		if j.dryRun {
			log.Println("Dry run: will just print issue content")
			log.Println(summary)
			log.Println(description)
			return nil
		}
		create, response, err := j.jiraClient.Issue.Create(newIssue(summary, description))
		if response != nil && err != nil {
			logError(err, response)
			return fmt.Errorf("could not create issue %s: %w", summary, err)
		}
		log.Printf("Created new issues: %s:%s", create.Key, summary)
		return nil
	}

	comment := jira.Comment{
		Body: description,
	}

	log.Printf("Found issue: %s %s. Creating a coment...", issue.ID, issue.Fields.Summary)

	if j.dryRun {
		log.Println("Dry run: will just print comment")
		log.Println(description)
		return nil
	}

	addComment, response, err := j.jiraClient.Issue.AddComment(issue.ID, &comment)
	if response != nil && err != nil {
		logError(err, response)
		return fmt.Errorf("could not create issue %s: %w", summary, err)
	}
	log.Printf("Created comment %s for %s:%s ", addComment.ID, issue.Key, summary)
	return nil
}

func newIssue(summary string, description string) *jira.Issue {
	return &jira.Issue{
		Fields: &jira.IssueFields{
			Type: jira.IssueType{
				Name: "Bug",
			},
			Project: jira.Project{
				Key: "ROX",
			},
			Summary:     summary,
			Description: description,
			Labels:      []string{"CI_Failure"},
		},
	}
}

func findMatchingIssue(search []jira.Issue, summary string) *jira.Issue {
	for _, i := range search {
		if i.Fields.Summary == summary {
			return &i
		}
	}
	return nil
}

func logError(err error, response *jira.Response) {
	log.Println(err)
	log.Println(response.StatusCode)
	all, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println("Could not read body", err)
	} else {
		log.Println(string(all))
	}
}

func junit2csv(testSuites []junit.Suite, p params, output io.Writer) error {
	w := csv.NewWriter(output)
	header := []string{
		"BuildId",
		"Timestamp",
		"Classname",
		"Name",
		"Duration",
		"Status",
		"JobName",
		"BuildTag",
	}
	err := w.Write(header)
	if err != nil {
		return fmt.Errorf("coud not write header: %w", err)
	}
	for _, ts := range testSuites {
		for _, tc := range ts.Tests {
			duration := fmt.Sprintf("%d", tc.Duration.Milliseconds())
			row := []string{
				p.BuildId,         // BuildId
				p.timestamp,       // Timestamp
				tc.Classname,      // Classname
				tc.Name,           // Name
				duration,          // Duration
				string(tc.Status), // Status
				p.JobName,         // JobName
				p.BuildTag,        // BuildTag
			}
			err := w.Write(row)
			if err != nil {
				return fmt.Errorf("coud not write row: %w", err)
			}
		}
	}
	w.Flush()
	if w.Error() != nil {
		return fmt.Errorf("could not flush CSV: %w", w.Error())
	}
	return nil
}

func (j junit2jira) findFailedTests(testSuites []junit.Suite) ([]testCase, error) {
	failedTests := make([]testCase, 0)
	for _, ts := range testSuites {
		for _, tc := range ts.Tests {
			if tc.Error == nil {
				continue
			}
			failedTests = j.addFailedTest(failedTests, tc)
		}
	}
	log.Printf("Found %d failed tests", len(failedTests))

	if len(failedTests) > j.threshold && j.threshold > 0 {
		return j.mergeFailedTests(failedTests)
	}

	return failedTests, nil
}

func (j junit2jira) mergeFailedTests(failedTests []testCase) ([]testCase, error) {
	log.Println("Too many failed tests, reporting them as a one failure.")
	msg := ""
	suite := failedTests[0].Suite
	for _, t := range failedTests {
		summary, err := t.summary()
		if err != nil {
			return nil, errors.Wrapf(err, "could not get summary of %+v", t)
		}
		// If there are multiple suites, do not report them.
		if suite != t.Suite {
			suite = j.JobName
		}
		msg += summary + "\n"
	}
	tc := NewTestCase(junit.Test{
		Message:   msg,
		Classname: suite,
	}, j.params)
	return []testCase{tc}, nil
}

func (j junit2jira) addFailedTest(failedTests []testCase, tc junit.Test) []testCase {
	if !isSubTest(tc) {
		return append(failedTests, NewTestCase(tc, j.params))
	}
	return j.addSubTestToFailedTest(tc, failedTests)
}

func isSubTest(tc junit.Test) bool {
	return strings.Contains(tc.Name, "/")
}

func (j junit2jira) addSubTestToFailedTest(subTest junit.Test, failedTests []testCase) []testCase {
	// As long as the separator is not empty, split will always return a slice of length 1.
	name := strings.Split(subTest.Name, "/")[0]
	for i, failedTest := range failedTests {
		// Only consider a failed test a "parent" of the test if the name matches _and_ the class name is the same.
		if isGoTest(subTest.Classname) && failedTest.Name == name && failedTest.Suite == subTest.Classname {
			failedTest.addSubTest(subTest)
			failedTests[i] = failedTest
			return failedTests
		}
	}
	// In case we found no matches, we will default to add the subtest plain.
	return append(failedTests, NewTestCase(subTest, j.params))
}

// isGoTest will verify that the corresponding classname refers to a go package by expecting the go module name as prefix.
func isGoTest(className string) bool {
	return strings.HasPrefix(className, "github.com/stackrox/rox")
}

const (
	desc = `
{{- if .Message }}
{code:title=Message|borderStyle=solid}
{{ .Message | truncate }}
{code}
{{- end }}
{{- if .Stderr }}
{code:title=STDERR|borderStyle=solid}
{{ .Stderr | truncate }}
{code}
{{- end }}
{{- if .Stdout }}
{code:title=STDOUT|borderStyle=solid}
{{ .Stdout | truncate }}
{code}
{{- end }}
{{- if .Error }}
{code:title=ERROR|borderStyle=solid}
{{ .Error | truncate }}
{code}
{{- end }}

||    ENV     ||      Value           ||
| BUILD ID     | [{{- .BuildId -}}|{{- .BuildLink -}}]|
| BUILD TAG    | [{{- .BuildTag -}}|{{- .BaseLink -}}]|
| JOB NAME     | {{- .JobName -}}      |
| ORCHESTRATOR | {{- .Orchestrator -}} |
`
	summaryTpl = `{{ (print .Suite " / " .Name) | truncateSummary }} FAILED`
)

type testCase struct {
	Name         string
	Suite        string
	Message      string
	Stdout       string
	Stderr       string
	Error        string
	BuildId      string
	JobName      string
	Orchestrator string
	BuildTag     string
	BaseLink     string
	BuildLink    string
}

type params struct {
	BuildId      string
	JobName      string
	Orchestrator string
	BuildTag     string
	BaseLink     string
	BuildLink    string

	threshold       int
	dryRun          bool
	jiraUrl         string
	junitReportsDir string
	timestamp       string
	csvOutput       string
}

func NewTestCase(tc junit.Test, p params) testCase {
	c := testCase{
		Name:         tc.Name,
		Message:      tc.Message,
		Stdout:       tc.SystemOut,
		Stderr:       tc.SystemErr,
		Suite:        tc.Classname,
		BuildId:      p.BuildId,
		JobName:      p.JobName,
		Orchestrator: p.Orchestrator,
		BuildTag:     p.BuildTag,
		BaseLink:     p.BaseLink,
		BuildLink:    p.BuildLink,
	}

	if tc.Error != nil {
		c.Error = tc.Error.Error()
	}
	return c
}

func (tc *testCase) description() (string, error) {
	return render(*tc, desc)
}

func (tc testCase) summary() (string, error) {
	s, err := render(tc, summaryTpl)
	if err != nil {
		return "", err
	}
	return clearString(s), nil
}

const subTestFormat = "\nSub test %s: %s"

func (tc *testCase) addSubTest(subTest junit.Test) {
	if subTest.Message != "" {
		tc.Message += fmt.Sprintf(subTestFormat, subTest.Name, subTest.Message)
	}
	if subTest.SystemOut != "" {
		tc.Stdout += fmt.Sprintf(subTestFormat, subTest.Name, subTest.SystemOut)
	}
	if subTest.SystemErr != "" {
		tc.Stderr += fmt.Sprintf(subTestFormat, subTest.Name, subTest.SystemErr)
	}
	if subTest.Error != nil {
		tc.Error += fmt.Sprintf(subTestFormat, subTest.Name, subTest.Error.Error())
	}
}

func render(tc testCase, text string) (string, error) {
	tmpl, err := template.New("test").Funcs(map[string]any{"truncate": truncate, "truncateSummary": truncateSummary}).Parse(text)
	if err != nil {
		return "", err
	}
	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, tc)
	if err != nil {
		return "", err
	}
	return tpl.String(), nil
}

func clearString(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '/' || r == '-' || r == '_' {
			return r
		}
		return ' '
	}, str)
}

var maxTextBlockLength = 10000

func truncate(s string) string {
	runes := []rune(s)
	if len(runes) > maxTextBlockLength {
		return string(runes[:maxTextBlockLength]) + "\n … too long, truncated."
	}
	return s
}

var maxSummaryLength = 200

func truncateSummary(s string) string {
	runes := []rune(s)
	if len(runes) > maxSummaryLength {
		return string(runes[:maxSummaryLength]) + "..."
	}
	return s
}
