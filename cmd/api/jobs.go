package main

import (
	"DigitalTwin/internal/database"
	"bytes"
	"database/sql"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	// var job database.Job

	// if err := c.ShouldBindJSON(&job); err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 	return
	// }

	// err := app.models.Jobs.Insert(&job)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create job"})
	// 	return
	// }

	// c.JSON(http.StatusCreated, job)

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
