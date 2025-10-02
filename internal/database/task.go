package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
)

type TaskModel struct {
	DB *sql.DB
}

type Task struct {
	TaskId                    int        `json:"task_id"`
	MachineId                 int        `json:"machine_id"`
	TimeInterval              string     `json:"time_interval"`
	CreatedAt                 time.Time  `json:"created_at"`
	LastRun                   *time.Time `json:"last_run"`
	StartTime                 time.Time  `json:"start_time"`
	EndTime                   time.Time  `json:"end_time"`
	InputParameters           []string   `json:"input_parameters"`
	OutputParameters          []string   `json:"output_parameters"`
	OutputParametersErrorRate []int64    `json:"output_parameters_error_rate"`
	FilePath                  string     `json:"file_path"`
	TaskName                  string     `json:"task_name"`
}

func (m *TaskModel) Insert(task *Task) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `INSERT INTO tasks(machine_id, time_interval, created_at, start_time, end_time, input_parameters, output_parameters, output_parameters_error_rate, file_path, task_name) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING task_id`

	return m.DB.QueryRowContext(ctx, query, task.MachineId, task.TimeInterval, task.CreatedAt, task.StartTime, task.EndTime, pq.Array(task.InputParameters), pq.Array(task.OutputParameters), pq.Array(task.OutputParametersErrorRate), task.FilePath, task.TaskName).Scan(&task.TaskId)
}

func (m *TaskModel) UpdateLastExecute(task *Task) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := "UPDATE tasks SET last_run = $1 WHERE task_id = $2"

	_, err := m.DB.ExecContext(ctx, query, time.Now().UTC(), task.TaskId)

	if err != nil {
		return err
	}

	return nil
}

func (m *TaskModel) GetAll() ([]*Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := "SELECT * FROM tasks"

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	tasks := []*Task{}

	for rows.Next() {
		var task Task

		err := rows.Scan(&task.TaskId, &task.TimeInterval, &task.CreatedAt, &task.LastRun,
			&task.StartTime, &task.EndTime, &task.MachineId, pq.Array(&task.InputParameters),
			pq.Array(&task.OutputParameters), pq.Array(&task.OutputParametersErrorRate), &task.FilePath, &task.TaskName)

		if err != nil {
			return nil, err
		}

		tasks = append(tasks, &task)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil

}

func (m *TaskModel) Get(id int) (*Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := "SELECT * FROM tasks WHERE task_id = $1"

	var task Task

	err := m.DB.QueryRowContext(ctx, query, id).Scan(&task.TaskId, &task.TimeInterval, &task.CreatedAt,
		&task.LastRun, &task.StartTime, &task.EndTime, &task.MachineId,
		pq.Array(&task.InputParameters), pq.Array(&task.OutputParameters),
		pq.Array(&task.OutputParametersErrorRate), &task.FilePath, &task.TaskName)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &task, nil
}

func (m *TaskModel) Delete(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := "DELETE FROM tasks WHERE task_id = $1"

	_, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	return nil
}
