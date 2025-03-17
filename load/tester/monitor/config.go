// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package monitor

import (
	"fmt"
	"time"
)

type Config struct {
	ProcessName      string            `yaml:"process_name"`
	MetricsNamespace string            `yaml:"metrics_namespace"`
	CollectInterval  time.Duration     `yaml:"collect_interval"`
	FlushInterval    time.Duration     `yaml:"flush_interval"`
	Region           string            `yaml:"region"`
	Dimensions       map[string]string `yaml:"dimensions"`
}

func (c *Config) Validate() error {
	if c.ProcessName == "" {
		return fmt.Errorf("process_name is required")
	}
	if c.MetricsNamespace == "" {
		return fmt.Errorf("metrics_namespace is required")
	}
	if c.Region == "" {
		return fmt.Errorf("region is required")
	}
	if c.CollectInterval < time.Second {
		return fmt.Errorf("collect_interval must be at least a second")
	}
	if c.CollectInterval > c.FlushInterval {
		return fmt.Errorf("collect_interval must be less than flush_interval")
	}
	return nil
}
