package monitor

import (
	"sync"
	"time"

	"github.com/openfaas/faas/ics/monitor/sampler"
	"github.com/openfaas/faas/ics/monitor/model"
	"github.com/openfaas/faas/ics/logger"
)

var DefaultSSEWindow = 1000

type LinearAnalyser struct {
	log                    logger.ILogger
	pause                  bool
	ySampler               sampler.Sampler
	xSampler               sampler.VariableSampler
	lastInterval           float64
	lastY, lastX           *sampler.Sample
	num                    float64
	sumx, sumy             float64
	ssx                    float64
	sumxy                  float64
	mSST                   float64
	mSSY, mSSE, mSumY      *model.MovingSum

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
		mSSY: model.NewMovingSum(DefaultSSEWindow),
		mSSE: model.NewMovingSum(DefaultSSEWindow),
		mSumY: model.NewMovingSum(DefaultSSEWindow),
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

func (ana *LinearAnalyser) Sample(event *ResourceEvent) error {
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
	return ana.queryLocked(ana.xSampler.MakeVariable(x, time.Now()))
}

func (ana *LinearAnalyser) Determinated() bool {
	ana.mu.RLock()
	defer ana.mu.RUnlock()

	return ana.determinated()
}

// calculate a, b, sst
func (ana *LinearAnalyser) calculate(y, x *sampler.Sample) {
	ana.num += 1
	ana.sumx += x.Value
	ana.sumy += y.Value
	ana.ssx += x.Value * x.Value
	ana.sumxy += x.Value * y.Value

	if ana.num < 2 {
		ana.lastY = y
		ana.lastX = x
		return
	}

	var b float64
	if ana.sumx > 0 {
		b = (ana.num * ana.sumxy - ana.sumx * ana.sumy) / (ana.num * ana.ssx - ana.sumx * ana.sumx)
	}
	a := (ana.sumy - b * ana.sumx) / ana.num
	ana.mSST = ana.mSSY.Sum() - ana.mSumY.Sum() * ana.mSumY.Sum() / float64(ana.mSumY.N())
	undetermination :=  ana.mSSE.Sum() / ana.mSST

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
		ana.mSSY.Add(y.Value * y.Value)
		ana.mSSE.Add(e * e)
		ana.mSumY.Add(y.Value)
	}
	return expected
}

func (ana *LinearAnalyser) determinated() bool {
	return ana.undetermination < 0.2
}
