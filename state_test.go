package breaker_test

import (
	"breaker"
	"github.com/benbjohnson/clock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type storageMock struct {
	failSetCurrentState bool
	failGetFailures     bool
	failureCount        int
}

type storageMockOptions struct {
	failSetCurrentState bool
	failGetFailures     bool
	failureCount        int
}

func newStorageMock(options storageMockOptions) *storageMock {
	return &storageMock{
		failSetCurrentState: options.failSetCurrentState,
		failGetFailures:     options.failGetFailures,
		failureCount:        options.failureCount,
	}
}

func (sm *storageMock) GetCurrentState() (breaker.State, error) {
	return breaker.NewClosed(), errors.New("server not available")
}

func (sm *storageMock) SetCurrentState(_ breaker.State) error {
	if sm.failSetCurrentState {
		return errors.New("server not available")
	}

	return nil
}

func (sm *storageMock) IncrementFailures() error {
	return errors.New("server not available")
}

func (sm *storageMock) GetFailures() (int, error) {
	if sm.failGetFailures {
		return 0, errors.New("server not available")
	}
	return sm.failureCount, nil
}

func (sm *storageMock) Clear() error {
	return errors.New("server not available")
}

func TestClosed_Ready(t *testing.T) {
	var closed breaker.Closed

	ready := closed.Ready()
	assert.True(t, ready)
}

func TestClosed_Next(t *testing.T) {
	closed := breaker.NewClosed()
	storage := breaker.NewMemoryStorage()
	storageMock := newStorageMock(storageMockOptions{
		failSetCurrentState: true,
		failGetFailures:     true,
	})
	maxFailures := 1

	state, err := closed.Next(storage, maxFailures)
	assert.NoError(t, err)
	_, ok := state.(*breaker.Closed)
	assert.True(t, ok)

	err = storage.IncrementFailures()
	assert.NoError(t, err)

	state, err = closed.Next(storage, maxFailures)
	assert.NoError(t, err)

	_, ok = state.(*breaker.Open)
	assert.True(t, ok)

	state, err = closed.Next(storageMock, maxFailures)
	assert.Error(t, err, "stateClosed -> Next -> GetFailures")
	_, ok = state.(*breaker.Closed)
	assert.True(t, ok)

}

func TestClosed_OnEntry(t *testing.T) {
	closed := breaker.NewClosed()
	storage := breaker.NewMemoryStorage()

	err := storage.IncrementFailures()
	assert.NoError(t, err)

	err = closed.OnEntry(storage, time.Second)
	assert.NoError(t, err)

	failures, err := storage.GetFailures()
	assert.NoError(t, err)
	assert.Equal(t, 0, failures)

	state, err := storage.GetCurrentState()
	assert.NoError(t, err)

	_, ok := state.(*breaker.Closed)
	assert.True(t, ok)

	storageMock := newStorageMock(storageMockOptions{
		failSetCurrentState: true,
	})
	err = closed.OnEntry(storageMock, time.Second)
	assert.Error(t, err, "stateClosed -> OnEntry -> SetCurrentState")

	storageMock = newStorageMock(storageMockOptions{
		failGetFailures: true,
	})
	err = closed.OnEntry(storageMock, time.Second)
	assert.Error(t, err, "stateClosed -> OnEntry -> Clear")
}

func TestClosed_OnSuccess(t *testing.T) {
	closed := breaker.NewClosed()
	storage := breaker.NewMemoryStorage()
	storageMock := newStorageMock(storageMockOptions{
		failSetCurrentState: true,
		failureCount:        1,
	})

	err := closed.OnSuccess(storage)
	assert.NoError(t, err)

	failures, err := storage.GetFailures()
	assert.NoError(t, err)
	assert.Equal(t, 0, failures)

	err = storage.IncrementFailures()
	assert.NoError(t, err)

	err = closed.OnSuccess(storageMock)
	assert.Error(t, err, "stateClosed -> OnSuccess -> Clear")

	err = closed.OnSuccess(storage)
	assert.NoError(t, err)

	failures, err = storage.GetFailures()
	assert.NoError(t, err)
	assert.Equal(t, 0, failures)
}

func TestClosed_OnFail(t *testing.T) {
	closed := breaker.NewClosed()
	storage := breaker.NewMemoryStorage()

	err := closed.OnFail(storage)
	assert.NoError(t, err)

	failures, err := storage.GetFailures()
	assert.NoError(t, err)
	assert.Equal(t, 1, failures)

	storageMock := newStorageMock(storageMockOptions{})
	err = closed.OnFail(storageMock)
	assert.Error(t, err, "stateClosed -> OnFail -> IncrementFailures")
}

func TestOpen_Ready(t *testing.T) {
	clockMock := clock.NewMock()
	open := breaker.NewOpen(clockMock)

	ready := open.Ready()
	assert.False(t, ready)
}

