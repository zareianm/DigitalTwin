package main

import (
	"DigitalTwin/internal/database"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"DigitalTwin/pkg/cppService"
	"DigitalTwin/pkg/deviceService"
	"DigitalTwin/pkg/fileService"
	"DigitalTwin/pkg/javaService"
	"DigitalTwin/pkg/jsService"
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
//			@Summary		Creates a new task
//			@Description	Creates a new task
//			@Tags			tasks
//		    @Accept       	multipart/form-data
//			@Produce		json
//		    @Param        	file  					  formData  	file		true	"C++ or Python or Java or Javascript source file to scan"
//			@Param        	taskName  				  formData  	string 		true 	"Name of task"
//			@Param        	deviceId  				  formData  	int 		true 	"ID of the device"
//			@Param        	intervalTimeInMinutes     formData  	int   		true  	"interval time in minutes"
//			@Param        	inputParameters     	  formData  	[]string   	true  	"input parmas" collectionFormat(multi)
//			@Param        	outputParameters          formData  	[]string   	true  	"output parmas" collectionFormat(multi)
//			@Param        	acceptableErrorPercentage formData  	[]int  	 	true  	"output parmas error percentage" collectionFormat(multi)
//			@Param    		startTime   			  formData      string  	true    "start time in UTC and RFC3339 format (e.g. 2025-08-18T14:30:00Z)" format(date-time)
//			@Param    		endTime     			  formData  	string      false    "end time in UTC and RFC3339 format (e.g. 2025-08-18T14:30:00Z)" format(date-time)
//			@Success		201			{object}	SaveTaskResult
//			@Router			/api/v1/tasks/create [post]
//	        @Security		BearerAuth
func (app *application) createTask(c *gin.Context) {

	userIdStr, _ := c.Get("user_id")
	userId, _ := userIdStr.(int)

	accessToken, _ := c.Get("access_token")

	// 1) get file
	f, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file parameter required"})
		return
	}
	defer f.Close()

	deviceId, err := strconv.Atoi(c.PostForm("deviceId"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid deviceId"})
		return
	}

	taskName := c.PostForm("taskName")

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
	var endTime *time.Time

	endTimeTemp, err := time.Parse(time.RFC3339, endTimeStr)

	if err != nil {
		endTime = nil
	} else {
		endTime = &endTimeTemp
	}

	if startTime.Before(time.Now().UTC()) {
		c.JSON(400, gin.H{"error": "invalid start time, must be in the future"})
		return
	}

	if endTime != nil && endTime.Before(startTime) {
		c.JSON(400, gin.H{"error": "invalid end time, must be greater than start time"})
		return
	}

	inputParams := c.PostFormArray("inputParameters")
	outputParams := c.PostFormArray("outputParameters")

	var acceptableErrorPercentage []int64
	for _, val := range c.PostFormArray("acceptableErrorPercentage") {
		num, err := strconv.Atoi(val)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid acceptableErrorPercentage"})
			return
		}
		acceptableErrorPercentage = append(acceptableErrorPercentage, int64(num))
	}

	device, err := deviceService.GetDeviceWithData(deviceId, accessToken.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retreive device"})
		return
	}

	if device == nil {
		c.JSON(404, gin.H{"error": "invalid deviceId"})
		return
	}

	args, err := deviceService.GetParameterValuesFromDevice(*device, inputParams)
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
	case "java":
		{
			stdOut, errorStr = javaService.CheckJavaError(filepath, args)
		}
	case "js":
		{
			stdOut, errorStr = jsService.CheckJsError(filepath, args)
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

	_, err = deviceService.GetOutputResultsFromCodeResult(stdOut, outputParams)
	if err != nil {
		c.JSON(404, gin.H{"error": "invalid outputParameters"})
		return
	}

	var task database.Task = database.Task{
		DeviceId:                  deviceId,
		TimeInterval:              intervalTime,
		CreatedAt:                 time.Now().UTC(),
		StartTime:                 startTime,
		EndTime:                   endTime,
		InputParameters:           inputParams,
		OutputParameters:          outputParams,
		AcceptableErrorPercentage: acceptableErrorPercentage,
		FilePath:                  filepath,
		TaskName:                  taskName,
		UserId:                    userId,
		AccessToken:               accessToken.(string),
		DeviceName:                device.DeviceName,
	}

	err = app.models.Tasks.Insert(&task)

	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return

	}

	cronId := taskService.RegisterTask(task, app.cr, app.models)
	app.models.Tasks.UpdateTaskCronId(&task, cronId)

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
