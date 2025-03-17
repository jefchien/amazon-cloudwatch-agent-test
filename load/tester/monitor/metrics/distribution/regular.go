// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package distribution

import (
	"fmt"
	"log"
	"math"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type RegularDistribution struct {
	maximum     float64
	minimum     float64
	sampleCount float64
	sum         float64
	buckets     map[float64]float64 // from  value to the counter (i.e. weight)
	unit        types.StandardUnit
}

func NewRegularDistribution() Distribution {
	return &RegularDistribution{
		maximum:     0, // negative number is not supported for now, so zero is the min value
		minimum:     math.MaxFloat64,
		sampleCount: 0,
		sum:         0,
		buckets:     map[float64]float64{},
	}
}

func (regularDist *RegularDistribution) Maximum() float64 {
	return regularDist.maximum
}

func (regularDist *RegularDistribution) Minimum() float64 {
	return regularDist.minimum
}

func (regularDist *RegularDistribution) SampleCount() float64 {
	return regularDist.sampleCount
}

func (regularDist *RegularDistribution) Sum() float64 {
	return regularDist.sum
}

func (regularDist *RegularDistribution) ValuesAndCounts() (values []float64, counts []float64) {
	values = []float64{}
	counts = []float64{}
	for value, counter := range regularDist.buckets {
		values = append(values, value)
		counts = append(counts, counter)
	}
	return
}

func (regularDist *RegularDistribution) Unit() types.StandardUnit {
	return regularDist.unit
}

func (regularDist *RegularDistribution) Size() int {
	return len(regularDist.buckets)
}

// weight is 1/samplingRate
func (regularDist *RegularDistribution) AddEntry(entry Entry) error {
	if entry.Weight <= 0 {
		return fmt.Errorf("unsupported weight %v: %w", entry.Weight, ErrUnsupportedWeight)
	}
	if !IsSupportedValue(entry.Value, 0, MaxValue) {
		return fmt.Errorf("unsupported value %v: %w", entry.Value, ErrUnsupportedValue)
	}
	//sample count
	regularDist.sampleCount += entry.Weight
	//sum
	regularDist.sum += entry.Value * entry.Weight
	//min
	if entry.Value < regularDist.minimum {
		regularDist.minimum = entry.Value
	}
	//max
	if entry.Value > regularDist.maximum {
		regularDist.maximum = entry.Value
	}

	//values and counts
	regularDist.buckets[entry.Value] += entry.Weight

	//unit
	if regularDist.unit == "" {
		regularDist.unit = entry.Unit
	} else if regularDist.unit != entry.Unit && entry.Unit != "" {
		log.Printf("D! Multiple units are detected: %s, %s", regularDist.unit, entry.Unit)
	}
	return nil
}

func (regularDist *RegularDistribution) AddDistribution(distribution Distribution) {
	regularDist.AddDistributionWithWeight(distribution, 1)
}

func (regularDist *RegularDistribution) AddDistributionWithWeight(distribution Distribution, weight float64) {
	if distribution.SampleCount()*weight > 0 {

		//values and counts
		if fromDistribution, ok := distribution.(*RegularDistribution); ok {
			for bucketNumber, bucketCounts := range fromDistribution.buckets {
				regularDist.buckets[bucketNumber] += bucketCounts * weight
			}
		} else {
			log.Printf("E! The from distribution type is not compatible with the to distribution type: from distribution type %T, to distribution type %T", regularDist, distribution)
			return
		}

		//sample count
		regularDist.sampleCount += distribution.SampleCount() * weight
		//sum
		regularDist.sum += distribution.Sum() * weight
		//min
		if distribution.Minimum() < regularDist.minimum {
			regularDist.minimum = distribution.Minimum()
		}
		//max
		if distribution.Maximum() > regularDist.maximum {
			regularDist.maximum = distribution.Maximum()
		}

		//unit
		if regularDist.unit == "" {
			regularDist.unit = distribution.Unit()
		} else if regularDist.unit != distribution.Unit() && distribution.Unit() != "" {
			log.Printf("D! Multiple units are dected: %s, %s", regularDist.unit, distribution.Unit())
		}
	} else {
		log.Printf("D! SampleCount * Weight should be larger than 0: %v, %v", distribution.SampleCount(), weight)
	}
}

func (regularDist *RegularDistribution) GetCount(value float64) float64 {
	return regularDist.buckets[value]
}
