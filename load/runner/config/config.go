// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"test-runner/process"
	"test-runner/test"
	"time"

	"gopkg.in/yaml.v3"
)

type TestCase struct {
	TemplateData
	LoadTesterConfig string
	ProcessConfigs   []string
}

type Test struct {
	TPS                      []int         `yaml:"tps"`
	ThreadCounts             []int         `yaml:"thread_counts"`
	FileCount                int           `yaml:"file_count"`
	Duration                 time.Duration `yaml:"duration"`
	LoadTesterConfigTemplate string        `yaml:"load_tester_config_template"`
	ProcessConfigTemplates   []string      `yaml:"process_config_templates"`
}

func (t Test) Validate() error {
	if len(t.TPS) == 0 {
		return fmt.Errorf("no tps specified")
	}
	if len(t.ThreadCounts) == 0 {
		return fmt.Errorf("no thread_counts specified")
	}
	if t.FileCount == 0 {
		return fmt.Errorf("no file_count specified")
	}
	if t.Duration == 0 {
		return fmt.Errorf("no duration specified")
	}
	if t.LoadTesterConfigTemplate == "" {
		return fmt.Errorf("missing load_tester_config_template")
	}
	if len(t.ProcessConfigTemplates) == 0 {
		return fmt.Errorf("no process config files specified")
	}
	return nil
}

type Config struct {
	Process       process.Process `yaml:"process"`
	ProcessAlias  string          `yaml:"process_alias"`
	ConfigDir     string          `yaml:"config_dir"`
	TestDataDir   string          `yaml:"test_data_dir"`
	Tests         []Test          `yaml:"tests"`
	LoadTester    test.LoadTester `yaml:"load_tester"`
	SetupDelay    time.Duration   `yaml:"setup_delay"`
	NextTestDelay time.Duration   `yaml:"next_test_delay"`
}

func (c Config) Validate() error {
	for i, t := range c.Tests {
		if err := t.Validate(); err != nil {
			return fmt.Errorf("invalid configuration for test %d: %w", i, err)
		}
	}
	return nil
}

func (c Config) GenerateTestCases(t Test) ([]*TestCase, error) {
	var testCases []*TestCase
	for _, threadCount := range t.ThreadCounts {
		for _, tps := range t.TPS {
			testDir := filepath.Join(c.ConfigDir, fmt.Sprintf("t%d_tps%d_f%d", threadCount, tps, t.FileCount))
			if err := os.MkdirAll(testDir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create config dir: %w", err)
			}
			testCase := &TestCase{
				TemplateData: TemplateData{
					ProcessName:     c.Process.String(),
					ProcessAlias:    c.ProcessAlias,
					ThreadCount:     threadCount,
					HalfThreadCount: threadCount / 2,
					TPS:             tps,
					FileCount:       t.FileCount,
					Duration:        t.Duration.String(),
				},
				LoadTesterConfig: filepath.Join(testDir, filepath.Base(t.LoadTesterConfigTemplate)),
				ProcessConfigs:   make([]string, 0, len(t.ProcessConfigTemplates)),
			}
			if err := processTemplate(t.LoadTesterConfigTemplate, testCase.LoadTesterConfig, testCase.TemplateData); err != nil {
				return nil, fmt.Errorf("failed to create load test config: %w", err)
			}
			for _, tmpl := range t.ProcessConfigTemplates {
				processConfig := filepath.Join(testDir, filepath.Base(tmpl))
				if err := processTemplate(tmpl, processConfig, testCase.TemplateData); err != nil {
					return nil, fmt.Errorf("failed to create process test config: %w", err)
				}
				testCase.ProcessConfigs = append(testCase.ProcessConfigs, processConfig)
			}
			testCases = append(testCases, testCase)
		}
	}
	return testCases, nil
}

func LoadConfig(path string) (*Config, error) {
	config := &Config{
		SetupDelay:    5 * time.Second,
		NextTestDelay: 2 * time.Minute,
	}
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
