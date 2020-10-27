package breaker

import (
	"time"

	"github.com/pkg/errors"
)

const defaultMaxFailures int = 10
const defaultOpenStateDuration time.Duration = time.Second * 10

// Options Circuit breaker settings.
type Options struct {
	// MaxFailure amount to open the circuit from open state. 10 failures by default
	MaxFailures int
	// OpenStateDuration time to move from open to half open state
	OpenStateDuration time.Duration
}

// Breaker Circuit braker pattern implementation
type Breaker struct {
	// State current circuit braker state. It implements State iterface
	State             State
	storageService    Storage
	openStateDuration time.Duration
	maxFailures       int
}

// New implements Breaker factory
func New(storageService Storage, options *Options) (*Breaker, error) {
	maxFailures := defaultMaxFailures
	openStateDuration := defaultOpenStateDuration

	if options != nil {
		if options.MaxFailures > 0 {
			maxFailures = options.MaxFailures
		}

		if options.OpenStateDuration > time.Second*0 {
			openStateDuration = options.OpenStateDuration
		}
	}

	currentState, err := storageService.GetCurrentState()

	return &Breaker{
		State:             currentState,
		storageService:    storageService,
		maxFailures:       maxFailures,
		openStateDuration: openStateDuration,
	}, errors.Wrap(err, "NewBreaker -> Closed state by default")
}

// Ready checks if circuit if closed, else returns a OpenCircuitError error
func (b *Breaker) Ready() error {
	nextState, _ := b.State.Next(b.storageService, b.maxFailures)
	b.State = nextState
	err := b.State.OnEntry(b.storageService, b.openStateDuration)

	if !b.State.Ready() {
		return OpenCircuitError
	}

	return errors.Wrap(err, "Ready -> Closed state by default")
}

// Success method to be called when controlled logic by circuit breaker works propertly.
func (b *Breaker) Success() error {
	err := b.State.OnSuccess(b.storageService)

	return errors.Wrap(err, "Success")
}

// Fail method to be called when controlled logic by circuit breaker fails.
func (b *Breaker) Fail() error {
	err := b.State.OnFail(b.storageService)

	return errors.Wrap(err, "Fail")
}
