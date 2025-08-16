package database

import (
	"context"
	"database/sql"
	"time"
)

type MachineModel struct {
	DB *sql.DB
}

type Machine struct {
	MachineId        int    `json:"machine_id"`
	Name             string `json:"name"`
	InputParameters  string `json:"input_parameters"`
	OutputParameters string `json:"output_parameters"`
}

func (m *MachineModel) Insert(Machine *Machine) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := "INSERT INTO machines (name, input_parameters, output_parameters) VALUES ($1, $2, $3) RETURNING machine_id"

	return m.DB.QueryRowContext(ctx, query, Machine.Name, Machine.InputParameters, Machine.OutputParameters).Scan(&Machine.MachineId)
}

func (m *MachineModel) GetAll() ([]*Machine, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := "SELECT * FROM machines"

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	Machines := []*Machine{}

	for rows.Next() {
		var Machine Machine

		err := rows.Scan(&Machine.MachineId, &Machine.Name, &Machine.InputParameters, &Machine.OutputParameters)
		if err != nil {
			return nil, err
		}

		Machines = append(Machines, &Machine)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return Machines, nil

}

func (m *MachineModel) Get(id int) (*Machine, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := "SELECT * FROM machines WHERE machine_id = $1"

	var Machine Machine

	err := m.DB.QueryRowContext(ctx, query, id).Scan(&Machine.MachineId, &Machine.Name, &Machine.InputParameters, &Machine.OutputParameters)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &Machine, nil
}

func (m *MachineModel) Delete(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := "DELETE FROM machines WHERE id = $1"

	_, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	return nil
}
