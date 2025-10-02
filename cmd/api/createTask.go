package main

import (
	"DigitalTwin/internal/database"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"DigitalTwin/pkg/cppService"
	"DigitalTwin/pkg/fileService"
	"DigitalTwin/pkg/pythonService"
	"DigitalTwin/pkg/taskService"

	"github.com/gin-gonic/gin"
)

type SaveTaskResult struct {
	TaskId int    `json:"taskId"`
	Error  string `json:"errors"`
}

// CreateTask creates a new task
//
//		@Summary		Creates a new task
//		@Description	Creates a new task
//		@Tags			tasks
//	    @Accept       	multipart/form-data
//		@Produce		json
//	    @Param        	file  		formData  	file	true	"C++ or Python source file to scan"
//		@Param        	machineId  	formData  	int 	true 	"ID of the machine"
//		@Param        	intervalTimeInMinutes     formData  	int   	true  	"interval time in minutes"
//		@Param        	inputParameters     formData  	[]string   	true  	"input parmas"
//		@Param        	outputParameters     formData  	[]string   	true  	"output parmas"
//		@Param        	outputParametersErrorRate     formData  	[]int  	 true  	"output parmas error rates"
//		@Param    		startTime   formData  string  true  "start time in RFC3339 format (e.g. 2025-08-18T14:30:00Z)" format(date-time)
//		@Param    		endTime     formData  string  true  "start time in RFC3339 format (e.g. 2025-08-18T14:30:00Z)" format(date-time)
//		@Success		201			{object}	SaveTaskResult
//		@Router			/api/v1/tasks/create [post]
func (app *application) createTask(c *gin.Context) {

	// 1) get file
	f, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file parameter required"})
		return
	}
	defer f.Close()

	machineId, err := strconv.Atoi(c.PostForm("machineId"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid machineId"})
		return
	}

	intervalTimeInMinutes, err := strconv.Atoi(c.PostForm("intervalTimeInMinutes"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid minutes"})
		return
	}

	intervalTime, err := GenerateCronSpec(intervalTimeInMinutes)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid minutes"})
		return
	}

	startTimeStr := c.PostForm("startTime")
	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid startTime, must be RFC3339"})
		return
	}

	endTimeStr := c.PostForm("endTime")
	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid endTime, must be RFC3339"})
		return
	}

	inputParams := c.PostFormArray("inputParameters")
	outputParams := c.PostFormArray("outputParameters")

	var outputErrorRates []int64
	for _, val := range c.PostFormArray("outputParametersErrorRate") {
		num, err := strconv.Atoi(val)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid outputParametersErrorRate"})
			return
		}
		outputErrorRates = append(outputErrorRates, int64(num))
	}

	machine, err := app.models.Machines.Get(machineId)
	if err != nil {
		c.JSON(404, gin.H{"error": "invalid machineId"})
		return
	}

	args, err := app.models.Machines.GetParameterValuesFromMachine(*machine, inputParams)
	if err != nil {
		c.JSON(404, gin.H{"error": "invalid inputParameters"})
		return
	}

	filepath, err := fileService.SaveFile(header)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var stdOut, errorStr string

	fileExtension := taskService.GetFileExtension(filepath)

	switch fileExtension {

	case "cpp":
		{
			stdOut, errorStr = cppService.CheckCppError(filepath, args)
		}
	case "py":
		{
			stdOut, errorStr = pythonService.CheckPythonError(filepath, args)
		}
	default:
		{
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file extension"})
		}
	}

	if errorStr != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errorStr})
		return
	}

	_, err = app.models.Machines.GetOutputResultsFromCodeResult(stdOut, outputParams)
	if err != nil {
		c.JSON(404, gin.H{"error": "invalid outputParameters"})
		return
	}

	var task database.Task = database.Task{
		MachineId:                 machineId,
		TimeInterval:              intervalTime,
		CreatedAt:                 time.Now().UTC(),
		StartTime:                 startTime,
		EndTime:                   endTime,
		InputParameters:           inputParams,
		OutputParameters:          outputParams,
		OutputParametersErrorRate: outputErrorRates,
		FilePath:                  filepath,
	}

	err = app.models.Tasks.Insert(&task)

	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return

	}
	taskService.RegisterTask(task, app.cr, app.models)

	result := SaveTaskResult{
		Error:  "",
		TaskId: task.TaskId,
	}

	c.JSON(http.StatusOK, result)
}

func GenerateCronSpec(totalMinutes int) (string, error) {
	if totalMinutes <= 0 {
		return "", errors.New("invalid value for minutes")
	}

	hours := totalMinutes / 60
	minutes := totalMinutes % 60

	if hours == 0 {
		return fmt.Sprintf("0 */%d * * * *", minutes), nil
	}
	cronSpec := fmt.Sprintf("0 %d */%d * * *", minutes, hours)
	return cronSpec, nil
}
