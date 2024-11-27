package stats

import (
	"log"
	"sync/atomic"
	"time"
)

// Stats
type Stats struct {
	NumberCalls       atomic.Int64
	TotalResponseTime atomic.Int64 // stored in NanoSeconds
}

type RenderedStats struct {
	NumberCalls     int64
	AvgResponseTime time.Duration
}

func (s *Stats) GetRenderedStats() *RenderedStats {
	return &RenderedStats{
		NumberCalls:     s.NumberCalls.Load(),
		AvgResponseTime: s.AvgResponseTime(),
	}
}

func (s *Stats) UpdateResponseTime(duration time.Duration) {
	s.TotalResponseTime.Add(duration.Nanoseconds()) // Add in nanoseconds
	s.NumberCalls.Add(1)
}

func (s *Stats) AvgResponseTime() time.Duration {
	total := s.TotalResponseTime.Load()
	count := s.NumberCalls.Load()
	if total == 0 {
		return 0
	}

	avg := total / count
	return time.Duration(avg)
}

func (s *Stats) LogStats() {
	log.Printf("Number of API Calls: %d", s.NumberCalls.Load())
	log.Printf("Average Response Time: %v", s.AvgResponseTime())
}
