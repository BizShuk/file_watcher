package svc

import "sync"

// Sink collects and drains warning messages.
type Sink struct {
	mu       sync.Mutex
	warnings []string
}

// NewSink creates a new warning sink.
func NewSink() *Sink {
	return &Sink{}
}

// Drain returns all collected warnings and resets the sink.
func (s *Sink) Drain() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	res := make([]string, len(s.warnings))
	copy(res, s.warnings)
	s.warnings = nil
	return res
}
