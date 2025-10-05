package machineService

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

type DeviceApiModel struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type MachineListModel struct {
	MachineId   int
	MachineName string
}

func GetAllMachines() ([]MachineListModel, error) {
	url := "https://api.metable.ir/api/device_list/"
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0b2tlbl90eXBlIjoiYWNjZXNzIiwiZXhwIjoxNzYwNTE5NDg2LCJpYXQiOjE3NTk2NTU0ODYsImp0aSI6IjQ2ZDJkYzhlNWJiYjQyMTQ5YjZmMjZiNjhjNjEwYjIzIiwidXNlcl9pZCI6MjA4fQ.oOaq_6AjfZkRW8ZL9DKqyteFLBUqSvkcCN-CjiH-9iM"

	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Perform request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// Read and parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get devices: %s", resp.Status)
	}

	var devices []DeviceApiModel
	err = json.Unmarshal(body, &devices)
	if err != nil {
		panic(err)
	}

	var machines []MachineListModel
	for _, d := range devices {
		machines = append(machines, MachineListModel{
			MachineId:   d.ID,
			MachineName: d.Name,
		})
	}

	return machines, nil

}

type DataListResponse struct {
	OriginalData []OriginalDatum `json:"original_data"`
}

type OriginalDatum struct {
	Device struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"device"`
	ID           int                    `json:"id"`
	JSON         map[string]interface{} `json:"json"` // flexible dynamic fields
	ReceivedTime time.Time              `json:"received_time"`
}

type MachineWithData struct {
	machine_id   int
	machine_name string
	Json         map[string]interface{} `json:"json"`
}

func GetMachineWithData(machineId int) (*MachineWithData, error) {
	url := "https://api.metable.ir/api/data_list/"
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0b2tlbl90eXBlIjoiYWNjZXNzIiwiZXhwIjoxNzYwNTE5NDg2LCJpYXQiOjE3NTk2NTU0ODYsImp0aSI6IjQ2ZDJkYzhlNWJiYjQyMTQ5YjZmMjZiNjhjNjEwYjIzIiwidXNlcl9pZCI6MjA4fQ.oOaq_6AjfZkRW8ZL9DKqyteFLBUqSvkcCN-CjiH-9iM"

	// Prepare the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Execute
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// Read and parse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: %s\n%s\n", resp.Status, string(body))
		return nil, fmt.Errorf("failed to get machine data: %s", resp.Status)
	}

	var dataResp DataListResponse
	err = json.Unmarshal(body, &dataResp)
	if err != nil {
		panic(err)
	}

	var machineData *MachineWithData = getLatestByDeviceId(dataResp.OriginalData, machineId)

	if machineData == nil {
		return nil, fmt.Errorf("no data found for machine ID %d", machineId)
	}

	return machineData, nil

}

func getLatestByDeviceId(data []OriginalDatum, deviceID int) *MachineWithData {
	var latest *OriginalDatum
	for i := range data {
		d := &data[i]
		if d.Device.ID != deviceID {
			continue
		}
		if latest == nil || d.ReceivedTime.After(latest.ReceivedTime) {
			latest = d
		}
	}

	if latest == nil {
		return nil
	}

	machineData := &MachineWithData{
		machine_id:   latest.Device.ID,
		machine_name: latest.Device.Name,
		Json:         latest.JSON,
	}

	return machineData
}

func GetOutputResultsFromCodeResult(outputResult string, outputParams []string) ([]string, error) {
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

func GetParameterValuesFromMachine(machine MachineWithData, neededParameters []string) ([]string, error) {

	results := make([]string, len(neededParameters))
	for i, key := range neededParameters {
		if val, ok := machine.Json[key]; ok && val != nil {
			results[i] = fmt.Sprintf("%v", val) // convert any type to string
		} else {
			return nil, errors.New("key not found: " + key)
		}
	}
	return results, nil
}
