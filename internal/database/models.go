package database

import "database/sql"

type Models struct {
	Tasks    TaskModel
	Machines MachineModel
	TaskLogs TaskLogModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		Tasks:    TaskModel{DB: db},
		Machines: MachineModel{DB: db},
		TaskLogs: TaskLogModel{DB: db},
	}
}
