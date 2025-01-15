package main

import (
	"cloud.google.com/go/bigquery"
	"context"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"time"
)

const projectID = "acs-san-stackroxci"
const queryTimeout = 1 * time.Minute
const queryStrGetFailureRatio = `
SELECT
    TotalAll,
    FailRatio
FROM
` + "`acs-san-stackroxci.ci_metrics.stackrox_tests__recent_flaky_tests`" + `
WHERE
    JobName = @jobName
    AND Classname = @className
    AND Name = @testName
`

type recentFlakyTestInfo struct {
	TotalAll  int
	FailRatio int
}

type biqQueryClient interface {
	GetRatioForTest(config flakeDetectionPolicyConfig, testName string) (int, int, error)
}

type bigQueryClient struct {
	client *bigquery.Client
}

func getNewBigQueryClient() (biqQueryClient, error) {
	ctx := context.Background()

	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, errors.Wrap(err, "creating BigQuery client")
	}

	return &bigQueryClient{client: client}, nil
}

func (c *bigQueryClient) GetRatioForTest(config flakeDetectionPolicyConfig, testName string) (int, int, error) {
	query := c.client.Query(queryStrGetFailureRatio)
	query.Parameters = []bigquery.QueryParameter{
		{Name: "jobName", Value: config.RatioJobName},
		{Name: "className", Value: config.ClassName},
		{Name: "testName", Value: testName},
	}

	ctx, cancelBigQueryRequest := context.WithTimeout(context.Background(), queryTimeout)
	defer cancelBigQueryRequest()

	resIter, err := query.Read(ctx)
	if err != nil {
		return 0, 0, errors.Wrap(err, "query data from BigQuery")
	}

	// We need only first flakyTestInfo. No need to loop over iterator.
	var flakyTestInfo recentFlakyTestInfo
	if errNext := resIter.Next(&flakyTestInfo); errNext != nil {
		return 0, 0, errors.Wrapf(errNext, "read BigQuery result for flaky test for query params: %v - query: %s", query.Parameters, queryStrGetFailureRatio)
	}

	if resIter.TotalRows > 1 {
		log.Warnf("Expected to find one row in DB, but got more for query params: %v - query: %s", query.Parameters, queryStrGetFailureRatio)
	}

	return flakyTestInfo.TotalAll, flakyTestInfo.FailRatio, nil
}
