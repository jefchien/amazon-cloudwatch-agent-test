// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"flag"
	"log"
	"test-runner/config"
	"test-runner/util"
	"time"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager, err := cfg.Process.Manager()
	if err != nil {
		log.Fatalf("Failed to get process manager: %v", err)
	}

	var testCases []*config.TestCase
	for _, t := range cfg.Tests {
		var tc []*config.TestCase
		tc, err = cfg.GenerateTestCases(t)
		if err != nil {
			log.Fatalf("Failed to generate test cases: %v", err)
		}
		testCases = append(testCases, tc...)
	}

	for i, testCase := range testCases {
		testID := i + 1
		log.Printf("Starting test %d of %d (%+v)", testID, len(testCases), testCase.TemplateData)
		if err = util.ClearDir(cfg.TestDataDir); err != nil {
			log.Fatalf("Failed to clear dir: %v", err)
		}
		if err = manager.Reconfigure(testCase.ProcessConfigs); err != nil {
			log.Fatalf("Failed to reconfigure test %d: %v", testID, err)
		}
		time.Sleep(cfg.SetupDelay)
		start := time.Now()
		if err = cfg.LoadTester.Run(ctx, testCase.LoadTesterConfig); err != nil {
			log.Fatalf("Failed to run test %d: %v", testID, err)
		}
		if err = manager.Stop(); err != nil {
			log.Fatalf("Failed to stop process %d: %v", testID, err)
		}
		end := time.Now()
		log.Printf("Test %d ran from %s to %s", testID, start, end)
		time.Sleep(cfg.NextTestDelay)
	}

	log.Printf("%d test cases run", len(testCases))
}
