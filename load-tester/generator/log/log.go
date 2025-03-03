// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package log

import (
	"context"
	"errors"
	"fmt"
	"load-tester/monitor"
	"load-tester/monitor/metrics/aggregator"
	"load-tester/monitor/metrics/distribution"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"golang.org/x/time/rate"
	"gopkg.in/natefinch/lumberjack.v2"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateRandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

type Generator struct {
	cfg      *Config
	logger   *lumberjack.Logger
	limiter  *rate.Limiter
	stats    *Stats
	reporter monitor.MetricsReporter
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
	//if g.reporter != nil {
	//    go g.reportFileSize(ctx)
	//}

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

func (g *Generator) reportFileSize(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sizes, err := g.getLogSizes()
			if err != nil {
				log.Printf("Error getting log sizes: %v", err)
				continue
			}
			var errs []error
			for name, size := range sizes {
				err = g.reporter.Add(aggregator.Key{
					Name: "FileSize",
					Dimensions: map[string]string{
						"FileName": name,
					},
				}, distribution.NewEntry(float64(size), types.StandardUnitBytes))
				if err != nil {
					errs = append(errs, err)
				}
			}
			if len(errs) > 0 {
				log.Printf("Error reporting log sizes: %v", errors.Join(errs...))
			}
		}
	}
}

func (g *Generator) getLogSizes() (map[string]int64, error) {
	dir := filepath.Dir(g.cfg.FilePath)
	base := filepath.Base(g.cfg.FilePath)
	ext := filepath.Ext(base)
	prefix := strings.TrimSuffix(base, ext)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := make(map[string]int64)
	var errs []error
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		var info os.FileInfo
		info, err = entry.Info()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		files[name] = info.Size()
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return files, nil
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
				_ = g.reporter.Add(aggregator.Key{
					Name: "WrittenBytes",
				}, distribution.NewEntry(float64(bytes), types.StandardUnitBytes))
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
	randomContent := generateRandomString(g.cfg.LineLength - len(timestamp) - 2)
	return fmt.Sprintf("%s %s\n", timestamp, randomContent)
}
