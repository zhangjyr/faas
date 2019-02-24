package sampler

import (
	// "time"

	"github.com/openfaas/faas/ics/proxy"
)

// ResponseSampler samples every N response
type ResponseSampler struct {
	proxy         *proxy.Server
	sampler       chan *Sample
}

func NewResponseSampler(p *proxy.Server) *ResponseSampler {
	return &ResponseSampler{
		proxy: p,
	}
}

// func (s *RequestSampler) Sample(ts time.Time) (*Sample, error) {
// 	stats := s.proxy.RequestStats()
//
// 	var ret *Sample
// 	var err error
// 	first := s.lastStats == nil
// 	if !first {
// 		ret = s.MakeVariable(float64(stats.Served), stats.Time)
// 	} else {
// 		err = ErrNotEnoughData
// 	}
// 	s.lastStats = stats
// 	return ret, err
// }
//
// func (s *RequestSampler) MakeVariable(num float64, ts time.Time) *Sample {
// 	return &Sample{
// 		Value: num - float64(s.lastStats.Served),
// 		Time: ts,
// 	}
// }
