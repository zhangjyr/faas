package monitor

import (
	"sync"
	"time"

	"github.com/openfaas/faas/ics/monitor/sampler"
	"github.com/openfaas/faas/ics/monitor/model"
	"github.com/openfaas/faas/ics/logger"
)

// var AnalysisWindow = 60
var DefaultSSEWindow int64 = 30

type LinearAnalyser struct {
	log                    logger.ILogger
	pause                  bool
	ySampler               sampler.Sampler
	xSampler               sampler.VariableSampler
	lastInterval           float64
	lastY, lastX           *sampler.Sample
	num                    float64
	x, y                   *model.Sum
	sx                     *model.Sum
	xy                     *model.Sum
	sst                    float64
	syv, sev, yv           *model.MovingSum	// Square of Y for Verfication, Square of Err for Verficatoin, and Y for Verfication

	// sync values
	mu                     sync.RWMutex
	a, b                   float64
	undetermination        float64
}

func NewLinearAnalyser(y sampler.Sampler, x sampler.VariableSampler) *LinearAnalyser {
	ana := &LinearAnalyser{
		ySampler: y,
		xSampler: x,
		pause: true,
		x: model.NewSum(),
		y: model.NewSum(),
		sx: model.NewSum(),
		xy: model.NewSum(),
		syv: model.NewMovingSum(DefaultSSEWindow),
		sev: model.NewMovingSum(DefaultSSEWindow),
		yv: model.NewMovingSum(DefaultSSEWindow),
	}
	return ana
}

func (ana *LinearAnalyser) Start() error {
	ana.pause = false
	return nil
}

func (ana *LinearAnalyser) Stop() error {
	ana.pause = true
	return nil
}

func (ana *LinearAnalyser) SetDebug(debug bool) {
	if debug {
		ana.log = &logger.ColorLogger{
			Level:       logger.LOG_LEVEL_ALL,
			Prefix:      "LinearAnalyser ",
			Color:       true,
		}
	} else {
		ana.log = nil
	}
}

func (ana *LinearAnalyser) Analyse(event *ResourceEvent) error {
	y, errY := ana.ySampler.Sample(event.Time)
	x, errX := ana.xSampler.Sample(event.Time)
	if errY == sampler.ErrNotEnoughData || errX == sampler.ErrNotEnoughData {
		return nil
	} else if errY != nil {
		return errY
	} else if errX != nil {
		return errX
	}

	if (ana.pause) {
		return nil
	}

	expected := ana.validate(y, x)
	ana.calculate(y, x)

	if ana.log != nil {
		ana.log.Debug("Sampled x:%f actural:%f expected:%f a:%f b:%f e:%f",
			x.Value, y.Value, expected, ana.a, ana.b, ana.undetermination)
	}

	return nil
}

func (ana *LinearAnalyser) Query(x float64) (float64, error) {
	ana.mu.RLock()
	defer ana.mu.RUnlock()

	if !ana.determinated() {
		return 0.0, ErrUndeterminate
	}
	sample := ana.xSampler.MakeVariable(x, time.Now())
	estimate, err := ana.queryLocked(sample)
	if err == nil && estimate > 2 {
		ana.log.Warn("Unusal estimate:%f, x:%f, sample.x:%f, lastInterval:%f, x.duration:%d",
			estimate, x, sample.Value, ana.lastInterval, sample.Time.Sub(ana.lastX.Time).Nanoseconds())
		return 1, ErrOverestimate
	} else {
		return estimate, err
	}
}

func (ana *LinearAnalyser) Determinated() bool {
	ana.mu.RLock()
	defer ana.mu.RUnlock()

	return ana.determinated()
}

// calculate a, b, sst
func (ana *LinearAnalyser) calculate(y, x *sampler.Sample) {
	ana.x.Add(x.Value)
	ana.y.Add(y.Value)
	ana.sx.Add(x.Value * x.Value)
	ana.xy.Add(x.Value * y.Value)
	ana.num = float64(ana.x.N())

	if ana.num < 2 {
		ana.lastY = y
		ana.lastX = x
		return
	}

	var b float64
	if ana.x.Sum() > 0 {
		b = (ana.num * ana.xy.Sum() - ana.x.Sum() * ana.y.Sum()) / (ana.num * ana.sx.Sum() - ana.x.Sum() * ana.x.Sum())
	}
	a := (ana.y.Sum() - b * ana.x.Sum()) / ana.num
	ana.sst = ana.syv.Sum() - ana.yv.Sum() * ana.yv.Sum() / float64(ana.yv.N())
	undetermination :=  ana.sev.Sum() / ana.sst

	ana.mu.Lock()
	ana.b = b
	ana.a = a
	ana.undetermination = undetermination
	ana.lastInterval = float64(x.Time.Sub(ana.lastX.Time).Nanoseconds())
	ana.lastY = y
	ana.lastX = x
	ana.mu.Unlock()
}

func (ana *LinearAnalyser) queryLocked(x *sampler.Sample) (float64, error) {
	if ana.num < 2 {
		return 0.0, ErrNotEnoughData
	}

	return (ana.a + ana.b * x.Value) * (ana.lastInterval / float64(x.Time.Sub(ana.lastX.Time).Nanoseconds())), nil
}


func (ana *LinearAnalyser) validate(y, x *sampler.Sample) float64 {
	ana.mu.RLock()
	expected, err := ana.queryLocked(x)
	ana.mu.RUnlock()

	if err == nil {
		e := y.Value - expected
		ana.syv.Add(y.Value * y.Value)
		ana.sev.Add(e * e)
		ana.yv.Add(y.Value)
	}
	return expected
}

func (ana *LinearAnalyser) determinated() bool {
	return ana.undetermination < 0.3
}
