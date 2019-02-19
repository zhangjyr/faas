package sampler

import (
	"time"
)

// Provide a simple cache for latest sample
type SamplerCache struct {
	last       *Sample
	lastErr    error
	lastTime   *time.Time
	sampler    Sampler
}

func NewSamplerCache(s Sampler) Sampler {
	return &SamplerCache{
		sampler: s,
	}
}

func (s *SamplerCache) Sample(ts time.Time) (*Sample, error) {
	if s.lastTime == nil || !s.lastTime.Equal(ts) {
		s.last, s.lastErr = s.sampler.Sample(ts)
		s.lastTime = &ts
	}
	return s.last, s.lastErr
}
