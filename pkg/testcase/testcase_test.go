package testcase

import (
	"github.com/joshdk/go-junit"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_getClearedSuites(t *testing.T) {
	tests := map[string]struct {
		ignoreList     []ignoreTestCase
		suites         []junit.Suite
		expectedSuites []junit.Suite
	}{
		"empty suites": {},
		"simple no match": {
			ignoreList: []ignoreTestCase{
				{Name: "match", Classname: "me"},
			},
			suites: []junit.Suite{
				{
					Suites: []junit.Suite{},
					Tests: []junit.Test{
						{
							Name:      "do not match",
							Classname: "me",
						},
					},
				},
			},
			expectedSuites: []junit.Suite{
				{
					Suites: []junit.Suite{},
					Tests: []junit.Test{
						{
							Name:      "do not match",
							Classname: "me",
						},
					},
				},
			},
		},
		"simple match": {
			ignoreList: []ignoreTestCase{
				{Name: "match", Classname: "me"},
			},
			suites: []junit.Suite{
				{
					Suites: []junit.Suite{},
					Tests: []junit.Test{
						{
							Name:      "match",
							Classname: "me",
						},
					},
				},
			},
			expectedSuites: []junit.Suite{},
		},
		"nested suites only": {
			ignoreList: []ignoreTestCase{
				{Name: "match", Classname: "me"},
			},
			suites: []junit.Suite{
				{
					Suites: []junit.Suite{
						{
							Suites: []junit.Suite{},
							Tests: []junit.Test{
								{
									Name:      "skip",
									Classname: "me",
								},
								{
									Name:      "match",
									Classname: "me",
								},
								{
									Name:      "match",
									Classname: "other",
								},
							},
						},
					},
					Tests: []junit.Test{},
				},
			},
			expectedSuites: []junit.Suite{
				{
					Suites: []junit.Suite{
						{
							Suites: []junit.Suite{},
							Tests: []junit.Test{
								{
									Name:      "skip",
									Classname: "me",
								},
								{
									Name:      "match",
									Classname: "other",
								},
							},
						},
					},
					Tests: []junit.Test{},
				},
			},
		},
		"nested suites and tests": {
			ignoreList: []ignoreTestCase{
				{Name: "match", Classname: "me"},
			},
			suites: []junit.Suite{
				{
					Suites: []junit.Suite{
						{
							Suites: []junit.Suite{},
							Tests: []junit.Test{
								{
									Name:      "skip",
									Classname: "me",
								},
								{
									Name:      "match",
									Classname: "me",
								},
								{
									Name:      "match",
									Classname: "other",
								},
							},
						},
					},
					Tests: []junit.Test{
						{
							Name:      "skip",
							Classname: "me",
						},
						{
							Name:      "match",
							Classname: "me",
						},
						{
							Name:      "match",
							Classname: "other",
						},
					},
				},
			},
			expectedSuites: []junit.Suite{
				{
					Suites: []junit.Suite{
						{
							Suites: []junit.Suite{},
							Tests: []junit.Test{
								{
									Name:      "skip",
									Classname: "me",
								},
								{
									Name:      "match",
									Classname: "other",
								},
							},
						},
					},
					Tests: []junit.Test{
						{
							Name:      "skip",
							Classname: "me",
						},
						{
							Name:      "match",
							Classname: "other",
						},
					},
				},
			},
		},
		"match all nested suites only": {
			ignoreList: []ignoreTestCase{
				{Name: "match", Classname: "me"},
				{Name: "me", Classname: "including"},
			},
			suites: []junit.Suite{
				{
					Suites: []junit.Suite{
						{
							Suites: []junit.Suite{},
							Tests: []junit.Test{
								{
									Name:      "match",
									Classname: "me",
								},
								{
									Name:      "me",
									Classname: "including",
								},
							},
						},
					},
					Tests: []junit.Test{},
				},
			},
			expectedSuites: []junit.Suite{},
		},
		"match all test only": {
			ignoreList: []ignoreTestCase{
				{Name: "match", Classname: "me"},
			},
			suites: []junit.Suite{
				{
					Suites: []junit.Suite{
						{
							Suites: []junit.Suite{},
							Tests: []junit.Test{
								{
									Name:      "skip",
									Classname: "me",
								},
							},
						},
					},
					Tests: []junit.Test{
						{
							Name:      "match",
							Classname: "me",
						},
					},
				},
			},
			expectedSuites: []junit.Suite{
				{
					Suites: []junit.Suite{
						{
							Suites: []junit.Suite{},
							Tests: []junit.Test{
								{
									Name:      "skip",
									Classname: "me",
								},
							},
						},
					},
					Tests: []junit.Test{},
				},
			},
		},
		"match all suites and tests": {
			ignoreList: []ignoreTestCase{
				{Name: "remove", Classname: "it"},
				{Name: "why", Classname: "me"},
			},
			suites: []junit.Suite{
				{
					Suites: []junit.Suite{
						{
							Suites: []junit.Suite{},
							Tests: []junit.Test{
								{
									Name:      "remove",
									Classname: "it",
								},
							},
						},
						{
							Suites: []junit.Suite{},
							Tests: []junit.Test{
								{
									Name:      "why",
									Classname: "me",
								},
							},
						},
					},
					Tests: []junit.Test{
						{
							Name:      "remove",
							Classname: "it",
						},
						{
							Name:      "why",
							Classname: "me",
						},
					},
				},
			},
			expectedSuites: []junit.Suite{},
		},
		"match empty name and classname": {
			ignoreList: []ignoreTestCase{
				{Name: "only-name", Classname: ""},
				{Name: "", Classname: "only-class"},
			},
			suites: []junit.Suite{
				{
					Suites: []junit.Suite{},
					Tests: []junit.Test{
						{
							Name:      "only-name",
							Classname: "",
						},
						{
							Name:      "only-name",
							Classname: "only-class",
						},
						{
							Name:      "",
							Classname: "only-class",
						},
					},
				},
			},
			expectedSuites: []junit.Suite{
				{
					Suites: []junit.Suite{},
					Tests: []junit.Test{
						{
							Name:      "only-name",
							Classname: "only-class",
						},
					},
				},
			},
		},
	}

	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			ignoreList = tt.ignoreList
			actual := getClearedSuites(tt.suites)
			assert.ElementsMatch(t, tt.expectedSuites, actual)
		})
	}
}

