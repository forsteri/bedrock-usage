package query

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type UsageRecord struct {
	Key          string
	InputTokens  int64
	OutputTokens int64
	Invocations  int64
}

type Client struct {
	cwl      *cloudwatchlogs.Client
	logGroup string
}

func NewClient(cwl *cloudwatchlogs.Client, logGroup string) *Client {
	return &Client{cwl: cwl, logGroup: logGroup}
}

func (c *Client) Query(ctx context.Context, startTime, endTime time.Time, params QueryParams) ([]UsageRecord, error) {
	queryStr := BuildQuery(params)

	startOutput, err := c.cwl.StartQuery(ctx, &cloudwatchlogs.StartQueryInput{
		LogGroupName: aws.String(c.logGroup),
		StartTime:    aws.Int64(startTime.Unix()),
		EndTime:      aws.Int64(endTime.Unix()),
		QueryString:  aws.String(queryStr),
	})
	if err != nil {
		return nil, fmt.Errorf("StartQuery に失敗しました: %w", err)
	}

	return c.waitForResults(ctx, *startOutput.QueryId)
}

func (c *Client) waitForResults(ctx context.Context, queryID string) ([]UsageRecord, error) {
	for {
		result, err := c.cwl.GetQueryResults(ctx, &cloudwatchlogs.GetQueryResultsInput{
			QueryId: aws.String(queryID),
		})
		if err != nil {
			return nil, fmt.Errorf("GetQueryResults に失敗しました: %w", err)
		}

		switch result.Status {
		case types.QueryStatusComplete:
			return parseResults(result.Results), nil
		case types.QueryStatusFailed, types.QueryStatusCancelled, types.QueryStatusTimeout:
			return nil, fmt.Errorf("クエリが失敗しました (status: %s)", result.Status)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func parseResults(rows [][]types.ResultField) []UsageRecord {
	var records []UsageRecord
	for _, row := range rows {
		rec := UsageRecord{}
		for _, field := range row {
			name := aws.ToString(field.Field)
			value := aws.ToString(field.Value)
			switch name {
			case "modelId", "identity.arn", "date":
				rec.Key = value
			case "totalInput":
				rec.InputTokens = parseFloat64ToInt64(value)
			case "totalOutput":
				rec.OutputTokens = parseFloat64ToInt64(value)
			case "invocations":
				rec.Invocations = parseFloat64ToInt64(value)
			}
		}
		if rec.Key != "" {
			records = append(records, rec)
		}
	}
	return records
}

func parseFloat64ToInt64(s string) int64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return int64(f)
}
