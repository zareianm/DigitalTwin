package main

import (
	"DigitalTwin/internal/database"
	"DigitalTwin/internal/env"
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "DigitalTwin/docs"

	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq"
	"github.com/robfig/cron/v3"
)

// @title Go Gin Rest API
// @version 1.0
// @description A rest API in Go using Gin framework
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your bearer token in the format **Bearer &lt;token&gt;**

type application struct {
	port      int
	jwtSecret string
	models    database.Models
	cr        *cron.Cron
}

func main() {
	dsn := "postgres://postgres:1234@localhost:5432/DigitalTwinDb?sslmode=disable"

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	baseUploadDir := "D:\\school\\term8\\project\\uploads"

	baseUploadDir = filepath.Clean(baseUploadDir)

	if err := os.MkdirAll(baseUploadDir, 0o755); err != nil {
		log.Fatalf("unable to create/upload dir: %v", err)
	}

	models := database.NewModels(db)
	app := &application{
		port:      env.GetEnvInt("PORT", 8080),
		jwtSecret: env.GetEnvString("JWT_SECRET", "some-secret-123456"),
		models:    models,
		cr:        cron.New(cron.WithSeconds()),
	}

	app.cr.Start()

	app.runMissedTasks(db)

	if err := app.serve(); err != nil {
		log.Fatal(err)
	}

}
