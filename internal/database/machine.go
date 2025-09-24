package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"
)

type MachineModel struct {
	DB *sql.DB
}

type Machine struct {
	MachineId  int    `json:"machine_id"`
	Name       string `json:"name"`
	Parameters string `json:"parameters"`
}

func (m *MachineModel) Insert(Machine *Machine) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := "INSERT INTO machines (name, parameters) VALUES ($1, $2) RETURNING machine_id"

	return m.DB.QueryRowContext(ctx, query, Machine.Name, Machine.Parameters).Scan(&Machine.MachineId)
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

		err := rows.Scan(&Machine.MachineId, &Machine.Name, &Machine.Parameters)
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

	err := m.DB.QueryRowContext(ctx, query, id).Scan(&Machine.MachineId, &Machine.Name, &Machine.Parameters)
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

func (m *MachineModel) GetParameterValuesFromMachine(machine Machine, neededParameters []string) ([]string, error) {
	paramsString := machine.Parameters

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(paramsString), &data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	results := make([]string, 0, len(neededParameters))

	for _, key := range neededParameters {
		val, exists := data[key]
		if !exists {
			return nil, errors.New("key not found: " + key)
		}
		results = append(results, fmt.Sprintf("%v", val))
	}

	return results, nil
}

func (m *MachineModel) GetOutputResultsFromCodeResult(outputResult string, outputParams []string) ([]string, error) {
	results := make([]string, 0, len(outputParams))

	for _, key := range outputParams {
		// Regex: match key=VALUE where VALUE is non-space, non-comma, non-dot
		re := regexp.MustCompile(fmt.Sprintf(`\b%s\s*=\s*([^,\s\.]+)`, regexp.QuoteMeta(key)))
		match := re.FindStringSubmatch(outputResult)
		if len(match) < 2 {
			return nil, errors.New("key not found: " + key)
		}
		results = append(results, match[1])
	}

	return results, nil
}
