package main

import (
	"DigitalTwin/internal/database"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"DigitalTwin/pkg/jobService"

	"github.com/gin-gonic/gin"
)

type SaveJobResult struct {
	TaskId  int    `json:"taskId"`
	Error   string `json:"errors"`
	Success bool   `json:"success"`
}

// CreateJob creates a new job
//
//		@Summary		Creates a new job
//		@Description	Creates a new job
//		@Tags			jobs
//	    @Accept       	multipart/form-data
//		@Produce		json
//	    @Param        	file  		formData  	file	true	"C++ source file to scan"
//		@Param        	machineId  	formData  	int 	true 	"ID of the machine"
//		@Param        	intervalTimeInMinutes     formData  	int   	true  	"interval time in minutes"
//		@Param        	inputParameters     formData  	[]string   	true  	"input parmas"
//		@Param        	outputParameters     formData  	[]string   	true  	"output parmas"
//		@Param        	outputParametersErrorRate     formData  	[]int  	 true  	"output parmas error rates"
//		@Param    		startTime   formData  string  true  "start time in RFC3339 format (e.g. 2025-08-18T14:30:00Z)" format(date-time)
//		@Param    		endTime     formData  string  true  "start time in RFC3339 format (e.g. 2025-08-18T14:30:00Z)" format(date-time)
//		@Success		201			{object}	SaveJobResult
//		@Router			/api/v1/jobs/create [post]
func (app *application) createJob(c *gin.Context) {

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

	filepath, err := jobService.SaveFile(header)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	stdOut, errorStr := jobService.CheckCppError(filepath, args)

	if errorStr != "" {
		result := SaveJobResult{
			Error:   errorStr,
			Success: false,
		}
		c.JSON(http.StatusOK, result)
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
		result := SaveJobResult{
			Error:   err.Error(),
			Success: true,
			TaskId:  task.TaskId,
		}
		c.JSON(http.StatusOK, result)
		return

	}
	app.registerTask(task)

	result := SaveJobResult{
		Error:   "",
		Success: true,
		TaskId:  task.TaskId,
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

func (app *application) runMissedTasks() {
	// load previously persisted tasks
	rows, err := app.models.Tasks.GetAll()

	if err != nil {
		log.Fatalf("query tasks: %v", err)
	}

	for i := 0; i < len(rows); i++ {
		app.registerTask(*rows[i])
	}
}

func (app *application) registerTask(t database.Task) {
	// capture by value so each closure has its own copy
	_, err := app.cr.AddFunc(t.TimeInterval, func() { app.runJob(t) })
	if err != nil {
		log.Printf("failed to register task %d: %v", t.TaskId, err)
	}
}

func (app *application) runJob(t database.Task) {

	now := time.Now().UTC()

	if now.Before(t.StartTime) || now.After(t.EndTime) {
		return
	}

	machine, err := app.models.Machines.Get(t.MachineId)
	if err != nil {
		return
	}

	args, err := app.models.Machines.GetParameterValuesFromMachine(*machine, t.InputParameters)
	if err != nil {
		return
	}

	stdOut, _, err := jobService.RunCppInDocker(t.FilePath, args)
	if err != nil {
		return
	}

	resultsFromCode, err := app.models.Machines.GetOutputResultsFromCodeResult(stdOut, t.OutputParameters)
	if err != nil {
		return
	}

	realOutputResult, _ := app.models.Machines.GetParameterValuesFromMachine(*machine, t.OutputParameters)

	var taskLog database.TaskLog = database.TaskLog{
		TaskId:                       t.TaskId,
		InputParameterNames:          t.InputParameters,
		OutputParameterNames:         t.OutputParameters,
		CreatedAt:                    time.Now().UTC(),
		Status:                       make([]bool, len(t.OutputParameters)),
		OutputParameterRealValues:    realOutputResult,
		InputParameterValues:         args,
		OutputParameterFromCodeVales: resultsFromCode,
	}

	for i, result := range resultsFromCode {
		expectedValue, _ := strconv.ParseFloat(result, 64)
		realResultValue, _ := strconv.ParseFloat(realOutputResult[i], 64)
		errorRate := t.OutputParametersErrorRate[i]

		taskLog.Status[i] = isSafe(expectedValue, realResultValue, errorRate)
	}

	err = app.models.TaskLogs.Insert(&taskLog)

	if err != nil {
		return
	}
	app.models.Tasks.UpdateLastExecute(&t)
}

func isSafe(expectedValue, realValue float64, errorRateInPercent int64) bool {
	if expectedValue == 0 {
		return realValue != 0
	}

	diff := math.Abs(realValue-expectedValue) / expectedValue * 100

	return diff <= float64(errorRateInPercent)
}
