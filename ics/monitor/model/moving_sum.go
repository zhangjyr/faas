package model

type MovingSum struct {
	window int64
	n      int64
	values []float64
	last   int64
	sum    [2]float64
	active int
	resetting int
}

func NewMovingSum(window int64) *MovingSum {
	return &MovingSum{
		window : window,
		n: 0,
		values : make([]float64, window),
		last: 0,
		active: 0,
		resetting: 1,
	}
}

func (sum *MovingSum) Add(val float64) {
	// Move forward.
	sum.last = (sum.last + 1) % sum.window

	// Add difference to sum.
	sum.sum[sum.active] += val - sum.values[sum.last]
	sum.sum[sum.resetting] += val	// Resetting is used to sum from ground above each window interval.
	if sum.last == 0 {
		sum.active = sum.resetting
		sum.resetting = (sum.resetting + 1) % len(sum.sum)
		sum.sum[sum.resetting] = 0.0
	}

	// Record history value
	sum.values[sum.last] = val

	// update length
	if sum.n < sum.window {
		sum.n += 1
	}
}

func (sum *MovingSum) Sum() float64 {
	return sum.sum[sum.active]
}

func (sum *MovingSum) Window() int64 {
	return sum.window
}

func (sum *MovingSum) N() int64 {
	return sum.n
}

func (sum *MovingSum) Last() float64 {
	return sum.values[sum.last]
}

func (sum *MovingSum) LastN(n int64) float64 {
	if n > sum.n {
		n = sum.n
	}
	return sum.values[(sum.last + sum.window - sum.n) % sum.window]
}
