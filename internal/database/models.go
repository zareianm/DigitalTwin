package database

import "database/sql"

type Models struct {
	Tasks    TaskModel
	TaskLogs TaskLogModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		Tasks:    TaskModel{DB: db},
		TaskLogs: TaskLogModel{DB: db},
	}
}
