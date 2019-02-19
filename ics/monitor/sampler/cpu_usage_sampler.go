package sampler

import (
	"time"
)

type CPUUsageSampler struct {
	lastUsage     *Sample
}

var cpuUsageSampler    Sampler

func CPUUsageSamplerInstance() Sampler {
	if cpuUsageSampler == nil {
		cpuUsageSampler = NewSamplerCache(&CPUUsageSampler{})
	}
	return cpuUsageSampler
}

func (s *CPUUsageSampler) Sample(ts time.Time) (*Sample, error) {
	ts, usage, err := getCgroupParamUint("/sys/fs/cgroup/cpuacct", "cpuacct.usage")
	if err != nil {
		return nil, err
	}

	var ret *Sample
	sample := &Sample{
		Value: float64(usage),
		Time: ts,
	}
	first := s.lastUsage == nil
	if !first {
		ret = sample.NewRate(s.lastUsage)
	} else {
		err = ErrNotEnoughData
	}
	s.lastUsage = sample
	return ret, err
}
