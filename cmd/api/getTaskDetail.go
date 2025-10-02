package main

import (
	"DigitalTwin/internal/database"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GetTaskDetail returns the details of a task
//
//	@Summary		returns the details of a task
//	@Description	returns the details of a task
//	@Tags			tasks
//	@Accept			json
//	@Produce		json
//	@Param			task_id	path		int	true	"Task ID"
//	@Success		200	{object} TaskDetailOutputModel
//	@Router			/api/v1/tasks/GetTaskDetail/{task_id} [get]
func (app *application) getTaskDetail(c *gin.Context) {
	taskId, err := strconv.Atoi(c.Param("task_id"))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	task, err := app.models.Tasks.Get(taskId)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retreive task details"})
		return
	}

	if task == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	taskLogs, err := app.models.TaskLogs.GetTaskLogsWithTaskId(taskId)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retreive task details"})
		return
	}

	result := TaskDetailOutputModel{
		TaskId:               task.TaskId,
		MachineId:            task.MachineId,
		CreatedAt:            task.CreatedAt,
		IsActive:             task.StartTime.Before(time.Now()) && task.EndTime.After(time.Now()),
		PluginOperatingHours: getTaskOperatingHour(*task),
		Data:                 []TaskLog{},
		MaximumErrorRates:    []MaximumErrorRate{},
		TaskName:             task.TaskName,
	}

	getTaskLogs(taskLogs, &result)

	getParameterMaximumErrorRate(*task, &result)

	calculateSystemErrorPercentage(&result)

	c.JSON(http.StatusOK, result)
}

func getTaskLogs(taskLogs []*database.TaskLog, result *TaskDetailOutputModel) {

	for _, taskLog := range taskLogs {
		outputLog := TaskLog{
			RunTime:          taskLog.CreatedAt,
			InputParameters:  []InputParameter{},
			OutputParameters: []OutputParameter{},
		}

		for i, inputParamName := range taskLog.InputParameterNames {
			inputParam := InputParameter{
				ParameterName:  inputParamName,
				ParameterValue: taskLog.InputParameterValues[i],
			}

			outputLog.InputParameters = append(outputLog.InputParameters, inputParam)
		}

		for i, outputParameterName := range taskLog.OutputParameterNames {
			outPutParam := OutputParameter{
				ParameterName:         outputParameterName,
				ParameterMachineValue: taskLog.OutputParameterRealValues[i],
				ParameterCodeValue:    taskLog.OutputParameterFromCodeVales[i],
				Status:                taskLog.Status[i],
			}

			outputLog.OutputParameters = append(outputLog.OutputParameters, outPutParam)
		}

		result.Data = append(result.Data, outputLog)
	}
}

func getTaskOperatingHour(t database.Task) float64 {
	now := time.Now().UTC()

	var operatingHours float64
	if now.After(t.EndTime) {
		operatingHours = t.EndTime.Sub(t.StartTime).Hours()
	} else {
		operatingHours = now.Sub(t.StartTime).Hours()
	}

	if operatingHours < 0 {
		return 0
	}

	return operatingHours
}

func getParameterMaximumErrorRate(task database.Task, result *TaskDetailOutputModel) {
	for i, errorRate := range task.OutputParametersErrorRate {

		maximumErrorRate := MaximumErrorRate{
			ParameterName: task.OutputParameters[i],
			ErrorRate:     errorRate,
		}

		result.MaximumErrorRates = append(result.MaximumErrorRates, maximumErrorRate)
	}
}

func calculateSystemErrorPercentage(result *TaskDetailOutputModel) {
	var sum float64
	var n int

	for _, taskLog := range result.Data {

		for _, outputParameter := range taskLog.OutputParameters {

			expected, _ := strconv.ParseFloat(outputParameter.ParameterCodeValue, 64)

			if expected == 0 {
				continue
			}

			realValue, _ := strconv.ParseFloat(outputParameter.ParameterMachineValue, 64)

			val := math.Abs(realValue-expected) / expected * 100
			sum += val
			n++
		}
	}
	if n == 0 {
		result.SystemErrorPercentage = 0
	} else {

		result.SystemErrorPercentage = sum / float64(n)
	}
}

type TaskDetailOutputModel struct {
	TaskId                int                `json:"taskId"`
	MachineId             int                `json:"machineId"`
	CreatedAt             time.Time          `json:"createdAt"`
	IsActive              bool               `json:"isActive"`
	PluginOperatingHours  float64            `json:"pluginOperatingHours"`
	Data                  []TaskLog          `json:"data"`
	MaximumErrorRates     []MaximumErrorRate `json:"maximumErrorRates"`
	SystemErrorPercentage float64            `json:"systemErrorPercentage"`
	TaskName              string             `json:"taskName"`
}

type TaskLog struct {
	RunTime          time.Time         `json:"runTime"`
	InputParameters  []InputParameter  `json:"inputParameters"`
	OutputParameters []OutputParameter `json:"outputParameters"`
}

type InputParameter struct {
	ParameterName  string `json:"parameterName"`
	ParameterValue string `json:"parameterValue"`
}

type OutputParameter struct {
	ParameterName         string `json:"parameterName"`
	ParameterMachineValue string `json:"parameterMachineValue"`
	ParameterCodeValue    string `json:"parameterCodeValue"`
	Status                bool   `json:"status"`
}

type MaximumErrorRate struct {
	ParameterName string `json:"parameterName"`
	ErrorRate     int64  `json:"errorRate"`
}
