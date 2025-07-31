package main

import (
	"DigitalTwin/internal/database"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
)

// GetUsers returns all users
//
//	@Summary		Returns all users
//	@Description	Returns all users
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Success		200		{object}	[]database.User
//	@Router			/api/v1/users [get]
func (app *application) getAllUsers(c *gin.Context) {
	users, err := app.models.Users.GetAll()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retreive users"})
		return
	}

	c.JSON(http.StatusOK, users)
}

// GetUser returns a single user
//
//	@Summary		Returns a single user
//	@Description	Returns a single user
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"User ID"
//	@Success		200	{object}	database.User
//	@Router			/api/v1/users/{id} [get]
func (app *application) getUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := app.models.Users.Get(id)

	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retreive user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// CreateUser creates a new user
//
//	@Summary		Creates a new user
//	@Description	Creates a new user
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			user	body		database.User	true	"User"
//	@Success		201		{object}	database.User
//	@Router			/api/v1/users/create [post]
//	@Security		BearerAuth
func (app *application) createUser(c *gin.Context) {
	var user database.User

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := app.models.Users.Insert(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// UpdateUser updates an existing user
//
//	@Summary		Updates an existing user
//	@Description	Updates an existing user
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"User ID"
//	@Param			user	body		database.User	true	"User"
//	@Success		200	{object}	database.User
//	@Router			/api/v1/users/update/{id} [put]
//	@Security		BearerAuth
func (app *application) updateUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	existingUser, err := app.models.Users.Get(id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retreive user"})
		return
	}

	if existingUser == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	updatedUser := &database.User{}

	if err := c.ShouldBindJSON(updatedUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedUser.Id = id

	if err := app.models.Users.Update(updatedUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, updatedUser)
}

// DeleteUser deletes an existing user
//
//	@Summary		Deletes an existing user
//	@Description	Deletes an existing user
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"User ID"
//	@Success		204
//	@Router			/api/v1/users/{id} [delete]
//	@Security		BearerAuth
func (app *application) deleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Invalid user ID"})
		return
	}

	existingUser, err := app.models.Users.Get(id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"Error": "Failed to retreive user"})
		return
	}

	if existingUser == nil {
		c.JSON(http.StatusNotFound, gin.H{"Error": "User not found"})
		return
	}

	if err := app.models.Users.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
