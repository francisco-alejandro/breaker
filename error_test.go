package breaker_test

import (
	"testing"

	"github.com/francisco-alejandro/breaker"
	"github.com/stretchr/testify/assert"
)

func TestCircuitError_Error(t *testing.T) {
	assert.Equal(t, "breaker: open circuit", breaker.OpenCircuitError.Error())
}
