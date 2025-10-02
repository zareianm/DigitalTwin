package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GetTasks returns all tasks
//
//	@Summary		Returns all tasks
//	@Description	Returns all tasks
//	@Tags			tasks
//	@Accept			json
//	@Produce		json
//	@Success		200		{object}	[]TaskOutputModel
//	@Router			/api/v1/tasks/getTaskList [get]
func (app *application) getAllTasks(c *gin.Context) {

	tasks, err := app.models.Tasks.GetAll()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retreive tasks"})
		return
	}

	output := make([]TaskOutputModel, len(tasks))

	now := time.Now().UTC()

	for i, t := range tasks {

		var operatingHours float64
		if now.After(t.EndTime) {
			operatingHours = t.EndTime.Sub(t.StartTime).Hours()
		} else {
			operatingHours = now.Sub(t.StartTime).Hours()
		}

		if operatingHours < 0 {
			operatingHours = 0
		}

		output[i] = TaskOutputModel{
			TaskId:               t.TaskId,
			MachineId:            t.MachineId,
			CreatedAt:            t.CreatedAt,
			IsActive:             t.StartTime.Before(time.Now()) && t.EndTime.After(time.Now()),
			PluginOperatingHours: operatingHours,
			TaskName:             t.TaskName,
		}
	}

	c.JSON(http.StatusOK, output)
}

type TaskOutputModel struct {
	TaskId               int       `json:"taskId"`
	MachineId            int       `json:"machineId"`
	CreatedAt            time.Time `json:"createdAt"`
	IsActive             bool      `json:"isActive"`
	PluginOperatingHours float64   `json:"pluginOperatingHours"`
	TaskName             string    `json:"taskName"`
}
