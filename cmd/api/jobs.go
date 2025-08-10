package main

import (
	"DigitalTwin/internal/database"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type SaveJobResult struct {
	Error   string `json:"errors"`
	Success bool   `json:"success"`
}

// CreateJob creates a new job
//
//		@Summary		Creates a new job
//		@Description	Creates a new job
//		@Tags			jobs
//	    @Accept       multipart/form-data
//		@Produce		json
//	    @Param        file  formData  file  true  "C++ source file to scan"
//		@Success		201		{object}	SaveJobResult
//		@Router			/api/v1/jobs/create [post]
func (app *application) createJob(c *gin.Context) {

	// 1) get file
	f, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file parameter required"})
		return
	}
	defer f.Close()

	// 2) save to a temp file
	tmpDir, _ := ioutil.TempDir("", "cppcheck")
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, header.Filename)
	out, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot save file"})
		return
	}
	defer out.Close()
	io.Copy(out, f)

	errorStr := checkCppError(filePath)

	// 7) build and return result
	result := SaveJobResult{
		Error:   errorStr,
		Success: errorStr == "",
	}
	c.JSON(http.StatusOK, result)
}

func checkCppError(filePath string) string {

	// 3) run syntax check with g++
	syntaxErrs := runCommand("g++", "-std=c++17", "-fsyntax-only", filePath)

	if syntaxErrs != "" {
		return syntaxErrs
	}

	// 4) read source for heuristics
	contentBytes, _ := ioutil.ReadFile(filePath)
	src := string(contentBytes)

	// 5) count loops
	loopCount := countLoops(src)
	bigLoop := loopCount > 20 // arbitrary threshold

	if bigLoop {
		return "big loop detected"
	}

	// 6) scan for syscall-related keywords
	syscallKeys := scanSyscalls(src)
	syscallWarn := len(syscallKeys) > 0

	if syscallWarn {
		return "system call detected"
	}

	return ""
}

// ScheduleTask schedules a task
//
//		@Summary		schedules a task
//		@Description	schedules a task
//		@Tags			jobs
//	    @Accept       	json
//		@Produce		json
//		@Param			task	body		database.Task	true	"Task"
//		@Success		201		{object}	SaveJobResult
//		@Router			/api/v1/jobs/scheduleTask [post]
func (app *application) scheduleTask(c *gin.Context) {

	var task database.Task

	if err := c.BindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := app.models.Tasks.Insert(&task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	app.register(task)
	c.JSON(http.StatusCreated, task)
}

// runCommand executes a command and returns its stderr/stdout lines.
func runCommand(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()

	combined := stderr.String() + out.String()
	if err != nil {
		// append the Go error (e.g. "exit status 1" or "executable file not found")
		combined += "\n" + err.Error()

		return combined
	}
	return ""
}

// countLoops returns the number of 'for' and 'while' occurrences.
func countLoops(src string) int {
	forRe := regexp.MustCompile(`\bfor\s*\(`)
	whileRe := regexp.MustCompile(`\bwhile\s*\(`)
	return len(forRe.FindAllString(src, -1)) + len(whileRe.FindAllString(src, -1))
}

// scanSyscalls looks for common C/C++ system-call or dangerous API keywords.
func scanSyscalls(src string) []string {
	keywords := []string{
		`\bsystem\s*\(`,
		`\bexec\s*vp?\s*\(`,
		`\bfork\s*\(`,
		`\bpopen\s*\(`,
		`\bopen\s*\(`,
		`\bioctl\s*\(`,
		`\bptrace\s*\(`,
		`\bsocket\s*\(`,
		`\bconnect\s*\(`,
		`\bbind\s*\(`,
		`\baccept\s*\(`,
	}
	found := make(map[string]struct{})
	for _, kw := range keywords {
		re := regexp.MustCompile(kw)
		if loc := re.FindStringIndex(src); loc != nil {
			// extract the bare keyword (e.g. "system(")
			match := re.FindString(src)
			found[match] = struct{}{}
		}
	}
	var list []string
	for k := range found {
		list = append(list, k)
	}
	return list
}

func (app *application) runMissedTasks(db *sql.DB) {
	// --------------------------- scheduler setup ----------------------------

	// load previously persisted tasks
	rows, err := db.Query(`SELECT id, name, cron_spec, payload, last_run FROM tasks`)
	if err != nil {
		log.Fatalf("query tasks: %v", err)
	}
	for rows.Next() {
		var (
			t          database.Task
			lastRunInt int64
		)
		if err := rows.Scan(&t.ID, &t.Name, &t.CronSpec, &t.Payload, &lastRunInt); err != nil {
			log.Printf("scan task: %v", err)
			continue
		}
		if lastRunInt != 0 {
			t.LastRun = time.Unix(lastRunInt, 0)
		}
		app.register(t)
	}
	rows.Close()
}

func (app *application) register(t database.Task) {
	// capture by value so each closure has its own copy
	_, err := app.cr.AddFunc(t.CronSpec, func() { app.runJob(t) })
	if err != nil {
		log.Printf("failed to register task %d: %v", t.ID, err)
	}
}

func (app *application) runJob(t database.Task) {
	log.Printf("Running task %s (ID %d) with payload: %s", t.Name, t.ID, t.Payload)

	app.models.Tasks.UpdateLastExecute(&t)
}

const (
	compileTimeout = 2000 * time.Second
	runTimeout     = 2000 * time.Second
	maxUploadSize  = 1 << 20 // 1 MiB; adjust as needed
)

const dockerImage = "gcc:13"

// CreateJob creates a new job
//
//		@Summary		Creates a new job
//		@Description	Creates a new job
//		@Tags			jobs
//	    @Accept       multipart/form-data
//		@Produce		json
//	    @Param        file  formData  file  true  "C++ source file to scan"
//		@Success		201		{object}	SaveJobResult
//		@Router			/api/v1/jobs/testDocker [post]
func (app *application) handleRun(c *gin.Context) {
	// Read file
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'source' file: " + err.Error()})
		return
	}

	// Create temp working directory
	workDir, err := os.MkdirTemp("", "cppwork-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create temp dir"})
		return
	}
	defer os.RemoveAll(workDir)

	// Save uploaded file as main.cpp
	cppPath := filepath.Join(workDir, "main.cpp")
	if err := saveUploadedFile(fileHeader, cppPath); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Compile inside Docker
	appPath := filepath.Join(workDir, "app")
	if err := dockerCompile(c, workDir, cppPath, appPath); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, context.DeadlineExceeded) {
			status = http.StatusGatewayTimeout
		}
		c.JSON(status, gin.H{"error": fmt.Sprintf("compile failed: %v", err)})
		return
	}

	// Parse args: support repeated form field and JSON array
	// args := parseArgs(c)

	args := []string{"alice"}

	// Run the binary inside Docker with limits
	stdout, stderr, exitCode, runErr := dockerRun(workDir, appPath, args)
	if runErr != nil && !errors.Is(runErr, context.DeadlineExceeded) {
		// Non-timeout run error (e.g., container failed to start)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":    fmt.Sprintf("run failed: %v", runErr),
			"stdout":   string(stdout),
			"stderr":   string(stderr),
			"exitCode": exitCode,
		})
		return
	}

	status := http.StatusOK
	if errors.Is(runErr, context.DeadlineExceeded) {
		status = http.StatusGatewayTimeout
	}
	c.JSON(status, gin.H{
		"stdout":   string(stdout),
		"stderr":   string(stderr),
		"exitCode": exitCode,
	})
}

