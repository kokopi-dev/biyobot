package services

import (
	"biyobot/models"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

type ExternalRunner struct {
	Executable string
	Args       []string
	Timeout    time.Duration
	WorkingDir string
	Env        []string // extra env vars in "KEY=VALUE" form
}

func (e *ExternalRunner) Run(input json.RawMessage) models.ServiceResult {
	timeout := e.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, e.Executable, e.Args...)

	if e.WorkingDir != "" {
		cmd.Dir = e.WorkingDir
	}
	if len(e.Env) > 0 {
		cmd.Env = append(cmd.Environ(), e.Env...)
	}

	// Pass input JSON to the process via stdin
	if len(input) > 0 {
		cmd.Stdin = bytes.NewReader(input)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return models.Failure(fmt.Sprintf("timed out after %s", timeout))
		}
		// Non-zero exit: try to parse stdout as ServiceResult anyway,
		// fall back to a generic error with stderr
		if result, parseErr := parseOutput(stdout.Bytes()); parseErr == nil {
			return result
		}
		msg := fmt.Sprintf("process exited with error: %v", err)
		if s := stderr.String(); s != "" {
			msg += "\nstderr: " + s
		}
		return models.Failure(msg)
	}

	result, err := parseOutput(stdout.Bytes())
	if err != nil {
		raw := stdout.String()
		if len(raw) > 200 {
			raw = raw[:200] + "..."
		}
		return models.Failure(fmt.Sprintf("invalid output (expected ServiceResult JSON): %v\ngot: %s", err, raw))
	}
	return result
}

func parseOutput(b []byte) (models.ServiceResult, error) {
	b = bytes.TrimSpace(b)
	var result models.ServiceResult
	if err := json.Unmarshal(b, &result); err != nil {
		return models.ServiceResult{}, err
	}
	return result, nil
}
