package breaker

import (
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/pkg/errors"
)

const (
	stateClosed   string = "closed"
	stateOpen     string = "open"
	stateHalfOpen string = "half-open"
)

// State is the interface for circuit breaker state. Immplementation of this interface ensure a valid state
type State interface {
	Ready() bool
	Next(sr Storage, maxFailures int) (State, error)
	OnEntry(sr Storage, stateDuration time.Duration) error
	OnSuccess(sr Storage) error
	OnFail(sr Storage) error
	String() string
}

// Closed state
type Closed struct{}

// NewClosed returns a closed circuit breaker state
func NewClosed() *Closed {
	return &Closed{}
}

// Ready during close state is always true. Managed logic can be executed
func (sc *Closed) Ready() bool { return true }

// Next return next circuit breaker state checking failures.
// When failures is bigger than maxFaiules, circuit breaker goes to Open state
func (sc *Closed) Next(sr Storage, maxFailures int) (State, error) {
	failures, err := sr.GetFailures()
	if err != nil {
		return sc, errors.Wrap(err, "stateClosed -> Next -> GetFailures")
	}

	if failures < maxFailures {
		return sc, nil
	}

	return NewOpen(clock.New()), nil
}

// OnEntry clears failures using storage service
func (sc *Closed) OnEntry(sr Storage, _ time.Duration) error {
	err := sr.SetCurrentState(sc)
	if err != nil {
		return errors.Wrap(err, "stateClosed -> OnEntry -> SetCurrentState")
	}
	err = sr.Clear()
	if err != nil {
		return errors.Wrap(err, "stateClosed -> OnEntry -> Clear")
	}

	return nil
}

// OnSuccess clears failures using storage service when controlled logic by circuit breaker works propertly.
func (sc *Closed) OnSuccess(sr Storage) error {
	failures, _ := sr.GetFailures()
	if failures == 0 {
		return nil
	}

	err := sr.Clear()
	if err != nil {
		return errors.Wrap(err, "stateClosed -> OnSuccess -> Clear")
	}

	return nil
}

// OnFail increments failures count using storage service when controlled logic by circuit breaker fails.
func (sc *Closed) OnFail(sr Storage) error {
	err := sr.IncrementFailures()

	if err != nil {
		return errors.Wrap(err, "stateClosed -> OnFail -> IncrementFailures")
	}

	return nil
}

func (sc *Closed) String() string {
	return stateClosed
}

// Open state
type Open struct {
	expiredTime bool
	mu          sync.RWMutex
	clock       clock.Clock
}

// NewOpen returns an open circuit breaker state
func NewOpen(clock clock.Clock) *Open {
	return &Open{
		clock: clock,
	}
}

// Ready during open state is always false. Managed logic can not be executed
func (so *Open) Ready() bool { return false }

// Next return next circuit breaker state checking time in open state.
// When time in open state is bigger than max expiration time, circuit breaker goes to half open state
func (so *Open) Next(_ Storage, _ int) (State, error) {
	so.mu.RLock()
	defer so.mu.RUnlock()
	if so.expiredTime {
		return NewHalfOpen(), nil
	}

	return so, nil
}

// OnEntry starts time to check open state expiration time. Failures are clear too
func (so *Open) OnEntry(sr Storage, stateDuration time.Duration) error {
	_ = so.clock.AfterFunc(stateDuration, func() {
		so.mu.Lock()
		defer so.mu.Unlock()
		so.expiredTime = true
	})

	err := sr.SetCurrentState(so)
	if err != nil {
		return errors.Wrap(err, "stateOpen -> OnEntry -> SetCurrentState")
	}

	err = sr.Clear()
	if err != nil {
		return errors.Wrap(err, "stateOpen -> OnEntry -> Clear")
	}

	return nil
}

// OnSuccess to implement State interface.
func (so *Open) OnSuccess(_ Storage) error { return nil }

// OnFail to implement State interface.
func (so *Open) OnFail(_ Storage) error { return nil }

func (so *Open) String() string {
	return stateOpen
}

// HalfOpen state
type HalfOpen struct{}

// NewHalfOpen returns an half-open circuit breaker state
func NewHalfOpen() *HalfOpen {
	return &HalfOpen{}
}

// Ready during half open state is always true. Managed logic can be executed to check its behaviour
func (sho *HalfOpen) Ready() bool { return true }

// Next returns next circuit breaker state checking failures.
// If failures, circuit breaker goes to open state, else to closed state
func (sho *HalfOpen) Next(sr Storage, _ int) (State, error) {
	closed := NewClosed()
	failures, err := sr.GetFailures()
	if err != nil {
		return closed, errors.Wrap(err, "stateHalfOpen -> Next -> GetFailures")
	}

	if failures > 0 {
		return NewOpen(clock.New()), nil
	}

	return closed, nil
}

// OnEntry clears failures using storage service
func (sho *HalfOpen) OnEntry(sr Storage, _ time.Duration) error {
	err := sr.SetCurrentState(sho)
	if err != nil {
		return errors.Wrap(err, "stateHalfOpen -> OnEntry -> SetCurrentState")
	}
	err = sr.Clear()
	if err != nil {
		return errors.Wrap(err, "stateHalfOpen -> OnEntry -> Clear")
	}

	return nil
}

// OnSuccess to implement State interface.
func (sho *HalfOpen) OnSuccess(_ Storage) error { return nil }

// OnFail increments failures count using storage service when controlled logic by circuit breaker fails.
func (sho *HalfOpen) OnFail(sr Storage) error {
	err := sr.IncrementFailures()

	if err != nil {
		return errors.Wrap(err, "stateHalfOpen -> OnFail -> IncrementFailures")
	}

	return nil
}

func (sho *HalfOpen) String() string {
	return stateHalfOpen
}
