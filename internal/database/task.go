package database

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/lib/pq"
)

type TaskModel struct {
	DB *sql.DB
}

type Task struct {
	TaskId                    int        `json:"task_id"`
	DeviceId                  int        `json:"device_id"`
	TimeInterval              string     `json:"time_interval"`
	CreatedAt                 time.Time  `json:"created_at"`
	LastRun                   *time.Time `json:"last_run"`
	StartTime                 time.Time  `json:"start_time"`
	EndTime                   *time.Time `json:"end_time"`
	InputParameters           []string   `json:"input_parameters"`
	OutputParameters          []string   `json:"output_parameters"`
	AcceptableErrorPercentage []int64    `json:"acceptable_error_percentage"`
	FilePath                  string     `json:"file_path"`
	TaskName                  string     `json:"task_name"`
	UserId                    int        `json:"user_id"`
	AccessToken               string     `json:"access_token"`
	DeviceName                string     `json:"device_name"`
}

func (m *TaskModel) Insert(task *Task) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `INSERT INTO tasks(device_id, time_interval, created_at, start_time, end_time, input_parameters, output_parameters, acceptable_error_percentage, file_path, task_name, user_id, access_token, device_name) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) RETURNING task_id`

	return m.DB.QueryRowContext(ctx, query, task.DeviceId, task.TimeInterval, task.CreatedAt, task.StartTime, task.EndTime, pq.Array(task.InputParameters), pq.Array(task.OutputParameters), pq.Array(task.AcceptableErrorPercentage), task.FilePath, task.TaskName, task.UserId, task.AccessToken, task.DeviceName).Scan(&task.TaskId)
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

func (m *TaskModel) GetAllUserTask(userId int) ([]*Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := "SELECT * FROM tasks WHERE user_id = " + strconv.Itoa(userId)

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	tasks := []*Task{}

	for rows.Next() {
		var task Task

		err := rows.Scan(&task.TaskId, &task.TimeInterval, &task.CreatedAt, &task.LastRun,
			&task.StartTime, &task.EndTime, &task.DeviceId, pq.Array(&task.InputParameters),
			pq.Array(&task.OutputParameters), pq.Array(&task.AcceptableErrorPercentage),
			&task.FilePath, &task.TaskName, &task.UserId, &task.AccessToken, &task.DeviceName)

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

func (m *TaskModel) Get(userId int, taskId int) (*Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := "SELECT * FROM tasks WHERE user_id = $1 And task_id = $2"

	var task Task

	err := m.DB.QueryRowContext(ctx, query, userId, taskId).Scan(&task.TaskId, &task.TimeInterval, &task.CreatedAt,
		&task.LastRun, &task.StartTime, &task.EndTime, &task.DeviceId,
		pq.Array(&task.InputParameters), pq.Array(&task.OutputParameters),
		pq.Array(&task.AcceptableErrorPercentage), &task.FilePath, &task.TaskName, &task.UserId, &task.AccessToken, &task.DeviceName)

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
			&task.StartTime, &task.EndTime, &task.DeviceId, pq.Array(&task.InputParameters),
			pq.Array(&task.OutputParameters), pq.Array(&task.AcceptableErrorPercentage),
			&task.FilePath, &task.TaskName, &task.UserId, &task.AccessToken, &task.DeviceName)

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
