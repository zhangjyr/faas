package model

type Sumer interface {
	Add(float64)
	Sum() float64
	N() int64
}

type Sum struct {
	n      int64
	sum    float64
}

func NewSum() *Sum {
	return &Sum{
		n: 0,
		sum: 0.0,
	}
}

func (sum *Sum) Add(val float64) {
	sum.sum += val
	sum.n += 1
}

func (sum *Sum) Sum() float64 {
	return sum.sum
}

func (sum *Sum) N() int64 {
	return sum.n
}
