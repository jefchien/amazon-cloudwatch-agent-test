// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package cloudwatch

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type Client struct {
	cwClient  *cloudwatch.Client
	namespace string
}

func NewClient(ctx context.Context, namespace, region string) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %v", err)
	}

	return &Client{
		cwClient:  cloudwatch.NewFromConfig(cfg),
		namespace: namespace,
	}, nil
}

func (c *Client) PutMetricData(ctx context.Context, datums []types.MetricDatum) error {
	_, err := c.cwClient.PutMetricData(ctx, &cloudwatch.PutMetricDataInput{
		Namespace:  &c.namespace,
		MetricData: datums,
	})
	if err != nil {
		return fmt.Errorf("error putting metric data: %v", err)
	}

	return nil
}

func Prettify(datums []types.MetricDatum) string {
	lines := make([]string, 0, len(datums))
	for _, datum := range datums {
		average := *datum.StatisticValues.Sum / *datum.StatisticValues.SampleCount
		lines = append(lines, fmt.Sprintf("%s=%v", *datum.MetricName, average))
	}
	sort.Strings(lines)
	return strings.Join(lines, ", ")
}
