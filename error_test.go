package breaker_test

import (
	"breaker"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCircuitError_Error(t *testing.T) {
	assert.Equal(t, "breaker: open circuit", breaker.OpenCircuitError.Error())
}
