// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package monitor

import (
	"context"
	"fmt"
	"load-tester/monitor/cloudwatch"
	"load-tester/monitor/metrics"
	"load-tester/monitor/metrics/aggregator"
	"load-tester/monitor/metrics/distribution"
	"log"
	"sync"
	"time"
)

type MetricsReporter interface {
	Add(key aggregator.Key, entry distribution.Entry) error
}

type Monitor struct {
	cfg         *Config
	collector   *metrics.Collector
	aggregator  *aggregator.MetricAggregator
	client      *cloudwatch.Client
	metricsChan chan *metrics.ProcessMetrics
}

func New(ctx context.Context, cfg *Config) (*Monitor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	collector, err := metrics.NewCollector(cfg.ProcessName)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize metrics collector: %w", err)
	}
	client, err := cloudwatch.NewClient(ctx, cfg.MetricsNamespace, cfg.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize CloudWatch client: %w", err)
	}
	return &Monitor{
		cfg:         cfg,
		collector:   collector,
		aggregator:  aggregator.NewMetricAggregator(cfg.Dimensions),
		client:      client,
		metricsChan: make(chan *metrics.ProcessMetrics, 100),
	}, nil
}

func (m *Monitor) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("Starting monitoring of %s process...", m.cfg.ProcessName)
	go m.aggregator.Start(ctx, m.metricsChan)
	go m.collector.Start(ctx, m.metricsChan, m.cfg.CollectInterval)

	flushTicker := time.NewTicker(m.cfg.FlushInterval)
	defer flushTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-flushTicker.C:
			datums := m.aggregator.Flush()
			err := m.client.PutMetricData(ctx, datums)
			if err != nil {
				log.Printf("Error sending metrics to CloudWatch: %v", err)
				continue
			}

			log.Printf("Metrics sent successfully: %s", cloudwatch.Prettify(datums))
		}
	}
}

func (m *Monitor) MetricsReporter() MetricsReporter {
	return m.aggregator
}
