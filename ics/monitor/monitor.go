package monitor

import "time"

// ContainerEvent represents a
type ResourceEvent struct {
	// Name of Event.
	Name string

	// Data source.
	Source string

	// Time event is triggered.
	Time time.Time
}


type ResourceMonitor interface {
	// Starts Watching.
	Start() error

	// Stops watching.
	Stop() error

	// Errors are returned on this channel.
	Error() chan error

	AddAnalyser(string, ResourceAnalyser) (bool, error)

	RemoveAnalyser(string) error

	GetAnalyser(string) ResourceAnalyser
}
