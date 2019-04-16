package monitor

import (
	"math"

	"github.com/openfaas/faas/ics/monitor/model"
	"github.com/openfaas/faas/ics/logger"
)

type LatencyReporter struct {
	log                    logger.ILogger
	pause                  bool
	stats                  *model.LightStats
}

func NewLatencyReporter() *LatencyReporter {
	ana := &LatencyReporter{
		pause: true,
		stats: model.NewMovingLightStats(60), // 60s
	}
	return ana
}

func (ana *LatencyReporter) Start() error {
	ana.pause = false
	return nil
}

func (ana *LatencyReporter) Stop() error {
	ana.pause = true
	return nil
}

func (ana *LatencyReporter) SetDebug(debug bool) {
	if debug {
		ana.log = &logger.ColorLogger{
			Level:       logger.LOG_LEVEL_ALL,
			Prefix:      "StatsAnalyser ",
			Color:       true,
		}
	} else {
		ana.log = nil
	}
}

func (ana *LatencyReporter) PipeFrom(feed <-chan interface{}) {
	for i := range feed {
		if !ana.pause {
			ana.stats.Add(i.(float64))
		}
	}
}

func (ana *LatencyReporter) Analyse(event *ResourceEvent) error {
	_, mean, var2 := ana.stats.NMeanVar2()
	if ana.log != nil {
		ana.log.Debug("Worker latency mean:%f var:%f", mean, math.Sqrt(var2))
	}

	return nil
}

func (ana *LatencyReporter) Query(float64) (float64, error) {
	return ana.stats.Mean(), nil
}
