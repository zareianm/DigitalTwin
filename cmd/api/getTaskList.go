package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GetTasks returns tasks with pagination
//
//	@Summary		returns tasks with pagination
//	@Description	returns tasks with pagination
//	@Tags			tasks
//	@Accept			json
//	@Produce		json
//	@Param			page	path		int	true	"number of page"
//	@Success		200		{object}	[]TaskOutputModel
//	@Router			/api/v1/tasks/getTaskList/{page} [get]
//	@Security		BearerAuth
func (app *application) getAllTasks(c *gin.Context) {

	page, err := strconv.Atoi(c.Param("page"))

	if err != nil || page < 1 {
		page = 1
	}

	userIdStr, _ := c.Get("user_id")
	userId, _ := userIdStr.(int)

	tasks, err := app.models.Tasks.GetUserTaskWithPagination(userId, page)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retreive tasks"})
		return
	}

	output := make([]TaskOutputModel, len(tasks))

	now := time.Now().UTC()

	for i, t := range tasks {

		var operatingHours float64
		if t.EndTime != nil && now.After(*t.EndTime) {
			operatingHours = t.EndTime.Sub(t.StartTime).Hours()
		} else {
			operatingHours = now.Sub(t.StartTime).Hours()
		}

		if operatingHours < 0 {
			operatingHours = 0
		}

		output[i] = TaskOutputModel{
			TaskId:               t.TaskId,
			DeviceId:             t.DeviceId,
			CreatedAt:            t.CreatedAt,
			IsActive:             t.Active && (t.StartTime.Before(time.Now()) && (t.EndTime == nil || t.EndTime.After(time.Now()))),
			PluginOperatingHours: operatingHours,
			TaskName:             t.TaskName,
			DeviceName:           t.DeviceName,
		}
	}

	c.JSON(http.StatusOK, output)
}

type TaskOutputModel struct {
	TaskId               int       `json:"taskId"`
	DeviceId             int       `json:"deviceId"`
	CreatedAt            time.Time `json:"createdAt"`
	IsActive             bool      `json:"isActive"`
	PluginOperatingHours float64   `json:"pluginOperatingHours"`
	TaskName             string    `json:"taskName"`
	DeviceName           string    `json:"deviceName"`
}
