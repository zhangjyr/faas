package monitor

// import (
// 	"sync"
// 	"time"
//
// 	"github.com/openfaas/faas/ics/monitor/sampler"
// 	"github.com/openfaas/faas/ics/monitor/model"
// 	"github.com/openfaas/faas/ics/logger"
// )
//
// // var AnalysisWindow = 60
// var DefaultSSEWindow = 30
//
// type LatencyReporter struct {
// 	log                    logger.ILogger
// 	pause                  bool
// 	ySampler               sampler.Sampler
// 	xSampler               sampler.VariableSampler
// 	lastInterval           float64
// 	lastY, lastX           *sampler.Sample
// 	num                    float64
// 	x, y                   *model.Sum
// 	sx                     *model.Sum
// 	xy                     *model.Sum
// 	sst                    float64
// 	syv, sev, yv           *model.MovingSum	// Square of Y for Verfication, Square of Err for Verficatoin, and Y for Verfication
//
// 	// sync values
// 	mu                     sync.RWMutex
// 	a, b                   float64
// 	undetermination        float64
// }
