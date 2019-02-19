package monitor

import (
	"errors"
)

var (
	ErrNotEnoughData = errors.New("not enough data")
	ErrUndeterminate = errors.New("the result is undeterminated")
)

type ResourceAnalyser interface {
	// Mark the start of analysing.
	Start() error

	// Stop analysing.
	Stop() error

	// Take sample.
	Sample(*ResourceEvent) error

	// Events are returned on this channel.
	Query(float64) (float64, error)
}
