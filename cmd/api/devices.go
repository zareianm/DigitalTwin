package main

import (
	"DigitalTwin/pkg/deviceService"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
)

// GetDevices Returns devices with pagination
//
//	@Summary		Returns devices with pagination
//	@Description	Returns devices with pagination
//	@Tags			devices
//	@Accept			json
//	@Produce		json
//	@Param			page	path		int	true	"number of page"
//	@Success		200		{object}	[]DeviceListOutputModel
//	@Router			/api/v1/devices/{page} [get]
//	@Security		BearerAuth
func (app *application) getDevicesWithPagination(c *gin.Context) {

	accessToken, _ := c.Get("access_token")

	page, err := strconv.Atoi(c.Param("page"))

	if err != nil || page < 1 {
		page = 1
	}

	devices, statusCode, err := deviceService.GetDevicesWithPagination(accessToken.(string), page)

	if err != nil {
		c.JSON(statusCode, err.Error())
		return
	}

	var outputDevices []DeviceListOutputModel
	for _, d := range devices {
		outputDevices = append(outputDevices, DeviceListOutputModel{
			DeviceID:   d.DeviceId,
			DeviceName: d.DeviceName,
		})
	}

	c.JSON(http.StatusOK, outputDevices)
}

type DeviceListOutputModel struct {
	DeviceID   int    `json:"deviceId"`
	DeviceName string `json:"deviceName"`
}

// GetDeviceParameters returns device parameters
//
//	@Summary		returns device parameters
//	@Description	returns device parameters
//	@Tags			devices
//	@Accept			json
//	@Produce		json
//	@Param			device_id	path		int	true	"device ID"
//	@Success		200		{object}	[]string
//	@Router			/api/v1/devices/getDeviceParameters/{device_id} [get]
//	@Security		BearerAuth
func (app *application) getDeviceParameters(c *gin.Context) {
	deviceId, err := strconv.Atoi(c.Param("device_id"))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}

	accessToken, _ := c.Get("access_token")

	deviceData, statusCode, err := deviceService.GetDeviceWithData(deviceId, accessToken.(string))

	if err != nil {
		c.JSON(statusCode, err.Error())
		return
	}

	fields := make([]string, 0, len(deviceData.Json))
	for field := range deviceData.Json {
		fields = append(fields, field)
	}

	c.JSON(http.StatusOK, fields)
}
