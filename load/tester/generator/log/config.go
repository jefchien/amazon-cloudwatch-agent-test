// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package log

import (
	"fmt"
	"time"
)

type Config struct {
	FilePath        string        `yaml:"file_path"`
	TimestampFormat string        `yaml:"timestamp_format"`
	BytesPerSecond  int           `yaml:"bytes_per_second"`
	LineLength      int           `yaml:"line_length"`
	BatchSize       int           `yaml:"batch_size"`
	MaxFileSize     int           `yaml:"max_file_size"`
	MaxFileCount    int           `yaml:"max_file_count"`
	StartDelay      time.Duration `yaml:"start_delay"`
}

func (c *Config) Validate() error {
	if c.FilePath == "" {
		return fmt.Errorf("file_path is required")
	}
	if c.TimestampFormat == "" {
		return fmt.Errorf("timestamp_format is required")
	}
	if c.BytesPerSecond <= 0 {
		return fmt.Errorf("bytes_per_second must be greater than 0")
	}
	if c.LineLength <= 0 {
		return fmt.Errorf("line_length must be greater than 0")
	}
	if c.BatchSize <= 0 {
		return fmt.Errorf("batch_size must be greater than 0")
	}
	if c.MaxFileSize <= 0 {
		return fmt.Errorf("max_file_size must be greater than 0")
	}
	if c.MaxFileCount <= 0 {
		return fmt.Errorf("max_file_size must be greater than 0")
	}
	return nil
}
