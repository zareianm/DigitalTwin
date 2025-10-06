package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

// StopTask stops a task from running
//
//	@Summary		stops a task from running
//	@Description	stops a task from running
//	@Tags			tasks
//	@Accept			json
//	@Produce		json
//	@Param			task_id	path		int	true	"Task ID"
//	@Success		200	{object} StopTaskResultModel
//	@Router			/api/v1/tasks/stopTask/{task_id} [post]
//	@Security		BearerAuth
func (app *application) stopTask(c *gin.Context) {
	taskId, err := strconv.Atoi(c.Param("task_id"))

	userIdStr, _ := c.Get("user_id")
	userId, _ := userIdStr.(int)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	task, err := app.models.Tasks.Get(userId, taskId)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retreive task details"})
		return
	}

	if task == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	if task.CronId != nil && *task.CronId > 0 {
		app.cr.Remove(cron.EntryID(*task.CronId))
	}

	result := StopTaskResultModel{
		Success: true,
	}

	c.JSON(http.StatusOK, result)
}

type StopTaskResultModel struct {
	Success bool `"json:"success"`
}
