package reparos

import "runtime"

// Options holds configuration for the predictor.
type Options struct {
	BeamSize      int
	NumHypotheses int
	Device        string
	ComputeType   string
	NativeLibPath string
	// IntraThreads is the number of threads used by one translation (default 1;
	// 0 lets the engine use every core, which hurts tail latency on short queries).
	IntraThreads int
	// InterThreads is the number of translations that may run in parallel
	// (default min(NumCPU, 4)).
	InterThreads int
}

// DefaultOptions returns the default production options.
func DefaultOptions() Options {
	return Options{
		BeamSize:      10,
		NumHypotheses: 1,
		Device:        "cpu",
		ComputeType:   "int8",
		// One thread per translation: multi-threading a short query makes tail
		// latency far worse. Throughput comes from running translations in parallel.
		IntraThreads: 1,
		InterThreads: min(runtime.NumCPU(), 4),
	}
}

// Option configures the Predictor.
type Option func(*Options)

// WithBeamSize sets the beam search size (default: 10).
func WithBeamSize(size int) Option {
	return func(o *Options) {
		if size > 0 {
			o.BeamSize = size
		}
	}
}

// WithNumHypotheses sets the number of top hypotheses to return (default: 1).
func WithNumHypotheses(n int) Option {
	return func(o *Options) {
		if n > 0 {
			o.NumHypotheses = n
		}
	}
}

// WithDevice sets the inference device ("cpu" or "cuda").
func WithDevice(device string) Option {
	return func(o *Options) {
		o.Device = device
	}
}

// WithComputeType sets the quantization / compute type ("int8", "float16", "float32").
func WithComputeType(computeType string) Option {
	return func(o *Options) {
		o.ComputeType = computeType
	}
}

// WithNativeLib specifies the path to the compiled C++ bridge library (.dll or .so).
func WithNativeLib(path string) Option {
	return func(o *Options) {
		o.NativeLibPath = path
	}
}

// WithThreads sets threads per translation (intra) and parallel translations (inter).
func WithThreads(intra, inter int) Option {
	return func(o *Options) {
		o.IntraThreads = intra
		o.InterThreads = inter
	}
}
