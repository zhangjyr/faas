package model

type MovingSum struct {
	window int64
	n      int64
	values []float64
	last   int64
	sum    float64
}

func NewMovingSum(window int64) *MovingSum {
	return &MovingSum{
		window : window,
		n: 0,
		values : make([]float64, window),
		last: 0.0,
		sum: 0.0,
	}
}

func (sum *MovingSum) Add(val float64) {
	// Move forward.
	sum.last = (sum.last + 1) % sum.window

	// Substract last value.
	sum.sum -= sum.values[sum.last]

	// Record history value
	sum.values[sum.last] = val

	// Add val to sum
	sum.sum += val

	// update length
	if sum.n < sum.window {
		sum.n += 1
	}
}

func (sum *MovingSum) Sum() float64 {
	return sum.sum
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
