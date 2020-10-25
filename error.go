package breaker

type circuitError string

func (e circuitError) Error() string {
	return string(e)
}

// OpenCircuitError raises when circuit is open
const OpenCircuitError = circuitError("breaker: open circuit")
