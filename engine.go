package reparos

// engine is an inference backend. Implementations must be safe for
// concurrent use by multiple goroutines.
type engine interface {
	predict(query string, beamSize, numHypotheses int) (*Result, error)
	close() error
}
