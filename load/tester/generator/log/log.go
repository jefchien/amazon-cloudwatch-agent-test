// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package log

import (
	"context"
	"fmt"
	"load-tester/generator/log/lumberjack"
	"load-tester/monitor"
	"load-tester/monitor/metrics/aggregator"
	"load-tester/monitor/metrics/distribution"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"golang.org/x/time/rate"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var (
	writtenBytesKey = aggregator.Key{Name: "WrittenBytes"}
	writtenEntries  = aggregator.Key{Name: "WrittenEntries"}
)

func generateRandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

type Generator struct {
	cfg            *Config
	logger         *lumberjack.Logger
	limiter        *rate.Limiter
	stats          *Stats
	reporter       monitor.MetricsReporter
	sequenceNumber atomic.Uint64
}

func New(cfg *Config) *Generator {
	return &Generator{
		cfg: cfg,
		logger: &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxFileSize,
			MaxAge:     1,
			MaxBackups: cfg.MaxFileCount,
		},
		limiter: rate.NewLimiter(rate.Limit(cfg.LinesPerSecond), cfg.MaxFileCount),
	}
}

func (g *Generator) SetReporter(reporter monitor.MetricsReporter) {
	g.reporter = reporter
}

func (g *Generator) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer g.logger.Close()

	time.Sleep(g.cfg.StartDelay)

	log.Printf("Starting log generation for %s...", g.cfg.FilePath)

	g.stats = NewStats()
	go g.reportStats(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := g.limiter.Wait(ctx); err != nil {
				continue
			}
			if err := g.writeEntry(); err != nil {
				log.Printf("Error writing log entry to file: %v", err)
			}
		}
	}
}

func (g *Generator) reportStats(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			bytes, entries, errs, duration := g.stats.GetAndReset()
			log.Printf("Stats for last minute: wrote %d entries (%d bytes) with %d errors"+
				"(%.2f entries/sec, %.2f bytes/sec)", entries, bytes, errs,
				float64(entries)/duration.Seconds(), float64(bytes)/duration.Seconds())
			if g.reporter != nil {
				_ = g.reporter.Add(writtenBytesKey, distribution.NewEntry(float64(bytes), types.StandardUnitBytes))
				_ = g.reporter.Add(writtenEntries, distribution.NewEntry(float64(entries), types.StandardUnitCount))
			}
		}
	}
}

func (g *Generator) writeEntry() error {
	n, err := g.logger.Write([]byte(g.generateLogEntry()))
	g.stats.Update(n, err)
	return err
}

func (g *Generator) generateLogEntry() string {
	timestamp := time.Now().Format(g.cfg.TimestampFormat)
	sequenceNumber := g.sequenceNumber.Add(1)
	randomContent := generateRandomString(g.cfg.LineLength - len(timestamp) - 20)
	return fmt.Sprintf("%s seq=%d %s\n", timestamp, sequenceNumber, randomContent)
}
