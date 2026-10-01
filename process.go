package reparos

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// processEngine is the legacy backend: it talks JSON lines to a child process
// (a standalone reparos_engine binary, or runner.py under uv/python).
// It is only a fallback for when the native bridge library is unavailable.
type processEngine struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	closed bool
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

func newProcessEngine(modelDir, preferredRunner string, opts Options) (*processEngine, error) {
	runnerPath := preferredRunner

	// 1. Standalone engine binary (no Python required).
	cacheBase, _ := os.UserCacheDir()
	execDir, _ := os.Getwd()
	possibleEngines := []string{
		filepath.Join(cacheBase, "reparos-go", "engine", "reparos_engine.exe"),
		filepath.Join(cacheBase, "reparos-go", "engine", "reparos_engine"),
		filepath.Join(execDir, "internal", "engine", "reparos_engine.exe"),
		filepath.Join(execDir, "internal", "engine", "reparos_engine"),
		filepath.Join(modelDir, "..", "internal", "engine", "reparos_engine.exe"),
	}

	var engineCmd *exec.Cmd
	for _, eng := range possibleEngines {
		if _, err := os.Stat(eng); err == nil {
			engineCmd = exec.Command(eng, modelDir, opts.Device, opts.ComputeType)
			break
		}
	}

	// 2. Fall back to the Python runner.
	if engineCmd == nil {
		if runnerPath == "" {
			possiblePaths := []string{
				filepath.Join(execDir, "internal", "runner", "runner.py"),
				filepath.Join(modelDir, "..", "internal", "runner", "runner.py"),
				filepath.Join(modelDir, "..", "..", "internal", "runner", "runner.py"),
			}
			for _, path := range possiblePaths {
				if _, err := os.Stat(path); err == nil {
					runnerPath = path
					break
				}
			}
		}
		if runnerPath == "" {
			return nil, errors.New("cannot locate inference engine or runner.py")
		}

		if _, err := exec.LookPath("uv"); err == nil {
			// uv can auto-provision ctranslate2 and sentencepiece on the fly
			engineCmd = exec.Command("uv", "run", "--python", "3.11", "--with", "ctranslate2", "--with", "sentencepiece", "python", runnerPath, modelDir, opts.Device, opts.ComputeType)
		} else if _, err := exec.LookPath("python3"); err == nil {
			engineCmd = exec.Command("python3", runnerPath, modelDir, opts.Device, opts.ComputeType)
		} else {
			engineCmd = exec.Command("python", runnerPath, modelDir, opts.Device, opts.ComputeType)
		}
	}

	stdin, err := engineCmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open stdin pipe: %w", err)
	}
	stdoutPipe, err := engineCmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open stdout pipe: %w", err)
	}
	engineCmd.Stderr = os.Stderr

	if err := engineCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start inference engine: %w", err)
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
			engineCmd.Process.Kill()
			return nil, err
		}
	case <-time.After(15 * time.Second):
		engineCmd.Process.Kill()
		return nil, errors.New("inference runner startup timed out after 15s")
	}

	return &processEngine{cmd: engineCmd, stdin: stdin, stdout: reader}, nil
}

func (p *processEngine) predict(query string, beamSize, numHypotheses int) (*Result, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, errors.New("predictor is already closed")
	}

	reqBytes, err := json.Marshal(runnerRequest{Query: query, BeamSize: beamSize, NumHypotheses: numHypotheses})
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
		Top1Query:  resp.Top1Query,
		Hypotheses: resp.Hypotheses,
		Scores:     resp.Scores,
		Changed:    resp.Changed,
		LatencyMs:  resp.LatencyMs,
	}, nil
}

func (p *processEngine) close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}
	p.closed = true

	p.stdin.Write([]byte("QUIT\n"))
	p.stdin.Close()

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
