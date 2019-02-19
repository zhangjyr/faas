package sampler

import (
	"time"

	"github.com/openfaas/faas/ics/proxy"
)

// RequestSampler samples requested and served every interval, the number of requests the proxy can actually served
// falls in range [end.served - start.served, end.started - start.served]. By taking minium value in the range, we
// underestimated the real value, and hence to overestimate the utilization per request for precaution.
type RequestSampler struct {
	proxy         *proxy.Server
	lastStats     *proxy.Stats
}

func NewRequestSampler(p *proxy.Server) *RequestSampler {
	return &RequestSampler{
		proxy: p,
	}
}

func (s *RequestSampler) Sample(ts time.Time) (*Sample, error) {
	stats := s.proxy.RequestStats()

	var ret *Sample
	var err error
	first := s.lastStats == nil
	if !first {
		ret = s.MakeVariable(float64(stats.Served), stats.Time)
	} else {
		err = ErrNotEnoughData
	}
	s.lastStats = stats
	return ret, err
}

func (s *RequestSampler) MakeVariable(num float64, ts time.Time) *Sample {
	return &Sample{
		Value: num - float64(s.lastStats.Served),
		Time: ts,
	}
}
