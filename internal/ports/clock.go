package ports

import "time"

// Clock abstracts time measurement for deterministic testing and simulations.
type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
}

// RealClock uses the real host system time.
type RealClock struct{}

func NewRealClock() *RealClock {
	return &RealClock{}
}

func (r *RealClock) Now() time.Time {
	return time.Now()
}

func (r *RealClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}
