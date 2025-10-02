package taskService

import (
	"DigitalTwin/internal/database"
	"DigitalTwin/pkg/cppService"
	"DigitalTwin/pkg/javaService"
	"DigitalTwin/pkg/pythonService"
	"log"
	"math"
	"path/filepath"
	"strconv"
	"time"

	"github.com/robfig/cron/v3"
)

func RunMissedTasks(cr *cron.Cron, models database.Models) {
	// load previously persisted tasks
	rows, err := models.Tasks.GetAll()

	if err != nil {
		log.Fatalf("query tasks: %v", err)
	}

	for i := 0; i < len(rows); i++ {
		RegisterTask(*rows[i], cr, models)
	}
}

func RegisterTask(t database.Task, cr *cron.Cron, models database.Models) {
	// capture by value so each closure has its own copy
	_, err := cr.AddFunc(t.TimeInterval, func() { RunTask(t, models) })
	if err != nil {
		log.Printf("failed to register task %d: %v", t.TaskId, err)
	}
}

func RunTask(t database.Task, models database.Models) {

	now := time.Now().UTC()

	if now.Before(t.StartTime) || now.After(t.EndTime) {
		return
	}

	machine, err := models.Machines.Get(t.MachineId)
	if err != nil {
		return
	}

	args, err := models.Machines.GetParameterValuesFromMachine(*machine, t.InputParameters)
	if err != nil {
		return
	}

	var stdOut string

	fileExtension := GetFileExtension(t.FilePath)

	switch fileExtension {

	case "cpp":
		{
			stdOut, _, err = cppService.RunCppInDocker(t.FilePath, args)
		}
	case "py":
		{
			stdOut, _, err = pythonService.RunPythonInDocker(t.FilePath, args)
		}
	case "java":
		{
			stdOut, _, err = javaService.RunJavaInDocker(t.FilePath, args)
		}
	default:
		{
			return
		}
	}

	if err != nil {
		return
	}

	resultsFromCode, err := models.Machines.GetOutputResultsFromCodeResult(stdOut, t.OutputParameters)
	if err != nil {
		return
	}

	realOutputResult, _ := models.Machines.GetParameterValuesFromMachine(*machine, t.OutputParameters)

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

	err = models.TaskLogs.Insert(&taskLog)

	if err != nil {
		return
	}
	models.Tasks.UpdateLastExecute(&t)
}

func isSafe(expectedValue, realValue float64, errorRateInPercent int64) bool {
	if expectedValue == 0 {
		return realValue != 0
	}

	diff := math.Abs(realValue-expectedValue) / expectedValue * 100

	return diff <= float64(errorRateInPercent)
}

func GetFileExtension(filePath string) string {
	ext := filepath.Ext(filePath)
	if len(ext) > 0 {
		return ext[1:]
	}
	return ""
}
