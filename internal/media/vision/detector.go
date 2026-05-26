package vision

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
)

type Result struct {
	IsCapybara bool
	Score      float64
}

type Detector struct {
	cfg    config.CapybaraDetectorConfig
	logger *zap.Logger

	workerMu sync.Mutex
	worker   *detectorWorker
}

type detectorWorker struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
}

func NewDetector(cfg config.CapybaraDetectorConfig, logger *zap.Logger) *Detector {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Detector{cfg: cfg, logger: logger}
}

func (d *Detector) Warmup(ctx context.Context) error {
	if !d.cfg.Enabled {
		return nil
	}
	d.workerMu.Lock()
	defer d.workerMu.Unlock()
	return d.ensureWorkerLocked(ctx)
}

func (d *Detector) DetectFile(ctx context.Context, imagePath string) (Result, error) {
	if !d.cfg.Enabled {
		return Result{}, nil
	}
	return d.detectViaWorker(ctx, imagePath)
}

func (d *Detector) detectViaWorker(ctx context.Context, imagePath string) (Result, error) {
	timeout := time.Duration(d.cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	d.workerMu.Lock()
	defer d.workerMu.Unlock()

	if err := d.ensureWorkerLocked(runCtx); err != nil {
		d.logger.Warn("capybara worker unavailable", zap.Error(err))
		return Result{}, nil
	}

	req, err := json.Marshal(map[string]any{
		"image_path": imagePath,
		"threshold":  d.cfg.ScoreThreshold,
	})
	if err != nil {
		return Result{}, nil
	}
	if _, err = d.worker.stdin.Write(append(req, '\n')); err != nil {
		d.stopWorkerLocked()
		d.logger.Warn("capybara worker write failed", zap.Error(err))
		return Result{}, nil
	}

	line, err := d.readWorkerLine(runCtx)
	if err != nil {
		d.stopWorkerLocked()
		d.logger.Warn("capybara detector failed",
			zap.String("path", imagePath),
			zap.Error(err),
			zap.Bool("likely_oom", isLikelyOOM(err)),
		)
		return Result{}, nil
	}

	var parsed struct {
		IsCapybara bool    `json:"is_capybara"`
		Score      float64 `json:"score"`
		Error      string  `json:"error"`
	}
	if err := json.Unmarshal([]byte(line), &parsed); err != nil {
		d.logger.Warn("capybara detector invalid json", zap.String("output", line), zap.Error(err))
		return Result{}, nil
	}
	if parsed.Error != "" {
		d.logger.Warn("capybara detector error", zap.String("path", imagePath), zap.String("error", parsed.Error))
		return Result{}, nil
	}
	return Result{IsCapybara: parsed.IsCapybara, Score: parsed.Score}, nil
}

func (d *Detector) ensureWorkerLocked(ctx context.Context) error {
	if d.worker != nil {
		return nil
	}
	return d.startWorkerLocked(ctx)
}

func (d *Detector) startWorkerLocked(ctx context.Context) error {
	python, script, err := validateWorkerPaths(d.cfg.PythonPath, d.cfg.ScriptPath)
	if err != nil {
		return err
	}
	threshold := fmt.Sprintf("%g", d.cfg.ScoreThreshold)
	cmd := exec.Command(python, script, "--serve", "--threshold", threshold) //nosec G204
	cmd.Env = append(os.Environ(),
		"OMP_NUM_THREADS=1",
		"MKL_NUM_THREADS=1",
		"OPENBLAS_NUM_THREADS=1",
		"HF_HUB_OFFLINE=1",
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = os.Stderr
	if startErr := cmd.Start(); startErr != nil {
		_ = stdin.Close()
		return fmt.Errorf("start worker: %w", startErr)
	}

	worker := &detectorWorker{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewScanner(stdout),
	}
	d.worker = worker

	readyCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	line, err := d.readWorkerLine(readyCtx)
	if err != nil {
		d.stopWorkerLocked()
		return fmt.Errorf("wait ready: %w", err)
	}
	var ready struct {
		Ready bool `json:"ready"`
	}
	if err := json.Unmarshal([]byte(line), &ready); err != nil || !ready.Ready {
		d.stopWorkerLocked()
		return fmt.Errorf("unexpected ready response: %q", line)
	}
	return nil
}

func (d *Detector) readWorkerLine(ctx context.Context) (string, error) {
	if d.worker == nil {
		return "", fmt.Errorf("worker not started")
	}
	type readResult struct {
		line string
		err  error
	}
	ch := make(chan readResult, 1)
	go func() {
		if !d.worker.stdout.Scan() {
			err := d.worker.stdout.Err()
			if err == nil {
				err = io.EOF
			}
			ch <- readResult{"", err}
			return
		}
		ch <- readResult{d.worker.stdout.Text(), nil}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case res := <-ch:
		return res.line, res.err
	}
}

func (d *Detector) stopWorkerLocked() {
	if d.worker == nil {
		return
	}
	_ = d.worker.stdin.Close()
	if d.worker.cmd.Process != nil {
		_ = d.worker.cmd.Process.Kill()
	}
	_ = d.worker.cmd.Wait()
	d.worker = nil
}

func (d *Detector) DetectBytes(ctx context.Context, data []byte) (Result, error) {
	if !d.cfg.Enabled || len(data) == 0 {
		return Result{}, nil
	}
	f, err := os.CreateTemp("", "capybara-*.img")
	if err != nil {
		return Result{}, fmt.Errorf("temp file: %w", err)
	}
	path := f.Name()
	defer func() {
		_ = f.Close()
		_ = os.Remove(path)
	}()
	if _, err = f.Write(data); err != nil {
		return Result{}, fmt.Errorf("write temp: %w", err)
	}
	if err = f.Close(); err != nil {
		return Result{}, fmt.Errorf("close temp: %w", err)
	}
	return d.DetectFile(ctx, path)
}

func DefaultScriptPath() string {
	return filepath.Join("scripts", "vision", "detect_capybara.py")
}

func validateWorkerPaths(pythonPath, scriptPath string) (string, string, error) {
	pythonPath = strings.TrimSpace(pythonPath)
	scriptPath = strings.TrimSpace(scriptPath)
	if pythonPath == "" {
		pythonPath = "python3"
	}
	switch pythonPath {
	case "python3", "python":
	default:
		clean := filepath.Clean(pythonPath)
		if !filepath.IsAbs(clean) || strings.Contains(clean, "..") {
			return "", "", fmt.Errorf("invalid python path %q", pythonPath)
		}
		pythonPath = clean
	}

	scriptPath = filepath.Clean(scriptPath)
	if scriptPath == "" || strings.Contains(scriptPath, "..") {
		return "", "", fmt.Errorf("invalid script path %q", scriptPath)
	}
	if filepath.Base(scriptPath) != "detect_capybara.py" {
		return "", "", fmt.Errorf("unexpected capybara script %q", scriptPath)
	}
	return pythonPath, scriptPath, nil
}

func isLikelyOOM(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return msg == "signal: killed" || msg == "EOF"
}
