package jsService

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
	runTimeout = 200 * time.Second
)

const dockerImage = "node:20-slim"

func RunJsInDocker(path string, args []string) (string, int, error) {
	// Create a temporary directory in /tmp (which is accessible from both container and host)
	tempDir, err := os.MkdirTemp("/tmp", "jsbuild-")
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
	if !strings.HasSuffix(lower, ".js") && !strings.HasSuffix(lower, ".mjs") {
		return "", -1, errors.New("unsupported extension; expected .js or .mjs")
	}

	// Copy source file to temp directory
	srcFile, err := os.Open(path)
	if err != nil {
		return "", -1, fmt.Errorf("failed to open source file: %v", err)
	}
	defer srcFile.Close()

	destPath := filepath.Join(tempDir, "script.js")
	destFile, err := os.Create(destPath)
	if err != nil {
		return "", -1, fmt.Errorf("failed to create dest file: %v", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return "", -1, fmt.Errorf("failed to copy source file: %v", err)
	}

	// Run JavaScript file
	stdout, _, exitCode, runErr := dockerRunTemp(tempDir, args)
	if runErr != nil {
		return "", exitCode, runErr
	}

	return string(stdout), exitCode, nil
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
		"node",
		// Security flags for Node.js
		"--max-old-space-size=128", // Limit heap to 128MB
		"--no-warnings",
		"--disallow-code-generation-from-strings", // Disable eval and Function constructor
		"script.js",
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
