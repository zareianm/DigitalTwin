package deviceService

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type DeviceApiModel struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type DeviceListModel struct {
	DeviceId   int
	DeviceName string
}

func GetDevicesWithPagination(token string, page int) ([]DeviceListModel, int, error) {
	url := fmt.Sprintf("https://api.metable.ir/api/device_list/?page=%d", page)

	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, 500, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", token)

	// Perform request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 500, err
	}
	defer resp.Body.Close()

	// Read and parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 500, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, 500, fmt.Errorf("failed to get devices: %s", resp.Status)
	}

	var devicesFromApi []DeviceApiModel
	err = json.Unmarshal(body, &devicesFromApi)
	if err != nil {
		return nil, 500, err
	}

	var devices []DeviceListModel
	for _, d := range devicesFromApi {
		devices = append(devices, DeviceListModel{
			DeviceId:   d.ID,
			DeviceName: d.Name,
		})
	}

	return devices, 200, nil

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

type DeviceWithData struct {
	DeviceId   int
	DeviceName string
	Json       map[string]interface{} `json:"json"`
}

func GetDeviceWithData(deviceId int, token string) (*DeviceWithData, int, error) {
	url := fmt.Sprintf("https://api.metable.ir/api/data_list/?devices=%d", deviceId)

	// Prepare the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, 500, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", token)

	// Execute
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 500, err
	}
	defer resp.Body.Close()

	// Read and parse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 500, err
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: %s\n%s\n", resp.Status, string(body))
		return nil, 500, fmt.Errorf("failed to get device data: %s", resp.Status)
	}

	var dataResp DataListResponse
	err = json.Unmarshal(body, &dataResp)
	if err != nil {
		return nil, 500, err
	}

	var deviceData *DeviceWithData = getLatestByDeviceId(dataResp.OriginalData, deviceId)

	if deviceData == nil {
		return nil, 404, fmt.Errorf("no data found for device ID %d", deviceId)
	}

	return deviceData, 200, nil

}

func getLatestByDeviceId(data []OriginalDatum, deviceID int) *DeviceWithData {
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

	deviceData := &DeviceWithData{
		DeviceId:   latest.Device.ID,
		DeviceName: latest.Device.Name,
		Json:       latest.JSON,
	}

	return deviceData
}

func GetOutputResultsFromCodeResult(outputResult string, outputParams []string) ([]string, error) {
	// Parse JSON into a generic map
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(outputResult), &obj); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	results := make([]string, 0, len(outputParams))

	for _, key := range outputParams {
		val, ok := obj[key]
		if !ok {
			return nil, errors.New("key not found: " + key)
		}

		switch v := val.(type) {
		case string:
			results = append(results, v)
		case float64:
			// Json numbers go to float64 => format without trailing ".0"
			results = append(results, strconv.FormatFloat(v, 'f', -1, 64))
		case bool:
			results = append(results, strconv.FormatBool(v))
		case nil:
			// choose behavior; here we treat nil as an error (you can switch to append(""))
			return nil, errors.New("key is null: " + key)
		default:
			// For arrays/objects or anything unexpected, compact back to JSON
			b, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("cannot stringify key %q: %w", key, err)
			}
			results = append(results, string(b))
		}
	}

	return results, nil
}

func GetParameterValuesFromDevice(device DeviceWithData, neededParameters []string) ([]string, error) {

	results := make([]string, len(neededParameters))
	for i, key := range neededParameters {
		if val, ok := device.Json[key]; ok && val != nil {
			results[i] = fmt.Sprintf("%v", val) // convert any type to string
		} else {
			return nil, errors.New("key not found: " + key)
		}
	}
	return results, nil
}
