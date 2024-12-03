package main

import (
	"cloud.google.com/go/bigquery"
	"context"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
	"time"
)

const projectID = "acs-san-stackroxci"
const queryTimeout = 1 * time.Minute
const queryStrGetFailureRatio = `
SELECT
    JobName,
    Name,
    Classname,
    TotalAll,
    FailRatio
FROM
` + "`acs-san-stackroxci.ci_metrics.stackrox_tests__recent_flaky_tests`" + `
WHERE
    JobName = @jobName
    AND Name = @name
    AND Classname = @classname
`

type biqQueryClient interface {
	GetRatioForTest(flakeTestConfig *flakeDetectionPolicy, testName string) (int, int, error)
}

type biqQueryClientImpl struct {
	client *bigquery.Client
}

func getNewBigQueryClient() (biqQueryClient, error) {
	ctx := context.Background()

	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, errors.Wrap(err, "creating BigQuery client")
	}

	return &biqQueryClientImpl{client: client}, nil
}

func (c *biqQueryClientImpl) GetRatioForTest(flakeTestRec *flakeDetectionPolicy, testName string) (int, int, error) {
	query := c.client.Query(queryStrGetFailureRatio)
	query.Parameters = []bigquery.QueryParameter{
		{Name: "jobName", Value: flakeTestRec.config.RatioJobName},
		{Name: "name", Value: testName},
		{Name: "classname", Value: flakeTestRec.config.Classname},
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

	if errNext := resIter.Next(&flakyTestInfo); !errors.Is(errNext, iterator.Done) {
		log.Warnf("Expected to find one row in DB, but got more for query params: %v - query: %s", query.Parameters, queryStrGetFailureRatio)
	}

	return flakyTestInfo.TotalAll, flakyTestInfo.FailRatio, nil
}
