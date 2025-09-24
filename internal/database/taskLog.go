package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
)

type TaskLogModel struct {
	DB *sql.DB
}

type TaskLog struct {
	TaskLogId                    int       `json:"task_log_id"`
	TaskId                       int       `json:"task_id"`
	InputParameterNames          []string  `json:"input_parameter_names"`
	OutputParameterNames         []string  `json:"output_parameter_names"`
	Status                       []bool    `json:"status"`
	CreatedAt                    time.Time `json:"created_at"`
	InputParameterValues         []string  `json:"input_parameter_values"`
	OutputParameterRealValues    []string  `json:"output_parameter_real_values"`
	OutputParameterFromCodeVales []string  `json:"output_parameter_from_code_values"`
}

func (m *TaskLogModel) Insert(taskLog *TaskLog) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `INSERT INTO task_logs(task_id, input_parameter_names, output_parameter_names, status, created_at, input_parameter_values, output_parameter_real_values, output_parameter_from_code_values) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING task_log_id`

	return m.DB.QueryRowContext(ctx, query, taskLog.TaskId, pq.Array(taskLog.InputParameterNames), pq.Array(taskLog.OutputParameterNames), pq.Array(taskLog.Status), taskLog.CreatedAt, pq.Array(taskLog.InputParameterValues), pq.Array(taskLog.OutputParameterRealValues), pq.Array(taskLog.OutputParameterFromCodeVales)).Scan(&taskLog.TaskLogId)
}

// func (m *TaskLogModel) GetAll() ([]*TaskLog, error) {
// 	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
// 	defer cancel()

// 	query := "SELECT * FROM tasks"

// 	rows, err := m.DB.QueryContext(ctx, query)
// 	if err != nil {
// 		return nil, err
// 	}

// 	defer rows.Close()

// 	tasks := []*TaskLog{}

// 	for rows.Next() {
// 		var task TaskLog

// 		err := rows.Scan(&task.TaskId, &task.TimeInterval, &task.CreatedAt, &task.LastRun,
// 			&task.StartTime, &task.EndTime, &task.MachineId, pq.Array(&task.InputParameters),
// 			pq.Array(&task.OutputParameters), pq.Array(&task.OutputParametersErrorRate), &task.FilePath)

// 		if err != nil {
// 			return nil, err
// 		}

// 		tasks = append(tasks, &task)
// 	}

// 	if err = rows.Err(); err != nil {
// 		return nil, err
// 	}

// 	return tasks, nil

// }

// func (m *TaskLogModel) Get(id int) (*TaskLog, error) {
// 	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
// 	defer cancel()

// 	query := "SELECT * FROM task WHERE task_id = $1"

// 	var task TaskLog

// 	err := m.DB.QueryRowContext(ctx, query, id).Scan(&task.TaskId, &task.CreatedAt, &task.EndTime, &task.InputParameters,
// 		&task.LastRun, &task.MachineId, &task.OutputParameters, &task.OutputParametersErrorRate,
// 		&task.StartTime, &task.TaskId, &task.TimeInterval)

// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, nil
// 		}
// 		return nil, err
// 	}

// 	return &task, nil
// }

// func (m *TaskLogModel) Delete(id int) error {
// 	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
// 	defer cancel()

// 	query := "DELETE FROM tasks WHERE task_id = $1"

// 	_, err := m.DB.ExecContext(ctx, query, id)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }
