// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package aggregator

import (
	"context"
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

type Key struct {
	Name       string
	Dimensions map[string]string
}

func (key Key) String() string {
	if len(key.Dimensions) == 0 {
		return key.Name
	}
	pairs := make([]string, 0, len(key.Dimensions))
	for k, v := range key.Dimensions {
		pairs = append(pairs, k+"="+v)
	}
	sort.Strings(pairs)
	return key.Name + "|" + strings.Join(pairs, ";")
}

func toKey(str string) Key {
	parts := strings.Split(str, "|")
	if len(parts) == 1 {
		return Key{Name: str}
	}
	pairs := strings.Split(parts[1], ";")
	dimensions := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) == 2 {
			dimensions[kv[0]] = kv[1]
		}
	}
	return Key{Name: parts[0], Dimensions: dimensions}
}

type MetricAggregator struct {
	metrics    map[string]distribution.Distribution
	dimensions []types.Dimension
	mu         sync.Mutex
}

func NewMetricAggregator(dimensions map[string]string) *MetricAggregator {
	return &MetricAggregator{
		metrics:    make(map[string]distribution.Distribution),
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
	m.mu.Lock()
	defer m.mu.Unlock()

	keyStr := key.String()
	if _, ok := m.metrics[keyStr]; !ok {
		m.metrics[keyStr] = distribution.NewRegularDistribution()
	}
	return m.metrics[keyStr].AddEntry(entry)
}

func (m *MetricAggregator) Flush() []types.MetricDatum {
	m.mu.Lock()
	defer m.mu.Unlock()

	datums := make([]types.MetricDatum, 0, len(m.metrics))
	for keyStr, dist := range m.metrics {
		values, counts := dist.ValuesAndCounts()
		s := types.StatisticSet{
			Maximum:     aws.Float64(dist.Maximum()),
			Minimum:     aws.Float64(dist.Minimum()),
			SampleCount: aws.Float64(dist.SampleCount()),
			Sum:         aws.Float64(dist.Sum()),
		}
		key := toKey(keyStr)
		ds := append(m.dimensions, toDimensions(key.Dimensions)...)
		datums = append(datums, types.MetricDatum{
			MetricName:      aws.String(key.Name),
			Timestamp:       aws.Time(time.Now()),
			Unit:            dist.Unit(),
			Values:          values,
			Counts:          counts,
			StatisticValues: &s,
			Dimensions:      ds,
		})
	}

	m.metrics = make(map[string]distribution.Distribution)
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
