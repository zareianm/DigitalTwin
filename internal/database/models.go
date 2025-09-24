package database

import "database/sql"

type Models struct {
	Users    UserModel
	Tasks    TaskModel
	Machines MachineModel
	TaskLogs TaskLogModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		Users:    UserModel{DB: db},
		Tasks:    TaskModel{DB: db},
		Machines: MachineModel{DB: db},
		TaskLogs: TaskLogModel{DB: db},
	}
}
