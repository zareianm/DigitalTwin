package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

// DeleteTask deletes an existing task
//
//	@Summary		deletes an existing task
//	@Description	deletes an existing task
//	@Tags			tasks
//	@Accept			json
//	@Produce		json
//	@Param			task_id	path		int	true	"Task ID"
//	@Success		204
//	@Router			/api/v1/tasks/delete/{task_id} [delete]
//	@Security		BearerAuth
func (app *application) deleteTask(c *gin.Context) {
	task_id, err := strconv.Atoi(c.Param("task_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Invalid task ID"})
		return
	}

	userIdStr, _ := c.Get("user_id")
	userId, _ := userIdStr.(int)

	existingTask, err := app.models.Tasks.Get(userId, task_id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"Error": "Failed to retrieve task"})
		return
	}

	if existingTask == nil {
		c.JSON(http.StatusNotFound, gin.H{"Error": "Task not found"})
		return
	}

	if existingTask.CronId != nil && *existingTask.CronId > 0 {
		app.cr.Remove(cron.EntryID(*existingTask.CronId))
	}

	if err := app.models.Tasks.Delete(task_id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
