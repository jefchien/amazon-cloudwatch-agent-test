// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package log

import (
	"sync"
	"time"
)

type Stats struct {
	bytesWritten   uint64
	entriesWritten uint64
	errors         uint64
	startTime      time.Time
	mu             sync.Mutex
}

func NewStats() *Stats {
	return &Stats{startTime: time.Now()}
}

func (s *Stats) Update(bytes int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.errors++
		return
	}
	s.bytesWritten += uint64(bytes)
	s.entriesWritten++
}

func (s *Stats) GetAndReset() (bytes, entries, errors uint64, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()

	bytes = s.bytesWritten
	entries = s.entriesWritten
	errors = s.errors
	duration = now.Sub(s.startTime)

	s.bytesWritten = 0
	s.entriesWritten = 0
	s.errors = 0
	s.startTime = now

	return
}
