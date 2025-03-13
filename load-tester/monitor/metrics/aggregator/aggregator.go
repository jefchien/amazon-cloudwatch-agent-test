// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package aggregator

import (
	"context"
	"errors"
	"load-tester/monitor/metrics"
	"load-tester/monitor/metrics/distribution"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type internalKey struct {
	name         string
	dimensionStr string
}

type Key struct {
	Name       string
	Dimensions map[string]string
}

func (k Key) validate() error {
	if k.Name == "" {
		return errors.New("metric name cannot be empty")
	}
	return nil
}

func (k Key) toInternalKey() internalKey {
	return internalKey{
		name:         k.Name,
		dimensionStr: toDimensionStr(k.Dimensions),
	}
}

func toDimensionStr(dimensions map[string]string) string {
	pairs := make([]string, 0, len(dimensions))
	for k, v := range dimensions {
		pairs = append(pairs, k+"="+v)
	}
	sort.Strings(pairs)
	return strings.Join(pairs, ";")
}

func fromDimensionStr(dimensionStr string) []types.Dimension {
	if len(dimensionStr) == 0 {
		return nil
	}
	pairs := strings.Split(dimensionStr, ";")
	dimensions := make([]types.Dimension, 0, len(pairs))
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) != 2 {
			continue
		}
		dimensions = append(dimensions, types.Dimension{
			Name:  aws.String(kv[0]),
			Value: aws.String(kv[1]),
		})
	}
	return dimensions
}

type MetricAggregator struct {
	metrics    map[internalKey]distribution.Distribution
	dimensions []types.Dimension
	mu         sync.Mutex
}

func NewMetricAggregator(dimensions map[string]string) *MetricAggregator {
	return &MetricAggregator{
		metrics:    make(map[internalKey]distribution.Distribution),
		dimensions: toDimensions(dimensions),
	}
}

func (m *MetricAggregator) Start(ctx context.Context, metricsChan <-chan *metrics.ProcessMetrics) {
	for {
		select {
		case <-ctx.Done():
			return
		case metric := <-metricsChan:
			for name, entry := range metric.Entries() {
				if err := m.Add(Key{Name: name}, entry); err != nil {
					log.Printf("Failed to add metric %s: %v", name, err)
				}
			}
		}
	}
}

func (m *MetricAggregator) Add(key Key, entry distribution.Entry) error {
	if err := key.validate(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	ik := key.toInternalKey()
	if _, ok := m.metrics[ik]; !ok {
		m.metrics[ik] = distribution.NewRegularDistribution()
	}
	return m.metrics[ik].AddEntry(entry)
}

func (m *MetricAggregator) Flush() []types.MetricDatum {
	m.mu.Lock()
	defer m.mu.Unlock()

	datums := make([]types.MetricDatum, 0, len(m.metrics))
	for ik, dist := range m.metrics {
		values, counts := dist.ValuesAndCounts()
		s := types.StatisticSet{
			Maximum:     aws.Float64(dist.Maximum()),
			Minimum:     aws.Float64(dist.Minimum()),
			SampleCount: aws.Float64(dist.SampleCount()),
			Sum:         aws.Float64(dist.Sum()),
		}
		ds := append(m.dimensions, fromDimensionStr(ik.dimensionStr)...)
		datums = append(datums, types.MetricDatum{
			MetricName:      aws.String(ik.name),
			Timestamp:       aws.Time(time.Now()),
			Unit:            dist.Unit(),
			Values:          values,
			Counts:          counts,
			StatisticValues: &s,
			Dimensions:      ds,
		})
	}

	m.metrics = make(map[internalKey]distribution.Distribution)
	return datums
}

func toDimensions(dimensions map[string]string) []types.Dimension {
	ds := make([]types.Dimension, 0, len(dimensions))
	for k, v := range dimensions {
		ds = append(ds, types.Dimension{
			Name:  aws.String(k),
			Value: aws.String(v),
		})
	}
	return ds
}
