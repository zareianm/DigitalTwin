package database

import (
	"context"
	"database/sql"
	"log"
	"time"
)

type TaskModel struct {
	DB *sql.DB
}

type Task struct {
	ID       int64     `json:"id"`
	Name     string    `json:"name"`
	CronSpec string    `json:"cron_spec"`
	Payload  string    `json:"payload"`
	LastRun  time.Time `json:"last_run"`
}

func (m *TaskModel) Insert(task *Task) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `INSERT INTO tasks(name, cron_spec, payload) VALUES ($1, $2, $3) RETURNING id`

	return m.DB.QueryRowContext(ctx, query, task.Name, task.CronSpec, task.Payload).Scan(&task.ID)
}

func (m *TaskModel) UpdateLastExecute(task *Task) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := "UPDATE tasks SET last_run = $1 WHERE id = $2"

	_, err := m.DB.ExecContext(ctx, query, time.Now().Unix(), task.ID)

	if err != nil {
		log.Printf("update last_run for task %d: %v", task.ID, err)

		return err
	}

	return nil
}
