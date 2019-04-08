package model

import (
	// "log"
	"sync"
	"sync/atomic"
	"time"
)

const SwapOnTimeout = 0
const SwapOnFull = 1

const len_buffers = 2
const buffer_size int64 = 10000

var (
	timerInterval time.Duration = 1 * time.Millisecond
	buffer_masks = [buffer_size]uint64{
		0x0001, 0x0003, 0x0007, 0x000F, 0x001F, 0x003F, 0x007F, 0x00FF,
		0x01FF, 0x03FF, 0x07FF, 0x0FFF, 0x1FFF, 0x3FFF, 0x7FFF, 0xFFFF,
	}
)

type lightStatsSafeBucket struct {
	n        int64
	filled   uint64
	sum      int64
	sum2     int64
}

func (bucket *lightStatsSafeBucket) add(val float64) bool {
	// Assign bucket slot.
	slot := atomic.AddInt64(&bucket.n, 1)

	// Fail if not enough slots.
	if (slot > buffer_size) {
		return false
	}

	// Add up and flag slot as filled.
	atomic.AddInt64(&bucket.sum, int64(val * 1e9))
	atomic.AddInt64(&bucket.sum2, int64(val * val * 1e18))
	atomic.AddUint64(&bucket.filled, 0x0001 << (uint64(slot) - 1))
	return true
}

func (bucket *lightStatsSafeBucket) isSafe(n int64) bool {
	if n > buffer_size {
		n = buffer_size
	}
	return (atomic.LoadUint64(&bucket.filled) & buffer_masks[n - 1]) == buffer_masks[n - 1]
}

func (bucket *lightStatsSafeBucket) load() (int64, float64, float64) {
	n := bucket.n
	if n > buffer_size {
		n = buffer_size
	}
	return n, float64(bucket.sum) / 1e9, float64(bucket.sum2) / 1e18
}

func (bucket *lightStatsSafeBucket) reset() {
	atomic.StoreInt64(&bucket.n, 0)
	atomic.StoreUint64(&bucket.filled, 0)
	atomic.StoreInt64(&bucket.sum, 0)
	atomic.StoreInt64(&bucket.sum2, 0)
}

type lightStatsBucket struct {
	n        int64
	sum      float64
	sum2     float64
}

func (bucket *lightStatsBucket) add(val float64) bool {
	if bucket.n == buffer_size {
		return false
	}

	bucket.n += 1
	bucket.sum += val
	bucket.sum2 += val * val
	return true
}

func (bucket *lightStatsBucket) isSafe(n int64) bool {
	return true
}

func (bucket *lightStatsBucket) load() (int64, float64, float64) {
	return bucket.n, bucket.sum, bucket.sum2
}

func (bucket *lightStatsBucket) reset() {
	bucket.n = 0
	bucket.sum = 0.0
	bucket.sum2 = 0.0
}

type LightStats struct {
	OnBlock  func()
	OnSwap   func(int, int)
	OnFailToSwap func(int)

	window   int64
	n        Sumer
	x        Sumer
	x2       Sumer
	buffers  [len_buffers]lightStatsBucket
	active   int32
	mean     float64
	var2     float64
	changed  time.Time
	updated  time.Time
	timer    *time.Timer
	mu       sync.RWMutex
	closed   chan struct{}
	readable chan float64
	flushable chan struct{}
}

func NewLightStats() *LightStats {
	stats := &LightStats{
		n: NewSum(),
		x: NewSum(),
		x2 : NewSum(),
		closed: make(chan struct{}),
		readable: make(chan float64, len_buffers * buffer_size),
		flushable: make(chan struct{}, len_buffers),
	}
	for i := 1; i < len_buffers; i++ {
		stats.flushable <- struct{}{}
	}
	go stats.initTimer()
	return stats
}

func NewMovingLightStats(window int64) *LightStats {
	stats := &LightStats{
		window : window,
		n: NewMovingSum(window),
		x: NewMovingSum(window),
		x2 : NewMovingSum(window),
		closed: make(chan struct{}),
	}
	go stats.initTimer()
	return stats
}

func (stats *LightStats) Add(val float64) {
	stats.readable <- val
	// stats.add(val)
}

func (stats *LightStats) N() int64 {
	return int64(stats.n.Sum())
}

func (stats *LightStats) Sum() float64 {
	return stats.x.Sum()
}

func (stats *LightStats) Mean() float64 {
	stats.calculate()
	return stats.mean
}

func (stats *LightStats) Var2() float64 {
	stats.calculate()
	return stats.var2
}

func (stats *LightStats) NMeanVar2() (int64, float64, float64) {
	stats.calculate()
	stats.mu.RLock()
	defer stats.mu.RUnlock()
	return int64(stats.n.Sum()), stats.mean, stats.var2
}

func (stats *LightStats) Close() {
	select {
	case <-stats.closed:
		// already closed.
	default:
		close(stats.closed)
	}
}

func (stats *LightStats) add(val float64) {
	active := atomic.LoadInt32(&stats.active)
	// active := stats.active
	for !stats.buffers[active % len_buffers].add(val) {
		stats.swap(active, SwapOnFull)
		active = atomic.LoadInt32(&stats.active)
		// active = stats.active
	}
}

func (stats *LightStats) swap(active int32, full int) bool {
	select {
	case <- stats.flushable:
	default:
		if stats.OnBlock != nil {
			stats.OnBlock()
		}
		<- stats.flushable
	}
	if !atomic.CompareAndSwapInt32(&stats.active, active, active + 1) {
		if stats.OnFailToSwap != nil {
			stats.OnFailToSwap(full)
		}
		return false
	}
	// stats.active = active + 1

	go stats.flush(active, full)
	return true
}

func (stats *LightStats) flush(active int32, full int) {
	flushing := &stats.buffers[active % len_buffers];
	n, x, x2 := flushing.n, 0.0, 0.0

	if n > 0 {
		for !flushing.isSafe(n) {
			time.Sleep(1 * time.Millisecond)
		}

		stats.mu.Lock()
		defer stats.mu.Unlock()
		n, x, x2 = flushing.load()
		stats.n.Add(float64(n))
		stats.x.Add(x)
		stats.x2.Add(x2)
		flushing.reset()
		stats.changed = time.Now()
	}

	if stats.OnSwap != nil {
		stats.OnSwap(full, int(n))
	}
	stats.flushable <- struct{}{}
	stats.resetTimer()
}

func (stats *LightStats) calculate() {
	if stats.changed != stats.updated {
		stats.mu.Lock()
		defer stats.mu.Unlock()

		if stats.changed != stats.updated {
			stats.mean = stats.x.Sum() / stats.n.Sum()
			stats.var2 = (stats.n.Sum() * stats.x2.Sum() - stats.x.Sum() * stats.x.Sum()) /
				(stats.n.Sum() * (stats.n.Sum() - 1))
			stats.updated = stats.changed
		}
	}
}

func (stats *LightStats) initTimer() {
	stats.timer = time.NewTimer(0 * time.Millisecond)
	defer stats.timer.Stop()
	for {
		select {
		case <-stats.closed:
			// Stop monitoring when signaled.
			return
		case <-stats.timer.C:
			stats.swap(stats.active, SwapOnTimeout)
		case val := <-stats.readable:
			stats.add(val)
		}
	}
}

func (stats *LightStats) resetTimer() {
	if stats.timer == nil {
		return
	}

	// Drain the timer to be accurate and safe to reset.
	if !stats.timer.Stop() {
		select {
		case <-stats.timer.C:
		default:
		}
	}
	stats.timer.Reset(timerInterval)
}
