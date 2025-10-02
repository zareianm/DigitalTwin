package main

import (
	runMigration "DigitalTwin/cmd/migrate"
	"DigitalTwin/internal/database"
	"DigitalTwin/internal/env"
	"DigitalTwin/pkg/taskService"
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "DigitalTwin/docs"

	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq"
	"github.com/robfig/cron/v3"
)

// @title DigitalTwin API
// @version 1.0

type application struct {
	port   int
	models database.Models
	cr     *cron.Cron
}

func main() {

	dsn := env.GetEnvString("databaseconnectionstring", "")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	runMigration.UpdateDatabase(db, "up")

	baseUploadDir := env.GetEnvString("baseUploadDir", "")

	baseUploadDir = filepath.Clean(baseUploadDir)

	if err := os.MkdirAll(baseUploadDir, 0o755); err != nil {
		log.Fatalf("unable to create/upload dir: %v", err)
	}

	models := database.NewModels(db)
	app := &application{
		port:   env.GetEnvInt("PORT", 8080),
		models: models,
		cr:     cron.New(cron.WithSeconds()),
	}

	app.cr.Start()

	taskService.RunMissedTasks(app.cr, app.models)

	if err := app.serve(); err != nil {
		log.Fatal(err)
	}

}
