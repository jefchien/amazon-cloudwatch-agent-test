// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

type LoadTester struct {
	Binary  string        `yaml:"binary_path"`
	LogPath string        `yaml:"log_path"`
	Timeout time.Duration `yaml:"timeout"`
}

func (lt *LoadTester) Run(ctx context.Context, configPath string) error {
	// Create/truncate the log file
	logFile, err := os.OpenFile(lt.LogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create/open log file: %w", err)
	}
	defer logFile.Close()

	// Prepare the command
	cmd := exec.CommandContext(ctx, lt.Binary, "-config", configPath)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Start the command
	if err = cmd.Start(); err != nil {
		return fmt.Errorf("failed to start load tester: %w", err)
	}

	// Create a channel for the command completion
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Wait for either completion or timeout
	select {
	case err = <-done:
		if err != nil {
			return fmt.Errorf("load tester failed: %w", err)
		}
	case <-ctx.Done():
		// Try to gracefully stop the process first
		if cmd.Process != nil {
			cmd.Process.Signal(os.Interrupt)
			// Give it a short time to cleanup
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				// Force kill if it doesn't stop
				cmd.Process.Kill()
			}
		}
		return ctx.Err()
	case <-time.After(lt.Timeout):
		// Duration completed, stop the process
		if cmd.Process != nil {
			cmd.Process.Signal(os.Interrupt)
			// Give it a short time to cleanup
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				// Force kill if it doesn't stop
				cmd.Process.Kill()
			}
		}
	}

	return nil
}
