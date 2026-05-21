package warning

import "sync"

// Sink collects and drains warning messages.
type Sink struct {
	mu       sync.Mutex
	warnings []string
}

// SinkInterface defines operations on the warning sink.
type SinkInterface interface {
	Add(msg string)
	Drain() []string
}

// NewSink creates a new warning sink.
func NewSink() *Sink {
	return &Sink{}
}

// Add appends a warning message.
func (s *Sink) Add(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.warnings = append(s.warnings, msg)
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