func Test_addFallbacks(t *testing.T) {
	tests := map[string]struct {
		tests        []junit.Test
		expectedTest []junit.Test
	}{
		"empty": {
			tests:        []junit.Test{},
			expectedTest: []junit.Test{},
		},
		"mysterious test case": {
			tests: []junit.Test{
				{
					Name:      "",
					Classname: "",
				},
			},
			expectedTest: []junit.Test{
				{
					Name:      fallbackName,
					Classname: fallbackClassname,
				},
			},
		},
		"add name": {
			tests: []junit.Test{
				{
					Name:      "",
					Classname: "no name",
				},
			},
			expectedTest: []junit.Test{
				{
					Name:      fallbackName,
					Classname: "no name",
				},
			},
		},
		"add classname": {
			tests: []junit.Test{
				{
					Name:      "no class",
					Classname: "",
				},
			},
			expectedTest: []junit.Test{
				{
					Name:      "no class",
					Classname: fallbackClassname,
				},
			},
		},
		"all good": {
			tests: []junit.Test{
				{
					Name:      "with name",
					Classname: "and class",
				},
			},
			expectedTest: []junit.Test{
				{
					Name:      "with name",
					Classname: "and class",
				},
			},
		},
	}

	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			actual := addFallbacks(tt.tests)
			assert.ElementsMatch(t, tt.expectedTest, actual)
		})
	}
}