func saveUploadedFile(fh *multipart.FileHeader, dest string) error {
	// Basic extension guard (optional): only allow .cpp/.cc/.cxx
	name := strings.ToLower(fh.Filename)
	if !(strings.HasSuffix(name, ".cpp") || strings.HasSuffix(name, ".cc") || strings.HasSuffix(name, ".cxx")) {
		return fmt.Errorf("unsupported file extension; expected .cpp/.cc/.cxx")
	}

	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, src); err != nil {
		return err
	}
	return nil
}

func dockerCompile(c *gin.Context, workDir, cppPath, appPath string) error {
	ctx, cancel := context.WithTimeout(c, compileTimeout)
	defer cancel()

	cmd := exec.CommandContext(
		ctx, "docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/work", workDir),
		"-w", "/work",
		dockerImage, "bash", "-lc",
		fmt.Sprintf("g++ -std=c++17 -O2 -pipe -static-libstdc++ -static-libgcc main.cpp -o app"),
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = nil

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return ctx.Err()
		}
		return fmt.Errorf("%v: %s", err, stderr.String())
	}
	// Ensure app exists
	if _, err := os.Stat(appPath); err != nil {
		return fmt.Errorf("compiled binary missing: %v", err)
	}
	return nil
}

func dockerRun(workDir, appPath string, args []string) (stdout, stderr []byte, exitCode int, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	// Build the docker run command with strong sandboxing
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
		"--tmpfs", "/run:rw,noexec,nosuid,nodev,size=8m",
		dockerImage,
		"/work/app",
	}
	cmdArgs := append(base, args...)

	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()
	stdout = outBuf.Bytes()
	stderr = errBuf.Bytes()

	exitCode = 0
	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		exitCode = exitErr.ExitCode()
	}

	if ctx.Err() == context.DeadlineExceeded {
		return stdout, stderr, exitCode, ctx.Err()
	}
	return stdout, stderr, exitCode, runErr
}

func parseArgs(c *gin.Context) []string {
	args := c.PostFormArray("args")
	if len(args) > 0 {
		return args
	}
	// Try JSON array in a single form field "args"
	raw := c.PostForm("args")
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var fromJSON []string
	if err := json.Unmarshal([]byte(raw), &fromJSON); err == nil {
		return fromJSON
	}
	return []string{raw} // fallback: treat as single arg
}
