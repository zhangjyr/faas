package monitor

import (
	"errors"
)

var (
	ErrNotEnoughData = errors.New("not enough data")
	ErrUndeterminate = errors.New("the result is undeterminated")
	ErrOverestimate = errors.New("the result is overestimated")
)

type ResourceAnalyser interface {
	// Mark the start of analysing.
	Start() error

	// Stop analysing.
	Stop() error

	// Take sample.
	Analyse(*ResourceEvent) error

	// Events are returned on this channel.
	Query(float64) (float64, error)
}
