// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package metrics

import (
	"context"
	"errors"
	"fmt"
	"load-tester/monitor/metrics/distribution"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/shirou/gopsutil/v3/process"
)

type ProcessMetrics struct {
	CPUPercent    float64
	MemoryPercent float64
	MemoryRSS     uint64
	MemoryVMS     uint64
	NumFDs        int32
	Threads       int32
	IOReadBytes   uint64
}

func (m *ProcessMetrics) Entries() map[string]distribution.Entry {
	return map[string]distribution.Entry{
		"CPUPercent":          distribution.NewEntry(m.CPUPercent, types.StandardUnitPercent),
		"MemoryPercent":       distribution.NewEntry(m.MemoryPercent, types.StandardUnitPercent),
		"MemoryRSS":           distribution.NewEntry(float64(m.MemoryRSS), types.StandardUnitBytes),
		"MemoryVMS":           distribution.NewEntry(float64(m.MemoryVMS), types.StandardUnitBytes),
		"OpenFileDescriptors": distribution.NewEntry(float64(m.NumFDs), types.StandardUnitCount),
		"ThreadCount":         distribution.NewEntry(float64(m.Threads), types.StandardUnitCount),
		"IOReadBytes":         distribution.NewEntry(float64(m.IOReadBytes), types.StandardUnitBytes),
	}
}

type Collector struct {
	processName string
	proc        *process.Process
}

func NewCollector(processName string) (*Collector, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("error getting processes: %v", err)
	}

	for _, p := range processes {
		name, err := p.Name()
		if err != nil {
			continue
		}
		if name == processName {
			return &Collector{
				processName: processName,
				proc:        p,
			}, nil
		}
	}

	return nil, fmt.Errorf("process %s not found", processName)
}

func (c *Collector) Start(ctx context.Context, metricsChan chan<- *ProcessMetrics, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics, err := c.Collect()
			if err != nil {
				log.Printf("Error collecting metrics: %v", err)
				continue
			}
			metricsChan <- metrics
		}
	}
}

func (c *Collector) Collect() (*ProcessMetrics, error) {
	m := &ProcessMetrics{}
	return m, errors.Join(c.collectCPU(m), c.collectMemory(m), c.collectFD(m), c.collectThreads(m), c.collectIO(m))
}

func (c *Collector) collectCPU(m *ProcessMetrics) error {
	cpuPercent, err := c.proc.CPUPercent()
	if err != nil {
		return fmt.Errorf("error getting CPU percent: %v", err)
	}
	m.CPUPercent = cpuPercent
	return nil
}

func (c *Collector) collectMemory(m *ProcessMetrics) error {
	var errs []error
	memInfo, err := c.proc.MemoryInfo()
	if err != nil {
		errs = append(errs, fmt.Errorf("error getting memory info: %w", err))
	} else {
		m.MemoryRSS = memInfo.RSS
		m.MemoryVMS = memInfo.VMS
	}
	memPercent, err := c.proc.MemoryPercent()
	if err != nil {
		errs = append(errs, fmt.Errorf("error getting memory percent: %w", err))
	} else {
		m.MemoryPercent = float64(memPercent)
	}
	return errors.Join(errs...)
}

func (c *Collector) collectFD(m *ProcessMetrics) error {
	fds, err := c.proc.NumFDs()
	if err != nil {
		return fmt.Errorf("error getting FD count: %w", err)
	}
	m.NumFDs = fds
	return nil
}

func (c *Collector) collectThreads(m *ProcessMetrics) error {
	threads, err := c.proc.NumThreads()
	if err != nil {
		return fmt.Errorf("error getting threads: %w", err)
	}
	m.Threads = threads
	return nil
}

func (c *Collector) collectIO(m *ProcessMetrics) error {
	io, err := c.proc.IOCounters()
	if err != nil {
		return fmt.Errorf("error getting I/O: %w", err)
	}
	m.IOReadBytes = io.ReadBytes
	return nil
}
