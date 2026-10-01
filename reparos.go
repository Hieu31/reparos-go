package reparos

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

//go:embed models/v4_int8/* internal/runner/runner.py
var embeddedFS embed.FS

// Result contains the spell correction output and metadata.
type Result struct {
	InputQuery string    `json:"input_query"`
	Top1Query  string    `json:"top1_query"`
	Hypotheses []string  `json:"hypotheses"`
	Scores     []float64 `json:"scores"`
	Changed    bool      `json:"changed"`
	LatencyMs  float64   `json:"latency_ms"`
}

// Predictor runs spell correction inference on input queries.
type Predictor struct {
	opts     Options
	modelDir string
	eng      engine
}

var (
	defaultPredictor *Predictor
	defaultOnce      sync.Once
	defaultErr       error
)

// Correct is a zero-config, plug-and-play function that uses the embedded model.
// Anyone can call this directly without passing any file paths or configuration.
// Example:
//
//	res, err := reparos.Correct("d pasteur q3")
//	// res = "đường pasteur quận 3"
func Correct(query string) (string, error) {
	defaultOnce.Do(func() {
		defaultPredictor, defaultErr = New()
	})
	if defaultErr != nil {
		return query, defaultErr
	}

	result, err := defaultPredictor.Predict(query)
	if err != nil {
		return query, err
	}
	return result.Top1Query, nil
}

// PredictDetailed returns the full Result including top-1, hypotheses, and latency
// using the global default embedded predictor.
func PredictDetailed(query string) (*Result, error) {
	defaultOnce.Do(func() {
		defaultPredictor, defaultErr = New()
	})
	if defaultErr != nil {
		return nil, defaultErr
	}
	return defaultPredictor.Predict(query)
}

// New creates and initializes a Predictor instance.
// If modelDir is omitted or empty (""), it automatically extracts and uses
// the embedded model.bin and tokenizer.model (Zero-Config).
func New(args ...any) (*Predictor, error) {
	var modelDir string
	var options []Option

	for _, arg := range args {
		switch v := arg.(type) {
		case string:
			modelDir = v
		case Option:
			options = append(options, v)
		}
	}

	opts := DefaultOptions()
	for _, opt := range options {
		opt(&opts)
	}

	var runnerPath string
	if modelDir == "" {
		// Use embedded model
		extractedModelDir, extractedRunner, err := extractEmbeddedAssets()
		if err != nil {
			return nil, fmt.Errorf("failed to prepare embedded model: %w", err)
		}
		modelDir = extractedModelDir
		runnerPath = extractedRunner
	}

	absModelDir, err := filepath.Abs(modelDir)
	if err != nil {
		return nil, fmt.Errorf("invalid model path: %w", err)
	}

	// Verify required model files exist
	modelBin := filepath.Join(absModelDir, "model.bin")
	tokenizerModel := filepath.Join(absModelDir, "tokenizer.model")
	if _, err := os.Stat(modelBin); os.IsNotExist(err) {
		return nil, fmt.Errorf("model.bin not found in %s", absModelDir)
	}
	if _, err := os.Stat(tokenizerModel); os.IsNotExist(err) {
		return nil, fmt.Errorf("tokenizer.model not found in %s", absModelDir)
	}

	p := &Predictor{
		opts:     opts,
		modelDir: absModelDir,
	}

	if err := p.initEngine(runnerPath); err != nil {
		return nil, err
	}

	return p, nil
}

func extractEmbeddedAssets() (string, string, error) {
	cacheBase, err := os.UserCacheDir()
	if err != nil {
		cacheBase = os.TempDir()
	}

	targetDir := filepath.Join(cacheBase, "reparos-go", "models", "v4_int8")
	runnerDir := filepath.Join(cacheBase, "reparos-go", "runner")

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(runnerDir, 0755); err != nil {
		return "", "", err
	}

	// Extract models
	entries, err := fs.ReadDir(embeddedFS, "models/v4_int8")
	if err != nil {
		return "", "", fmt.Errorf("failed to read embedded models: %w", err)
	}

	for _, entry := range entries {
		destPath := filepath.Join(targetDir, entry.Name())
		info, statErr := os.Stat(destPath)
		entryInfo, _ := entry.Info()

		// If file exists and size matches, skip rewriting
		if statErr == nil && entryInfo != nil && info.Size() == entryInfo.Size() {
			continue
		}

		data, err := embeddedFS.ReadFile("models/v4_int8/" + entry.Name())
		if err != nil {
			return "", "", err
		}
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return "", "", err
		}
	}

	// Extract runner.py
	runnerDest := filepath.Join(runnerDir, "runner.py")
	runnerData, err := embeddedFS.ReadFile("internal/runner/runner.py")
	if err == nil {
		os.WriteFile(runnerDest, runnerData, 0644)
	}

	return targetDir, runnerDest, nil
}

// initEngine prefers the in-process native bridge and falls back to the
// legacy child-process backend only when no native library can be found.
func (p *Predictor) initEngine(runnerPath string) error {
	libPath, err := findNativeLib(p.opts.NativeLibPath)
	if errors.Is(err, errNativeNotFound) {
		var dlErr error
		if libPath, dlErr = downloadNativeLib(); dlErr == nil {
			err = nil
		} else if !errors.Is(dlErr, errNativeNotFound) {
			err = fmt.Errorf("%w; auto-download failed: %v", err, dlErr)
		}
	}
	if err == nil {
		eng, err := newNativeEngine(libPath, p.modelDir, p.opts)
		if err != nil {
			return err
		}
		p.eng = eng
		return nil
	}
	if !errors.Is(err, errNativeNotFound) {
		return err
	}

	eng, perr := newProcessEngine(p.modelDir, runnerPath, p.opts)
	if perr != nil {
		return fmt.Errorf("%w (set WithNativeLib or REPAROS_NATIVE_LIB to %s); fallback engine failed: %v", err, nativeLibName(), perr)
	}
	p.eng = eng
	return nil
}

// Predict returns the corrected query result. It is safe for concurrent use.
func (p *Predictor) Predict(query string) (*Result, error) {
	start := time.Now()
	res, err := p.eng.predict(query, p.opts.BeamSize, p.opts.NumHypotheses)
	if err != nil {
		return nil, err
	}
	res.InputQuery = query
	res.Changed = res.Top1Query != query
	if res.LatencyMs == 0 {
		res.LatencyMs = float64(time.Since(start).Microseconds()) / 1000
	}
	return res, nil
}

// Close releases the engine and its resources.
func (p *Predictor) Close() error {
	return p.eng.close()
}
