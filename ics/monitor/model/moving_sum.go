package model

type MovingSum struct {
	window int
	n int
	values []float64
	last   int
	sum    float64
}

func NewMovingSum(window int) *MovingSum {
	return &MovingSum{
		window : window,
		n: 0.0,
		values : make([]float64, window),
		last: 0.0,
		sum: 0.0,
	}
}

func (sum *MovingSum) Add(val float64) float64 {
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

	return sum.sum
}

func (sum *MovingSum) Sum() float64 {
	return sum.sum
}

func (sum *MovingSum) Window() int {
	return sum.window
}

func (sum *MovingSum) N() int {
	return sum.n
}

func (sum *MovingSum) Last() float64 {
	return sum.values[sum.last]
}

func (sum *MovingSum) LastN(n int) float64 {
	if n > sum.n {
		n = sum.n
	}
	return sum.values[(sum.last + sum.window - sum.n) % sum.window]
}
