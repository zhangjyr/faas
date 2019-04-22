package monitor

import (
	"math"

	"github.com/openfaas/faas/ics/monitor/model"
	"github.com/openfaas/faas/ics/logger"
	"github.com/openfaas/faas/ics/utils/channel"
)

type LatencyReporter struct {
	log        logger.ILogger
	stats      *model.LightStats
	feed       channel.Out
}

func NewLatencyReporter(feed channel.Out) *LatencyReporter {
	ana := &LatencyReporter{
		log: logger.NilLogger,
		stats: model.NewMovingLightStats(1, 10), // 10ms
		feed: feed,
	}
	return ana
}

func (ana *LatencyReporter) Start() error {
	if ana.feed != nil {
		ana.feed.Pipe(ana.stats.ChanAdd())
	}
	return nil
}

func (ana *LatencyReporter) Stop() error {
	if ana.feed != nil {
		ana.feed.StopPipe()
	}
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
		ana.log = logger.NilLogger
	}
}

func (ana *LatencyReporter) PipeFrom(feed channel.Out) {
	ana.feed = feed
}

func (ana *LatencyReporter) Analyse(event *ResourceEvent) error {
	if ana.log != logger.NilLogger {
		_, mean, var2 := ana.stats.NMeanVar2()
		ana.log.Debug("Worker latency mean:%f var:%f", mean, math.Sqrt(var2))
	}

	return nil
}

func (ana *LatencyReporter) Query(float64) (float64, error) {
	return ana.stats.Mean(), nil
}
