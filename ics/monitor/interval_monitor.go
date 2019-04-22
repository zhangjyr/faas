package monitor

import (
	"log"
	"sync"
	"time"
)

var DefaultInterval = 1 * time.Second

// Watcher for container-related inotify events in the cgroup hierarchy.
//
// Implementation is thread-safe.
type IntervalMonitor struct {
	// Map of resources being monitored.
	monitored  map[string]ResourceAnalyser
	lock       sync.Mutex        // Lock for all datastructure access.
	err        chan error        // Errors are sent on this channel.
	interval   time.Duration     // Interval.
	stop       chan bool         // Tell monitor to stop.
	lastTick   time.Time         // Last tick monitor triggered.
}

func NewIntervalMonitor(d *time.Duration) *IntervalMonitor {
	if d == nil {
		d = &DefaultInterval
	}
	return &IntervalMonitor{
		monitored:   make(map[string]ResourceAnalyser),
		err:         make(chan error),
		stop:        make(chan bool),
		interval:    *d,
	}
}

// Add a analyser for specified resource. Returns if the analyser was already being monitored.
func (im *IntervalMonitor) AddAnalyser(name string, analyser ResourceAnalyser) (bool, error) {
	im.lock.Lock()
	defer im.lock.Unlock()

	_, alreadyWatched := im.monitored[name]

	// Record our watching of the container.
	if !alreadyWatched {
		im.monitored[name] = analyser
	}
	return alreadyWatched, nil
}

// Remove monitor for specified resource.
func (im *IntervalMonitor) RemoveAnalyser(name string) error {
	im.lock.Lock()
	defer im.lock.Unlock()

	// If we don't have a watch registered for this, just return.
	_, ok := im.monitored[name]
	if !ok {
		return nil
	}

	delete(im.monitored, name)
	return nil
}

func (im *IntervalMonitor) GetAnalyser(name string) ResourceAnalyser {
	im.lock.Lock()
	defer im.lock.Unlock()

	return im.monitored[name]
}

func (im *IntervalMonitor) Start() error {
	go im.monitor()
	return nil
}

// Errors are returned on this channel.
func (im *IntervalMonitor) Error() chan error {
	return im.err
}

// Closes the inotify watcher.
func (im *IntervalMonitor) Stop() error {
	for _, analyser := range im.monitored {
		analyser.Stop()
	}
	im.stop <- true
	return nil
}

func (im *IntervalMonitor) monitor() {
	// Allow analyser to initialize.
	for _, analyser := range im.monitored {
		go analyser.Start()
	}

	// Set timeout
	timeout := 100 * time.Millisecond
	if im.interval / 10 < timeout {
		timeout = im.interval / 10
	}

	timer := time.NewTimer(0 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-im.stop:
			// Stop monitoring when signaled.
			return
		case <-timer.C:
		}

		// Start monitoring parellel
		start := time.Now()
		for name, analyser := range im.monitored {
			go func() {
				tick := time.Now()
				err := analyser.Analyse(&ResourceEvent{
					Name: name,
					Time: tick,
				})
				if err != nil {
					im.err <- err
				}
				duration := time.Since(tick)
				if duration >= timeout {
					log.Printf("[%s] monitoring took %s\n", name, duration)
				}
			}()
			time.Sleep(timeout)
		}
		im.lastTick = start

		// Drain the timer to be accurate and safe to reset.
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		duration := time.Since(start)
		if (duration >= im.interval) {
			log.Println("Monitoring took too long")
		}
		timer.Reset(im.nextInterval(duration))
	}
}

// Allow dynamic interval for later evaluation
func (im *IntervalMonitor) nextInterval(skip time.Duration) time.Duration {
	if skip > im.interval {
		return im.interval
	}

	return im.interval - skip
}
