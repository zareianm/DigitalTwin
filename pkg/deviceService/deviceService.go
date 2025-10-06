package deviceService

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

type DeviceListModel struct {
	DeviceId   int
	DeviceName string
}

func GetAllDevices(token string) ([]DeviceListModel, error) {
	url := "https://api.metable.ir/api/device_list/"

	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", token)

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

	var devicesFromApi []DeviceApiModel
	err = json.Unmarshal(body, &devicesFromApi)
	if err != nil {
		panic(err)
	}

	var devices []DeviceListModel
	for _, d := range devicesFromApi {
		devices = append(devices, DeviceListModel{
			DeviceId:   d.ID,
			DeviceName: d.Name,
		})
	}

	return devices, nil

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

func GetDeviceWithData(deviceId int, token string) (*DeviceWithData, error) {
	url := "https://api.metable.ir/api/data_list/"

	// Prepare the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", token)

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
		return nil, fmt.Errorf("failed to get device data: %s", resp.Status)
	}

	var dataResp DataListResponse
	err = json.Unmarshal(body, &dataResp)
	if err != nil {
		panic(err)
	}

	var deviceData *DeviceWithData = getLatestByDeviceId(dataResp.OriginalData, deviceId)

	if deviceData == nil {
		return nil, fmt.Errorf("no data found for device ID %d", deviceId)
	}

	return deviceData, nil

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
