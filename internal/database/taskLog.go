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

func (m *TaskLogModel) GetTaskLogsWithTaskId(taskId int) ([]*TaskLog, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := "SELECT * FROM task_logs WHERE task_id = $1"

	rows, err := m.DB.QueryContext(ctx, query, taskId)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	taskLogs := []*TaskLog{}

	for rows.Next() {
		var taskLog TaskLog

		err := rows.Scan(&taskLog.TaskLogId, &taskLog.TaskId, pq.Array(&taskLog.InputParameterNames), pq.Array(&taskLog.OutputParameterNames),
			pq.Array(&taskLog.Status), &taskLog.CreatedAt, pq.Array(&taskLog.InputParameterValues),
			pq.Array(&taskLog.OutputParameterRealValues), pq.Array(&taskLog.OutputParameterFromCodeVales))

		if err != nil {
			return nil, err
		}

		taskLogs = append(taskLogs, &taskLog)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return taskLogs, nil
}
