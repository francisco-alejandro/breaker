# breaker

[![GoDoc](https://godoc.org/github.com/mercari/go-circuitbreaker?status.svg)](https://godoc.org/github.com/francisco-alejandro/breaker)

**breaker** is a Circuit Breaker pattern implementation in Go.

- Provides natural code flow.
- Protects logic without wrapping it and using type-unsafe interface{}
- Redis storage backend

# What is circuit breaker?

See: [Circuit Breaker pattern](https://martinfowler.com/bliki/CircuitBreaker.html)


## Usage
The struct Breaker is a state machine to prevent sending requests that are likely to fail. The function breaker.New creates a new Breaker.
```go
    func New(storageService Storage, options *Options) (*Breaker, error)
```

You can use Redis Storage using breaker.NewRedisStorage function
```go
    func NewRedisStorage(client redis.Cmdable, key *xid.ID) *RedisStorage
```

Optional [xid.ID](https://github.com/rs/xid) key to save state and failures count into Redis. If key is not provided, one is generated.


You can configure Breaker by the optional struct Options:

```go
    type Options struct {
        MaxFailures int
        OpenStateDuration time.Duration
    }
```

- `MaxFailures` is the maximum number of failed requests allowed to pass through. 10 by default

- `OpenStateDuration` is the period of the open state, after which the state of `CircuitBreaker` becomes half-open. By default it is set to 10 seconds.


## Example
```go
    var cb *breaker.Breaker

    func Get(url string) ([]byte, error) {
        err := cb.Ready()
        if err == breaker.OpenCircuitError {
            return nil, err
        }

        resp, err := http.Get(url)
        if err != nil {
            // Open circuit only with desired errors
            _ = cb.Fail()

            return nil, err
        }
        // Update counters and state after success in wrapped logic
        _ = cb.Success()
        defer resp.Body.Close()

        body, err := ioutil.ReadAll(resp.Body)
        if err != nil {
            return nil, err
        }

        return body, nil
    }
```

See [example](https://github.com/francisco-alejandro/breaker/blob/main/example) for details.

## Installation

```bash
    go get github.com/francisco-alejandro/breaker
```