func TestOpen_Next(t *testing.T) {
	clockMock := clock.NewMock()
	storage := breaker.NewMemoryStorage()
	maxFailures := 1

	open := breaker.NewOpen(clockMock)

	state, err := open.Next(storage, maxFailures)
	assert.NoError(t, err)
	_, ok := state.(*breaker.Open)
	assert.True(t, ok)

	err = open.OnEntry(storage, time.Second)
	assert.NoError(t, err)

	clockMock.Add(time.Second)

	state, err = open.Next(storage, maxFailures)
	assert.NoError(t, err)
	_, ok = state.(*breaker.HalfOpen)
	assert.True(t, ok)
}

func TestOpen_OnEntry(t *testing.T) {
	clockMock := clock.NewMock()
	storage := breaker.NewMemoryStorage()

	open := breaker.NewOpen(clockMock)

	err := storage.IncrementFailures()
	assert.NoError(t, err)

	err = open.OnEntry(storage, time.Second)
	assert.NoError(t, err)

	failures, err := storage.GetFailures()
	assert.NoError(t, err)
	assert.Equal(t, 0, failures)

	state, err := storage.GetCurrentState()
	assert.NoError(t, err)

	_, ok := state.(*breaker.Open)
	assert.True(t, ok)

	storageMock := newStorageMock(storageMockOptions{
		failSetCurrentState: true,
	})
	err = open.OnEntry(storageMock, time.Second)
	assert.Error(t, err, "stateOpen -> OnEntry -> SetCurrentState")

	storageMock = newStorageMock(storageMockOptions{})
	err = open.OnEntry(storageMock, time.Second)
	assert.Error(t, err, "stateOpen -> OnEntry -> Clear")
}

func TestOpen_OnFail(t *testing.T) {
	storage := breaker.NewMemoryStorage()
	clockMock := clock.NewMock()

	open := breaker.NewOpen(clockMock)

	err := open.OnFail(storage)
	assert.NoError(t, err)
}

func TestOpen_OnSuccess(t *testing.T) {
	storage := breaker.NewMemoryStorage()
	clockMock := clock.NewMock()

	open := breaker.NewOpen(clockMock)

	err := open.OnSuccess(storage)
	assert.NoError(t, err)
}

func TestHalfOpen_Ready(t *testing.T) {
	halfOpen := breaker.NewHalfOpen()

	ready := halfOpen.Ready()
	assert.True(t, ready)
}

func TestHalfOpen_Next(t *testing.T) {
	halfOpen := breaker.NewHalfOpen()
	storage := breaker.NewMemoryStorage()
	maxFailures := 0

	state, err := halfOpen.Next(storage, maxFailures)
	assert.NoError(t, err)
	_, ok := state.(*breaker.Closed)
	assert.True(t, ok)

	err = storage.IncrementFailures()
	assert.NoError(t, err)

	state, err = halfOpen.Next(storage, maxFailures)
	assert.NoError(t, err)
	_, ok = state.(*breaker.Open)
	assert.True(t, ok)

	storageMock := newStorageMock(storageMockOptions{
		failGetFailures: true,
	})
	state, err = halfOpen.Next(storageMock, maxFailures)
	assert.Error(t, err, "stateHalfOpen -> Next -> GetFailures")
	_, ok = state.(*breaker.Closed)
	assert.True(t, ok)
}

func TestHalfOpen_OnEntry(t *testing.T) {
	halfOpen := breaker.NewHalfOpen()
	storage := breaker.NewMemoryStorage()

	err := storage.IncrementFailures()
	assert.NoError(t, err)

	err = halfOpen.OnEntry(storage, time.Second)
	assert.NoError(t, err)

	failures, err := storage.GetFailures()
	assert.NoError(t, err)
	assert.Equal(t, 0, failures)

	state, err := storage.GetCurrentState()
	assert.NoError(t, err)

	_, ok := state.(*breaker.HalfOpen)
	assert.True(t, ok)

	storageMock := newStorageMock(storageMockOptions{
		failSetCurrentState: true,
	})
	err = halfOpen.OnEntry(storageMock, time.Second)
	assert.Error(t, err, "stateHalfOpen -> OnEntry -> SetCurrentState")

	storageMock = newStorageMock(storageMockOptions{})
	err = halfOpen.OnEntry(storageMock, time.Second)
	assert.Error(t, err, "stateHalfOpen -> OnEntry -> Clear")
}

func TestHalfOpen_OnSuccess(t *testing.T) {
	halfOpen := breaker.NewHalfOpen()
	storage := breaker.NewMemoryStorage()

	err := halfOpen.OnSuccess(storage)
	assert.NoError(t, err)
}

func TestHalfOpen_OnFail(t *testing.T) {
	halfOpen := breaker.NewHalfOpen()
	storage := breaker.NewMemoryStorage()

	err := halfOpen.OnFail(storage)
	assert.NoError(t, err)

	failures, err := storage.GetFailures()
	assert.NoError(t, err)
	assert.Equal(t, 1, failures)

	storageMock := newStorageMock(storageMockOptions{})
	err = halfOpen.OnFail(storageMock)
	assert.Error(t, err, "stateHalfOpen -> OnFail -> IncrementFailures")
}
