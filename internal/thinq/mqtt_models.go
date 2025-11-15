package thinq

// RouteResponse contains MQTT broker information
type RouteResponse struct {
	MessageID string     `json:"messageId"`
	Timestamp string     `json:"timestamp"`
	Response  RouteInfo  `json:"response"`
}

// RouteInfo contains the MQTT server details
type RouteInfo struct {
	MQTTServer string `json:"mqttServer"`
}

// ClientRegisterRequest for registering the client
type ClientRegisterRequest struct {
	Type        string `json:"type"`
	ServiceCode string `json:"service-code"`
	DeviceType  string `json:"device-type"`
	AllowExist  bool   `json:"allowExist"`
}

// ClientRegisterResponse contains registration details
type ClientRegisterResponse struct {
	MessageID string              `json:"messageId"`
	Timestamp string              `json:"timestamp"`
	Response  ClientRegisterInfo  `json:"response"`
}

// ClientRegisterInfo contains client registration data
type ClientRegisterInfo struct {
	CSR string `json:"csr"`
}

// CertificateRequest for requesting a certificate
type CertificateRequest struct {
	ServiceCode string `json:"service-code"`
	CSR         string `json:"csr"`
}

// CertificateResponse contains certificate details
type CertificateResponse struct {
	MessageID string                   `json:"messageId"`
	Timestamp string                   `json:"timestamp"`
	Response  CertificateResponseData  `json:"response"`
}

// CertificateResponseData wraps the result
type CertificateResponseData struct {
	ResultCode string          `json:"resultCode"`
	Result     CertificateInfo `json:"result"`
}

// CertificateInfo contains certificate and subscription data
type CertificateInfo struct {
	CertificatePem string   `json:"certificatePem"`
	Subscriptions  []string `json:"subscriptions"`
	Publications   []string `json:"publications"`
}

// DeviceStateMessage represents a device state change event
type DeviceStateMessage struct {
	DeviceID  string                 `json:"deviceId"`
	Timestamp string                 `json:"timestamp"`
	State     map[string]interface{} `json:"state"`
}
