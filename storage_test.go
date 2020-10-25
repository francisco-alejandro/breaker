package breaker_test

import (
	"breaker"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/benbjohnson/clock"
	"github.com/elliotchance/redismock"
	"github.com/go-redis/redis"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
)

const stateClosed string = "closed"

func TestMemoryStorage_GetCurrentState(t *testing.T) {
	ms := breaker.NewMemoryStorage()

	currentState, err := ms.GetCurrentState()
	assert.NoError(t, err)

	_, ok := currentState.(*breaker.Closed)
	assert.True(t, ok)
}

func TestMemoryStorage_SetCurrentState(t *testing.T) {
	ms := breaker.NewMemoryStorage()

	err := ms.SetCurrentState(breaker.NewHalfOpen())
	assert.NoError(t, err)

	currentState, err := ms.GetCurrentState()
	assert.NoError(t, err)

	_, ok := currentState.(*breaker.HalfOpen)
	assert.True(t, ok)
}

func TestMemoryStorage_IncrementFailures(t *testing.T) {
	ms := breaker.NewMemoryStorage()

	err := ms.IncrementFailures()
	assert.NoError(t, err)

	failures, err := ms.GetFailures()
	assert.NoError(t, err)
	assert.Equal(t, failures, 1)
}

func TestMemoryStorage_Clear(t *testing.T) {
	ms := breaker.NewMemoryStorage()

	err := ms.IncrementFailures()
	assert.NoError(t, err)

	failures, err := ms.GetFailures()
	assert.NoError(t, err)
	assert.Equal(t, 1, failures)

	err = ms.Clear()
	assert.NoError(t, err)

	failures, err = ms.GetFailures()
	assert.NoError(t, err)
	assert.Equal(t, 0, failures)
}

func newTestRedis() *redismock.ClientMock {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return redismock.NewNiceMock(client)
}

func TestRedisStorage_GetCurrentState(t *testing.T) {
	client := newTestRedis()
	key := xid.New()
	stateKey := fmt.Sprintf("%s_%s", key.String(), "STATE")

	rs := breaker.NewRedisStorage(client, nil)

	currentState, err := rs.GetCurrentState()
	assert.NoError(t, err)

	_, ok := currentState.(*breaker.Closed)
	assert.True(t, ok)

	client.On("Get", stateKey).
		Return(redis.NewStringResult("", errors.New("server not available")))

	rs = breaker.NewRedisStorage(client, &key)

	currentState, err = rs.GetCurrentState()
	assert.Error(t, err, "RedisStorage -> GetCurrentState")

	_, ok = currentState.(*breaker.Closed)
	assert.True(t, ok)
}

func TestRedisStorage_SetCurrentState(t *testing.T) {
	client := newTestRedis()
	key := xid.New()
	stateKey := fmt.Sprintf("%s_%s", key.String(), "STATE")

	rs := breaker.NewRedisStorage(client, nil)

	err := rs.SetCurrentState(breaker.NewClosed())
	assert.NoError(t, err)

	currentState, err := rs.GetCurrentState()
	assert.NoError(t, err)
	_, ok := currentState.(*breaker.Closed)
	assert.True(t, ok)

	err = rs.SetCurrentState(breaker.NewHalfOpen())
	assert.NoError(t, err)

	currentState, err = rs.GetCurrentState()
	assert.NoError(t, err)
	_, ok = currentState.(*breaker.HalfOpen)
	assert.True(t, ok)

	ticker := clock.New()
	err = rs.SetCurrentState(breaker.NewOpen(ticker))
	assert.NoError(t, err)

	currentState, err = rs.GetCurrentState()
	assert.NoError(t, err)

	_, ok = currentState.(*breaker.Open)
	assert.True(t, ok)

	client.On("Set", stateKey, stateClosed, time.Duration(0)).
		Return(redis.NewStatusResult("", errors.New("server not available")))

	rs = breaker.NewRedisStorage(client, &key)
	err = rs.SetCurrentState(breaker.NewClosed())
	assert.Error(t, err, "RedisStorage -> SetCurrentState")
}

func TestRedisStorage_IncrementFailures(t *testing.T) {
	key := xid.New()
	failuresKey := fmt.Sprintf("%s_%s", key.String(), "FAILURES")

	client := newTestRedis()

	rs := breaker.NewRedisStorage(client, &key)

	err := rs.IncrementFailures()
	assert.NoError(t, err)

	failures, err := rs.GetFailures()
	assert.NoError(t, err)
	assert.Equal(t, 1, failures)

	client.On("Incr", failuresKey).
		Return(redis.NewIntResult(0, errors.New("server not available")))

	rs = breaker.NewRedisStorage(client, &key)

	err = rs.IncrementFailures()
	assert.Error(t, err, "RedisStorage -> IncrementFailures")
}

func TestRedisStorage_GetFailures(t *testing.T) {
	key := xid.New()
	failuresKey := fmt.Sprintf("%s_%s", key.String(), "FAILURES")

	client := newTestRedis()

	rs := breaker.NewRedisStorage(client, &key)

	failures, err := rs.GetFailures()
	assert.NoError(t, err)
	assert.Equal(t, 0, failures)

	client.On("Get", failuresKey).
		Return(redis.NewStringResult("", errors.New("server not available")))

	rs = breaker.NewRedisStorage(client, &key)

	failures, err = rs.GetFailures()
	assert.Error(t, err, "RedisStorage -> GetFailures")
	assert.Equal(t, 0, failures)

	client = newTestRedis()
	client.On("Get", failuresKey).
		Return(redis.NewStringResult("INVALID INTEGER VALUE", nil))

	rs = breaker.NewRedisStorage(client, &key)

	err = rs.IncrementFailures()
	assert.NoError(t, err)

	failures, err = rs.GetFailures()
	assert.Error(t, err, "RedisStorage -> GetFailures -> Conversion")
	assert.Equal(t, 0, failures)
}

func TestRedisStorage_Clear(t *testing.T) {
	key := xid.New()
	client := newTestRedis()
	failuresKey := fmt.Sprintf("%s_%s", key.String(), "FAILURES")
	stateKey := fmt.Sprintf("%s_%s", key.String(), "STATE")

	rs := breaker.NewRedisStorage(client, &key)

	err := rs.IncrementFailures()
	assert.NoError(t, err)

	err = rs.Clear()
	assert.NoError(t, err)

	failures, err := rs.GetFailures()
	assert.NoError(t, err)
	assert.Equal(t, 0, failures)

	client.On("Set", stateKey, string(stateClosed), time.Duration(0)).
		Return(redis.NewStatusResult("", nil))

	client.On("Set", failuresKey, 0, time.Duration(0)).
		Return(redis.NewStatusResult("", errors.New("server not available")))

	rs = breaker.NewRedisStorage(client, &key)

	err = rs.Clear()
	assert.Error(t, err, "RedisStorage -> SetCurrentState")
}
