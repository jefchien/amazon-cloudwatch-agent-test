// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"load-tester/generator/log"
	"load-tester/monitor"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Monitor       *monitor.Config `yaml:"monitor"`
	LogGenerators []*log.Config   `yaml:"log_generators"`
	Duration      time.Duration   `yaml:"duration"`
}

func (c *Config) Validate() error {
	if c.Monitor != nil {
		if err := c.Monitor.Validate(); err != nil {
			return err
		}
	}
	if c.LogGenerators != nil {
		for _, generator := range c.LogGenerators {
			if err := generator.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

func LoadConfig(path string) (*Config, error) {
	config := &Config{}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err = yaml.Unmarshal(content, config); err != nil {
		return nil, err
	}
	if err = config.Validate(); err != nil {
		return nil, err
	}
	return config, nil
}
