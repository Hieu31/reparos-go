package reparos

import (
	"bufio"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
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

	mu     sync.Mutex
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	closed bool
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

func (p *Predictor) initEngine(preferredRunner string) error {
	var runnerPath = preferredRunner

	if runnerPath == "" {
		execDir, _ := os.Getwd()
		possiblePaths := []string{
			filepath.Join(execDir, "internal", "runner", "runner.py"),
			filepath.Join(p.modelDir, "..", "internal", "runner", "runner.py"),
			filepath.Join(p.modelDir, "..", "..", "internal", "runner", "runner.py"),
		}

		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				runnerPath = path
				break
			}
		}
	}

	if runnerPath == "" {
		return errors.New("cannot locate internal/runner/runner.py")
	}

	// Choose python executable
	var pyCmd *exec.Cmd
	if _, err := exec.LookPath("uv"); err == nil {
		pyCmd = exec.Command("uv", "run", "python", runnerPath, p.modelDir, p.opts.Device, p.opts.ComputeType)
	} else if _, err := exec.LookPath("python3"); err == nil {
		pyCmd = exec.Command("python3", runnerPath, p.modelDir, p.opts.Device, p.opts.ComputeType)
	} else {
		pyCmd = exec.Command("python", runnerPath, p.modelDir, p.opts.Device, p.opts.ComputeType)
	}

	stdin, err := pyCmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to open stdin pipe: %w", err)
	}

	stdoutPipe, err := pyCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to open stdout pipe: %w", err)
	}

	pyCmd.Stderr = os.Stderr

	if err := pyCmd.Start(); err != nil {
		return fmt.Errorf("failed to start inference runner: %w", err)
	}

	reader := bufio.NewReader(stdoutPipe)

	// Wait for READY signal
	readyChan := make(chan error, 1)
	go func() {
		line, err := reader.ReadString('\n')
		if err != nil {
			readyChan <- fmt.Errorf("failed to read from runner: %w", err)
			return
		}
		if line != "READY\n" && line != "READY\r\n" {
			readyChan <- fmt.Errorf("unexpected runner startup message: %s", line)
			return
		}
		readyChan <- nil
	}()

	select {
	case err := <-readyChan:
		if err != nil {
			pyCmd.Process.Kill()
			return err
		}
	case <-time.After(15 * time.Second):
		pyCmd.Process.Kill()
		return errors.New("inference runner startup timed out after 15s")
	}

	p.cmd = pyCmd
	p.stdin = stdin
	p.stdout = reader
	return nil
}

type runnerRequest struct {
	Query         string `json:"query"`
	BeamSize      int    `json:"beam_size"`
	NumHypotheses int    `json:"num_hypotheses"`
}

type runnerResponse struct {
	Top1Query  string    `json:"top1_query"`
	Hypotheses []string  `json:"hypotheses"`
	Scores     []float64 `json:"scores"`
	LatencyMs  float64   `json:"latency_ms"`
	Changed    bool      `json:"changed"`
	Error      string    `json:"error"`
}

// Predict sends a query to the model and returns the corrected query result.
func (p *Predictor) Predict(query string) (*Result, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, errors.New("predictor is already closed")
	}

	req := runnerRequest{
		Query:         query,
		BeamSize:      p.opts.BeamSize,
		NumHypotheses: p.opts.NumHypotheses,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	if _, err := p.stdin.Write(append(reqBytes, '\n')); err != nil {
		return nil, fmt.Errorf("failed to write to runner: %w", err)
	}

	line, err := p.stdout.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read runner response: %w", err)
	}

	var resp runnerResponse
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		return nil, fmt.Errorf("invalid json response from runner: %w", err)
	}

	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}

	return &Result{
		InputQuery: query,
		Top1Query:  resp.Top1Query,
		Hypotheses: resp.Hypotheses,
		Scores:     resp.Scores,
		Changed:    resp.Changed,
		LatencyMs:  resp.LatencyMs,
	}, nil
}

// Close gracefully shuts down the predictor process and frees resources.
func (p *Predictor) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}
	p.closed = true

	if p.stdin != nil {
		p.stdin.Write([]byte("QUIT\n"))
		p.stdin.Close()
	}

	if p.cmd != nil && p.cmd.Process != nil {
		done := make(chan error, 1)
		go func() { done <- p.cmd.Wait() }()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			p.cmd.Process.Kill()
		}
	}

	return nil
}
