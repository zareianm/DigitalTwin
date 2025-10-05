package main

import (
	"DigitalTwin/pkg/machineService"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetMachineParameters returns machine parameters
//
//	@Summary		returns machine parameters
//	@Description	returns machine parameters
//	@Tags			machines
//	@Accept			json
//	@Produce		json
//	@Param			machine_id	path		int	true	"machine ID"
//	@Success		200		{object}	[]string
//	@Router			/api/v1/machines/getMachineParameters/{machine_id} [get]
//	@Security		BearerAuth
func (app *application) getMachineParameters(c *gin.Context) {
	machineId, err := strconv.Atoi(c.Param("machine_id"))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid machine ID"})
		return
	}

	accessToken, _ := c.Get("access_token")

	machineData, err := machineService.GetMachineWithData(machineId, accessToken.(string))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fields := make([]string, 0, len(machineData.Json))
	for field := range machineData.Json {
		fields = append(fields, field)
	}

	c.JSON(http.StatusOK, fields)
}
