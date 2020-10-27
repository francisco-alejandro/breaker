package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/francisco-alejandro/breaker"
)

var cb *breaker.Breaker

func init() {
	var err error

	s := breaker.NewMemoryStorage()

	options := breaker.Options{
		MaxFailures:       1,
		OpenStateDuration: time.Second * 1,
	}

	cb, err = breaker.New(s, &options)
	if err != nil {
		log.Fatal(err)
	}
}

// Get wraps http.Get in CircuitBreaker.
func Get(url string) ([]byte, error) {
	err := cb.Ready()

	switch err {
	case breaker.OpenCircuitError:
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

func main() {
	body, err := Get("http://www.google.com/robots.txt")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(body))
}
