package jobService

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	compileTimeout = 200 * time.Second
	runTimeout     = 200 * time.Second
)

const dockerImage = "gcc:13"

type runReq struct {
	Path string   `json:"path"` // absolute path to *.cpp
	Args []string `json:"args"`
}

func RunCppInDocker(path string) (string, int, error) {
	args := []string{"Alice"}

	req := runReq{Path: path, Args: args}

	// Resolve to absolute, symlink-free path
	resolved, err := filepath.EvalSymlinks(req.Path)
	if err != nil {
		return "", -1, errors.New(fmt.Sprintf("invalid path: %v", err))
	}
	absPath, err := filepath.Abs(resolved)
	if err != nil {
		return "", -1, errors.New(fmt.Sprintf("cannot resolve path: %v", err))
	}

	// Validate file
	st, err := os.Stat(absPath)
	if err != nil || st.IsDir() {
		return "", -1, errors.New("file does not exist or is a directory")
	}
	lower := strings.ToLower(absPath)
	if !(strings.HasSuffix(lower, ".cpp") || strings.HasSuffix(lower, ".cc") || strings.HasSuffix(lower, ".cxx")) {
		return "", -1, errors.New("unsupported extension; expected .cpp/.cc/.cxx")
	}

	// Separate writable build dir from source dir
	srcDir := filepath.Dir(absPath)
	buildDir, err := os.MkdirTemp("", "cppbuild-*")
	if err != nil {
		return "", -1, errors.New("failed to create build directory")
	}
	defer os.RemoveAll(buildDir)

	// Compile
	if err := dockerCompileSeparated(srcDir, absPath, buildDir); err != nil {
		return "", -1, errors.New(fmt.Sprintf("compile failed: %v", err))
	}

	// Run
	stdout, _, exitCode, runErr := dockerRun(buildDir, []string{"/work/app"}, req.Args)

	var errStr string
	if runErr != nil {
		errStr = runErr.Error()

		return "", -exitCode, errors.New(errStr)
	}

	return string(stdout), exitCode, nil
}

func dockerCompileSeparated(srcDir, absSource, buildDir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), compileTimeout)
	defer cancel()

	// Make paths Docker-friendly across OSes
	srcBase := filepath.Base(absSource)

	cmd := exec.CommandContext(
		ctx, "docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/src:ro", srcDir),
		"-v", fmt.Sprintf("%s:/build", buildDir),
		"-w", "/build",
		dockerImage, "bash", "-lc",
		fmt.Sprintf("g++ -std=c++17 -O2 -pipe -static-libstdc++ -static-libgcc -I/src /src/%s -o /build/app", srcBase),
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

func dockerRun(workDir string, entry []string, args []string) ([]byte, []byte, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	base := []string{
		"run", "--rm",
		"--network", "none",
		"--pids-limit", "128",
		"-m", "256m",
		"--cpus", "0.5",
		"--read-only",
		"--security-opt", "no-new-privileges",
		"-v", fmt.Sprintf("%s:/work:ro", workDir),
		"-w", "/work",
		"--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=16m",
		dockerImage,
	}
	cmdArgs := append(base, append(entry, args...)...)

	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
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
