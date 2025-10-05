package main

import (
	"DigitalTwin/pkg/machineService"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
)

// GetMachines returns all machines
//
//	@Summary		Returns all machines
//	@Description	Returns all machines
//	@Tags			machines
//	@Accept			json
//	@Produce		json
//	@Success		200		{object}	[]MachineListOutputModel
//	@Router			/api/v1/machines [get]
//	@Security		BearerAuth
func (app *application) getAllMachines(c *gin.Context) {
	machines, err := machineService.GetAllMachines()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retreive machines"})
		return
	}

	var outputMachines []MachineListOutputModel
	for _, d := range machines {
		outputMachines = append(outputMachines, MachineListOutputModel{
			MachineID:   d.MachineId,
			MachineName: d.MachineName,
		})
	}

	c.JSON(http.StatusOK, outputMachines)
}

type MachineListOutputModel struct {
	MachineID   int    `json:"machineId"`
	MachineName string `json:"machineName"`
}
