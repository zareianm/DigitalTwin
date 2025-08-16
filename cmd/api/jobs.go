package main

import (
	"DigitalTwin/internal/database"
	"database/sql"
	"log"
	"net/http"
	"time"

	"DigitalTwin/pkg/jobService"

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

	errorStr := jobService.CheckCppError(f, header)

	if errorStr != "" {
		result := SaveJobResult{
			Error:   errorStr,
			Success: false,
		}
		c.JSON(http.StatusOK, result)
		return
	}

	filepath, err := jobService.SaveFile(header)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Run the C++ code in Docker
	_, _, runErr := jobService.RunCppInDocker(filepath)

	if runErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": runErr.Error()})
		return
	}

	result := SaveJobResult{
		Error:   filepath,
		Success: true,
	}
	c.JSON(http.StatusOK, result)
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
