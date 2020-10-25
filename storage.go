package breaker

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/benbjohnson/clock"
	"github.com/go-redis/redis"
	"github.com/pkg/errors"
	"github.com/rs/xid"
)

const (
	failureKey     string = "FAILURES"
	stateKey       string = "STATE"
	defaultFailure int    = 0
)

// Storage is the interface for circuit breaker state storage.
type Storage interface {
	GetCurrentState() (State, error)
	SetCurrentState(state State) error
	IncrementFailures() error
	GetFailures() (int, error)
	Clear() error
}

// RedisStorage to save circuit breaker current status using redis
type RedisStorage struct {
	key    xid.ID
	client redis.Cmdable
}

// NewRedisStorage returns a RedisStorage object
func NewRedisStorage(client redis.Cmdable, key *xid.ID) *RedisStorage {
	cbKey := xid.New()
	if key != nil {
		cbKey = *key
	}

	rs := RedisStorage{
		key:    cbKey,
		client: client,
	}

	return &rs
}

// GetCurrentState returns current circuit breaker state
func (rs *RedisStorage) GetCurrentState() (State, error) {
	key := rs.getStateKey()
	value, err := rs.client.Get(key).Result()
	if err == redis.Nil {
		return NewClosed(), nil
	}

	if err != nil {
		return NewClosed(), errors.Wrap(err, "RedisStorage -> GetCurrentState")
	}

	if value == stateOpen {
		ticker := clock.New()

		return NewOpen(ticker), nil
	}

	if value == stateHalfOpen {
		return NewHalfOpen(), nil
	}

	return NewClosed(), nil
}

// SetCurrentState persists the state
func (rs *RedisStorage) SetCurrentState(state State) error {
	key := rs.getStateKey()
	err := rs.client.Set(key, fmt.Sprint(state), 0).Err()

	if err != nil {
		return errors.Wrap(err, "RedisStorage -> SetCurrentState")
	}

	return nil
}

// IncrementFailures increments failures count
func (rs *RedisStorage) IncrementFailures() error {
	key := rs.getFailuresKey()
	err := rs.client.Incr(key).Err()

	if err != nil {
		return errors.Wrap(err, "RedisStorage -> IncrementFailures")
	}

	return nil
}

// GetFailures gets failures count
func (rs *RedisStorage) GetFailures() (int, error) {
	key := rs.getFailuresKey()
	value, err := rs.client.Get(key).Result()
	switch err {
	case nil:
		{
			failures, err := strconv.Atoi(value)
			if err != nil {
				return defaultFailure, errors.Wrap(err, "RedisStorage -> GetFailures -> Conversion")
			}

			return failures, nil
		}
	case redis.Nil:
		return defaultFailure, nil
	default:
		return defaultFailure, errors.Wrap(err, "RedisStorage -> GetFailures")
	}
}

// Clear sets failures counts to zero
func (rs *RedisStorage) Clear() error {
	key := rs.getFailuresKey()
	err := rs.client.Set(key, defaultFailure, 0).Err()

	if err != nil {
		return errors.Wrap(err, "RedisStorage -> Clear")
	}

	return nil
}

func (rs *RedisStorage) getFailuresKey() string {
	return fmt.Sprintf("%s_%s", rs.key.String(), failureKey)
}

func (rs *RedisStorage) getStateKey() string {
	return fmt.Sprintf("%s_%s", rs.key.String(), stateKey)
}

// MemoryStorage to save circuit breaker current status into memory.
// Avoid using it in multi container services
type MemoryStorage struct {
	mu       sync.RWMutex
	state    State
	failures int
}

// NewMemoryStorage returns a MemoryStorage object
func NewMemoryStorage() *MemoryStorage {
	ms := MemoryStorage{}

	return &ms
}

// GetCurrentState returns current circuit breaker state
func (ms *MemoryStorage) GetCurrentState() (State, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if ms.state == nil {
		return NewClosed(), nil
	}

	return ms.state, nil
}

// SetCurrentState persists the state
func (ms *MemoryStorage) SetCurrentState(state State) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.state = state

	return nil
}

// IncrementFailures increments failures count
func (ms *MemoryStorage) IncrementFailures() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.failures++

	return nil
}

// GetFailures gets failures count
func (ms *MemoryStorage) GetFailures() (int, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	return ms.failures, nil
}

// Clear sets failures counts to zero
func (ms *MemoryStorage) Clear() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.failures = 0

	return nil
}
