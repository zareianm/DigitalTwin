package main

import (
	auth "DigitalTwin/cmd/api/authMiddleware"
	"DigitalTwin/internal/env"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func (app *application) routes() http.Handler {
	g := gin.Default()

	config := cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	g.Use(cors.New(config))

	v1 := g.Group("/api/v1", auth.AuthMiddleware())
	{
		v1.GET("/devices", app.getAllDevices)
		v1.GET("/devices/getDeviceParameters/:device_id", app.getDeviceParameters)

		v1.POST("/tasks/create", app.createTask)
		v1.GET("/tasks/getTaskList", app.getAllTasks)
		v1.GET("/tasks/GetTaskDetail/:task_id", app.getTaskDetail)
	}

	g.GET("/swagger/*any", func(c *gin.Context) {
		if c.Request.RequestURI == "/swagger/" {
			c.Redirect(302, "/swagger/index.html")
			return
		}
		baseURL := env.GetEnvString("BASE_URL", "http://localhost:8080")
		ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL(baseURL+"/swagger/doc.json"))(c)
	})

	return g
}
