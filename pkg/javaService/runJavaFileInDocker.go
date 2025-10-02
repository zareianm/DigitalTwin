package javaService

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	compileTimeout = 200 * time.Second
	runTimeout     = 200 * time.Second
)

const dockerImage = "openjdk:17-slim"

func RunJavaInDocker(path string, args []string) (string, int, error) {
	// Create a temporary directory in /tmp (which is accessible from both container and host)
	tempDir, err := os.MkdirTemp("/tmp", "javabuild-")
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
	if !strings.HasSuffix(lower, ".java") {
		return "", -1, errors.New("unsupported extension; expected .java")
	}

	// Extract class name from file
	className, err := extractClassName(path)
	if err != nil {
		return "", -1, fmt.Errorf("failed to extract class name: %v", err)
	}

	// Copy source file to temp directory
	srcFile, err := os.Open(path)
	if err != nil {
		return "", -1, fmt.Errorf("failed to open source file: %v", err)
	}
	defer srcFile.Close()

	destPath := filepath.Join(tempDir, className+".java")
	destFile, err := os.Create(destPath)
	if err != nil {
		return "", -1, fmt.Errorf("failed to create dest file: %v", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return "", -1, fmt.Errorf("failed to copy source file: %v", err)
	}

	// Compile using temp directory
	if err := dockerCompileTemp(tempDir, className); err != nil {
		return "", -1, fmt.Errorf("compile failed: %v", err)
	}

	// Run
	stdout, _, exitCode, runErr := dockerRunTemp(tempDir, className, args)
	if runErr != nil {
		return "", exitCode, runErr
	}

	return string(stdout), exitCode, nil
}

// extractClassName extracts the public class name from a Java file
func extractClassName(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// Look for public class declaration
	// This regex matches: public class ClassName
	re := regexp.MustCompile(`public\s+class\s+([A-Za-z_][A-Za-z0-9_]*)\s*\{`)
	matches := re.FindSubmatch(content)
	if len(matches) > 1 {
		return string(matches[1]), nil
	}

	// If no public class, look for any class declaration
	re = regexp.MustCompile(`class\s+([A-Za-z_][A-Za-z0-9_]*)\s*\{`)
	matches = re.FindSubmatch(content)
	if len(matches) > 1 {
		return string(matches[1]), nil
	}

	return "", errors.New("could not find class name in Java file")
}

func dockerCompileTemp(tempDir, className string) error {
	ctx, cancel := context.WithTimeout(context.Background(), compileTimeout)
	defer cancel()

	// Compile Java file
	cmd := exec.CommandContext(
		ctx, "docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/work", tempDir),
		"-w", "/work",
		dockerImage, "javac",
		className+".java",
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

func dockerRunTemp(tempDir, className string, args []string) ([]byte, []byte, int, error) {
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
		"java",
		// Security flags for Java
		"-Djava.security.manager=allow",
		"-Xmx128m", // Max heap size
		"-Xss256k", // Stack size
		className,
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
