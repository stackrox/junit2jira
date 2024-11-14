package testcase

import (
	"fmt"
	"github.com/joshdk/go-junit"
	"strings"
)

const subTestFormat = "\nSub test %s: %s"

type TestCase struct {
	Name      string
	Classname string
	Suite     string
	Message   string
	Stdout    string
	Stderr    string
	Error     string
	IsSubtest bool
}

func (tc *TestCase) addSubTest(subTest junit.Test) {
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

func NewTestCase(tc junit.Test) TestCase {
	c := TestCase{
		Name:      tc.Name,
		Classname: tc.Classname,
		Message:   tc.Message,
		Stdout:    tc.SystemOut,
		Stderr:    tc.SystemErr,
		Suite:     tc.Classname,
	}

	if tc.Error != nil {
		c.Error = tc.Error.Error()
	}

	return c
}

func isSubTest(tc junit.Test) bool {
	return strings.Contains(tc.Name, "/")
}

// isGoTest will verify that the corresponding classname refers to a go package by expecting the go module name as prefix.
func isGoTest(className string) bool {
	return strings.HasPrefix(className, "github.com/stackrox/rox")
}

func addSubTestToFailedTest(subTest junit.Test, failedTests []TestCase) []TestCase {
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
	return append(failedTests, NewTestCase(subTest))
}

func addTest(failedTests []TestCase, tc junit.Test) []TestCase {
	if !isSubTest(tc) {
		return append(failedTests, NewTestCase(tc))
	}
	return addSubTestToFailedTest(tc, failedTests)
}

func addFailedTests(ts junit.Suite, failedTests []TestCase) []TestCase {
	for _, suite := range ts.Suites {
		failedTests = addFailedTests(suite, failedTests)
	}
	for _, tc := range ts.Tests {
		if tc.Error == nil {
			continue
		}
		failedTests = addTest(failedTests, tc)
	}
	return failedTests
}

func GetFailedTests(testSuites []junit.Suite) ([]TestCase, error) {
	failedTests := make([]TestCase, 0)
	for _, ts := range testSuites {
		failedTests = addFailedTests(ts, failedTests)
	}

	return failedTests, nil
}
