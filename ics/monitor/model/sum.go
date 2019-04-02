package model

type Sum struct {
	n      int
	sum    float64
}

func NewSum() *Sum {
	return &Sum{
		n: 0,
		sum: 0.0,
	}
}

func (sum *Sum) Add(val float64) float64 {
	sum.sum += val
	sum.n += 1

	return sum.sum
}

func (sum *Sum) Sum() float64 {
	return sum.sum
}

func (sum *Sum) N() int {
	return sum.n
}
