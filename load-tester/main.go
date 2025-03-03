// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"flag"
	"load-tester/config"
	loggenerator "load-tester/generator/log"
	"load-tester/monitor"
	"log"
	"sync"
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

	var wg sync.WaitGroup

	wg.Add(1)
	if cfg.Duration == 0 {
		log.Print("Test will run until killed")
	} else {
		log.Printf("Test will stop after %v", cfg.Duration)
		go cancelAfter(ctx, &wg, cancel, cfg.Duration)
	}

	var m *monitor.Monitor
	if cfg.Monitor != nil {
		m, err = monitor.New(ctx, cfg.Monitor)
		if err != nil {
			log.Fatalf("Failed to create monitor: %v", err)
		}
		wg.Add(1)
		go m.Run(ctx, &wg)
	}

	if cfg.LogGenerator != nil {
		lg := loggenerator.New(cfg.LogGenerator)
		if m != nil {
			lg.SetReporter(m.MetricsReporter())
		}
		wg.Add(1)
		go lg.Run(ctx, &wg)
	}

	wg.Wait()
}

func cancelAfter(ctx context.Context, wg *sync.WaitGroup, cancel context.CancelFunc, d time.Duration) {
	defer wg.Done()
	timer := time.NewTimer(d)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			log.Printf("Stopping tester...")
			cancel()
		case <-ctx.Done():
			return
		}
	}
}
