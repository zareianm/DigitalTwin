package main

import (
	"DigitalTwin/internal/database"
	"net/http"
	"strconv"

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
//	@Success		200		{object}	[]database.Machine
//	@Router			/api/v1/machines [get]
func (app *application) getAllMachines(c *gin.Context) {
	machines, err := app.models.Machines.GetAll()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retreive machines"})
		return
	}

	c.JSON(http.StatusOK, machines)
}

// GetMachine returns a single machine
//
//	@Summary		Returns a single machine
//	@Description	Returns a single machine
//	@Tags			machines
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"Machine ID"
//	@Success		200	{object}	database.Machine
//	@Router			/api/v1/machines/{id} [get]
func (app *application) getMachine(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid machine ID"})
		return
	}

	machine, err := app.models.Machines.Get(id)

	if machine == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Machine not found"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retreive machine"})
		return
	}

	c.JSON(http.StatusOK, machine)
}

// CreateMachine creates a new machine
//
//	@Summary		Creates a new machine
//	@Description	Creates a new machine
//	@Tags			machines
//	@Accept			json
//	@Produce		json
//	@Param			machine	body		database.Machine	true	"Machine"
//	@Success		201		{object}	database.Machine
//	@Router			/api/v1/machines/create [post]
//	@Security		BearerAuth
func (app *application) createMachine(c *gin.Context) {
	var machine database.Machine

	if err := c.ShouldBindJSON(&machine); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := app.models.Machines.Insert(&machine)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create machine"})
		return
	}

	c.JSON(http.StatusCreated, machine)
}

// DeleteMachine deletes an existing machine
//
//	@Summary		Deletes an existing machine
//	@Description	Deletes an existing machine
//	@Tags			machines
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"Machine ID"
//	@Success		204
//	@Router			/api/v1/machines/{id} [delete]
//	@Security		BearerAuth
func (app *application) deleteMachine(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Invalid machine ID"})
		return
	}

	existingMachine, err := app.models.Machines.Get(id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"Error": "Failed to retreive machine"})
		return
	}

	if existingMachine == nil {
		c.JSON(http.StatusNotFound, gin.H{"Error": "Machine not found"})
		return
	}

	if err := app.models.Machines.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete machine"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
