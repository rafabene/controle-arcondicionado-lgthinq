package thinq

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// GetDeviceState retrieves the current state of a device.
// The returned map has the same shape as the "report" of a DEVICE_STATUS push,
// so both can be read with PowerOn and TargetTemperature.
func (c *Client) GetDeviceState(deviceID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/devices/%s/state", baseURL, deviceID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var stateResp DeviceStateResponse
	if err := json.Unmarshal(body, &stateResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return stateResp.Response, nil
}

// PowerOn reports whether the device is powered on and whether that could be
// determined from the given state. Pushes often carry partial reports, so an
// unknown power state does not mean the device is off.
func PowerOn(state map[string]interface{}) (on bool, known bool) {
	operation, ok := state["operation"].(map[string]interface{})
	if !ok {
		return false, false
	}

	mode, ok := operation["airConOperationMode"].(string)
	if !ok {
		return false, false
	}

	switch strings.ToUpper(mode) {
	case "POWER_ON":
		return true, true
	case "POWER_OFF":
		return false, true
	}

	return false, false
}

// TargetTemperature returns the target temperature reported in the given state.
func TargetTemperature(state map[string]interface{}) (float64, bool) {
	temperature, ok := state["temperature"].(map[string]interface{})
	if !ok {
		return 0, false
	}

	return toFloat(temperature["targetTemperature"])
}

// toFloat converts a JSON number to float64
func toFloat(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	}

	return 0, false
}
