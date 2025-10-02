package pythonService

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	syntaxCheckTimeout = 10 * time.Second
	runTimeout         = 200 * time.Second
)

const dockerImage = "python:3.11-slim"

func RunPythonInDocker(path string, args []string) (string, int, error) {
	// Create a temporary directory in /tmp (which is accessible from both container and host)
	tempDir, err := os.MkdirTemp("/tmp", "pybuild-")
	if err != nil {
		return "", -1, fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Validate file
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return "", -1, errors.New("file does not exist or is a directory")
	}

	lower := strings.ToLower(path)
	if !strings.HasSuffix(lower, ".py") {
		return "", -1, errors.New("unsupported extension; expected .py")
	}

	// Copy source file to temp directory
	srcFile, err := os.Open(path)
	if err != nil {
		return "", -1, fmt.Errorf("failed to open source file: %v", err)
	}
	defer srcFile.Close()

	destPath := filepath.Join(tempDir, "script.py")
	destFile, err := os.Create(destPath)
	if err != nil {
		return "", -1, fmt.Errorf("failed to create dest file: %v", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return "", -1, fmt.Errorf("failed to copy source file: %v", err)
	}

	// Check syntax first
	if err := dockerSyntaxCheck(tempDir); err != nil {
		return "", -1, fmt.Errorf("syntax check failed: %v", err)
	}

	// Run
	stdout, _, exitCode, runErr := dockerRunTemp(tempDir, args)
	if runErr != nil {
		return "", exitCode, runErr
	}

	return string(stdout), exitCode, nil
}

func dockerSyntaxCheck(tempDir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), syntaxCheckTimeout)
	defer cancel()

	// Use bind mount with the temp directory
	// /tmp should be accessible from the host
	cmd := exec.CommandContext(
		ctx, "docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/work", tempDir),
		"-w", "/work",
		dockerImage, "python", "-m", "py_compile", "script.py",
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return ctx.Err()
		}
		return fmt.Errorf("%v: %s", err, stderr.String())
	}
	return nil
}

func dockerRunTemp(tempDir string, args []string) ([]byte, []byte, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	// Build the Docker command
	dockerArgs := []string{
		"run", "--rm",
		"--network", "none",
		"--pids-limit", "128",
		"-m", "256m",
		"--cpus", "0.5",
		"--security-opt", "no-new-privileges",
		"-v", fmt.Sprintf("%s:/work:ro", tempDir),
		"-w", "/work",
		"--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=16m",
		dockerImage,
		"python", "-u", "script.py",
	}

	// Add user arguments
	dockerArgs = append(dockerArgs, args...)

	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()
	exitCode := 0

	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		exitCode = exitErr.ExitCode()
	}

	if ctx.Err() == context.DeadlineExceeded {
		return outBuf.Bytes(), errBuf.Bytes(), exitCode, ctx.Err()
	}

	return outBuf.Bytes(), errBuf.Bytes(), exitCode, runErr
}
