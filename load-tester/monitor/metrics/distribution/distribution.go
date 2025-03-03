// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package distribution

import (
	"errors"
	"math"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

var (
	ErrUnsupportedWeight = errors.New("weight must be larger than 0")
	ErrUnsupportedValue  = errors.New("value cannot be negative, NaN, Inf, or greater than 2^360")
	MinValue             = -math.Pow(2, 360)
	MaxValue             = math.Pow(2, 360)
)

type Entry struct {
	Value  float64
	Weight float64
	Unit   types.StandardUnit
}

func NewEntry(value float64, unit types.StandardUnit) Entry {
	return Entry{Value: value, Weight: 1, Unit: unit}
}

type Distribution interface {
	Maximum() float64

	Minimum() float64

	SampleCount() float64

	Sum() float64

	ValuesAndCounts() ([]float64, []float64)

	Unit() types.StandardUnit

	Size() int

	AddEntry(entry Entry) error

	AddDistribution(distribution Distribution)

	AddDistributionWithWeight(distribution Distribution, weight float64)
}

// IsSupportedValue checks to see if the metric is between the min value and 2^360 and not a NaN.
// This matches the accepted range described in the MetricDatum documentation
// https://docs.aws.amazon.com/AmazonCloudWatch/latest/APIReference/API_MetricDatum.html
func IsSupportedValue(value, min, max float64) bool {
	return !math.IsNaN(value) && value >= min && value <= max
}
