package thinq

// DeviceListResponse represents the response from the device list API
type DeviceListResponse struct {
	MessageID string           `json:"messageId"`
	Timestamp string           `json:"timestamp"`
	Response  []DeviceResponse `json:"response"`
}

// DeviceResponse wraps device information
type DeviceResponse struct {
	DeviceID   string     `json:"deviceId"`
	DeviceInfo DeviceInfo `json:"deviceInfo"`
}

// DeviceInfo contains the device details
type DeviceInfo struct {
	DeviceType string `json:"deviceType"`
	ModelName  string `json:"modelName"`
	Alias      string `json:"alias"`
	Reportable bool   `json:"reportable"`
}

// Device represents a ThinQ device (simplified for display)
type Device struct {
	DeviceID   string `json:"deviceId"`
	DeviceType string `json:"deviceType"`
	ModelName  string `json:"modelName"`
	Alias      string `json:"alias"`
	Reportable bool   `json:"reportable"`
}

// ErrorResponse represents an error from the API
type ErrorResponse struct {
	ResultCode string `json:"resultCode"`
	Message    string `json:"message"`
}

// ErrorResponseNested represents an error response with nested error object
type ErrorResponseNested struct {
	MessageID string `json:"messageId"`
	Timestamp string `json:"timestamp"`
	Error     struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}
