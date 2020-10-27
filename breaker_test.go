package breaker_test

import (
	"testing"
	"time"

	"github.com/francisco-alejandro/breaker"
	"github.com/stretchr/testify/assert"
)

func TestBreaker_Ready(t *testing.T) {
	storageService := breaker.NewMemoryStorage()

	options := breaker.Options{
		MaxFailures:       1,
		OpenStateDuration: time.Second * 1,
	}

	b, err := breaker.New(storageService, &options)
	assert.NoError(t, err)

	err = b.Ready()
	assert.NoError(t, err)

	err = storageService.IncrementFailures()
	assert.NoError(t, err)

	err = b.Ready()
	assert.Error(t, err, "breaker: open circuit")
}

func TestBreaker_Success(t *testing.T) {
	storageService := breaker.NewMemoryStorage()
	err := storageService.SetCurrentState(breaker.NewHalfOpen())
	assert.NoError(t, err)

	b, err := breaker.New(storageService, nil)
	assert.NoError(t, err)

	err = b.Success()
	assert.NoError(t, err)

	err = b.Ready()
	assert.NoError(t, err)

	_, ok := b.State.(*breaker.Closed)
	assert.True(t, ok)
}

func TestBreaker_Fail(t *testing.T) {
	storageService := breaker.NewMemoryStorage()
	err := storageService.SetCurrentState(breaker.NewHalfOpen())
	assert.NoError(t, err)

	b, err := breaker.New(storageService, nil)
	assert.NoError(t, err)

	err = b.Fail()
	assert.NoError(t, err)

	err = b.Ready()
	assert.Error(t, err, "breaker: open circuit")

	err = storageService.SetCurrentState(breaker.NewClosed())
	assert.NoError(t, err)

	options := breaker.Options{
		MaxFailures: 1,
	}

	b, err = breaker.New(storageService, &options)
	assert.NoError(t, err)

	err = b.Fail()
	assert.NoError(t, err)

	err = b.Ready()
	assert.Error(t, err, "breaker: open circuit")
}
