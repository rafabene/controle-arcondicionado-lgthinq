package thinq

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	baseURL    = "https://api-aic.lgthinq.com"
	apiKey     = "v6GFvkweNo7DK7yD3ylIZ9w52aKBU0eJ7wLXkSR3"
)

// Client represents a ThinQ API client
type Client struct {
	httpClient  *http.Client
	accessToken string
	countryCode string
	clientID    string
}

// NewClient creates a new ThinQ API client
func NewClient(accessToken, countryCode, clientID string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		accessToken: accessToken,
		countryCode: countryCode,
		clientID:    clientID,
	}
}

// GetDeviceList retrieves the list of devices associated with the account
func (c *Client) GetDeviceList() ([]Device, error) {
	url := fmt.Sprintf("%s/devices", baseURL)

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
		// Try nested error format first
		var nestedErr ErrorResponseNested
		if err := json.Unmarshal(body, &nestedErr); err == nil && nestedErr.Error.Code != "" {
			return nil, fmt.Errorf("API error (code: %s): %s", nestedErr.Error.Code, nestedErr.Error.Message)
		}
		// Try flat error format
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.ResultCode != "" {
			return nil, fmt.Errorf("API error (code: %s): %s", errResp.ResultCode, errResp.Message)
		}
		// Fallback to raw body
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var deviceResp DeviceListResponse
	if err := json.Unmarshal(body, &deviceResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert DeviceResponse to Device
	devices := make([]Device, len(deviceResp.Response))
	for i, dr := range deviceResp.Response {
		devices[i] = Device{
			DeviceID:   dr.DeviceID,
			DeviceType: dr.DeviceInfo.DeviceType,
			ModelName:  dr.DeviceInfo.ModelName,
			Alias:      dr.DeviceInfo.Alias,
			Reportable: dr.DeviceInfo.Reportable,
		}
	}

	return devices, nil
}

// GetMQTTRoute retrieves MQTT broker information
func (c *Client) GetMQTTRoute() (string, error) {
	url := fmt.Sprintf("%s/route", baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var routeResp RouteResponse
	if err := json.Unmarshal(body, &routeResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Remove mqtts:// prefix if present
	mqttServer := routeResp.Response.MQTTServer
	mqttServer = strings.TrimPrefix(mqttServer, "mqtts://")

	return mqttServer, nil
}

// setHeaders sets common headers for API requests
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("x-client-id", c.clientID)
	req.Header.Set("x-country", c.countryCode)
	req.Header.Set("x-message-id", generateMessageID())
	req.Header.Set("x-service-phase", "OP")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
}

// GetMQTTCredentials retrieves MQTT credentials (certificate, private key, etc.)
func (c *Client) GetMQTTCredentials() (*MQTTCredentials, error) {
	// Step 1: Register client
	if err := c.registerClient(); err != nil {
		return nil, fmt.Errorf("failed to register client: %w", err)
	}

	// Step 2: Generate CSR
	privateKey, csrPEM, err := generateCSR(c.clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CSR: %w", err)
	}

	// Step 3: Request certificate from API
	certReq := CertificateRequest{
		ServiceCode: "SVC202",
		CSR:         csrPEM,
	}
	reqBody, err := json.Marshal(certReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal certificate request: %w", err)
	}

	url := fmt.Sprintf("%s/client/certificate", baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
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

	var certResp CertificateResponse
	if err := json.Unmarshal(body, &certResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &MQTTCredentials{
		Certificate:   certResp.Response.Result.CertificatePem,
		PrivateKey:    privateKey,
		Subscriptions: certResp.Response.Result.Subscriptions,
	}, nil
}

// MQTTCredentials contains all credentials needed for MQTT connection
type MQTTCredentials struct {
	Certificate   string
	PrivateKey    string
	Subscriptions []string
}

// registerClient registers the client with the ThinQ API
func (c *Client) registerClient() error {
	regReq := ClientRegisterRequest{
		Type:        "MQTT",
		ServiceCode: "SVC202",
		DeviceType:  "607",
		AllowExist:  true,
	}

	reqBody, err := json.Marshal(regReq)
	if err != nil {
		return fmt.Errorf("failed to marshal register request: %w", err)
	}

	url := fmt.Sprintf("%s/client", baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Accept both 200 and 409 (already registered)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// generateCSR generates a private key and certificate signing request
func generateCSR(clientID string) (string, string, error) {
	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create CSR template
	// According to Python SDK, CommonName should be "lg_thinq"
	template := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: "lg_thinq",
		},
	}

	// Create CSR
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &template, privateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to create CSR: %w", err)
	}

	// Encode CSR to PEM
	csrPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrDER,
	})

	// Encode private key to PEM
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return string(privateKeyPEM), string(csrPEM), nil
}

// SubscribeToDeviceEvents subscribes to events for a specific device
func (c *Client) SubscribeToDeviceEvents(deviceID string) error {
	url := fmt.Sprintf("%s/event/%s/subscribe", baseURL, deviceID)

	// Event subscription requires expiration time
	payload := map[string]interface{}{
		"expire": map[string]interface{}{
			"unit":  "HOUR",
			"timer": 4464, // ~6 months
		},
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Accept 200 (success) or 409 (already subscribed)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SubscribeToPushNotifications subscribes to push notifications for a specific device
func (c *Client) SubscribeToPushNotifications(deviceID string) error {
	url := fmt.Sprintf("%s/push/%s/subscribe", baseURL, deviceID)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Accept 200 (success), 409 (already subscribed), or 404 with code 1207 (already subscribed)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// If 404, verify it's the "already subscribed" message (code 1207)
	if resp.StatusCode == http.StatusNotFound {
		var errResp ErrorResponseNested
		if err := json.Unmarshal(body, &errResp); err == nil {
			if errResp.Error.Code != "1207" {
				return fmt.Errorf("API error (code: %s): %s", errResp.Error.Code, errResp.Error.Message)
			}
			// Code 1207 = "Already subscribed push" - this is OK, just silently ignore
		}
	}

	return nil
}

// SetTemperature sets the target temperature for a device
func (c *Client) SetTemperature(deviceID string, temperature int) error {
	url := fmt.Sprintf("%s/devices/%s/control", baseURL, deviceID)

	// Payload format without dataSetList wrapper - send resource directly
	payload := map[string]interface{}{
		"temperature": map[string]interface{}{
			"targetTemperature": temperature,
		},
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)
	req.Header.Set("x-conditional-control", "true")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// generateMessageID creates a unique message ID for each request
func generateMessageID() string {
	id := uuid.New()
	encoded := base64.URLEncoding.EncodeToString(id[:])
	// Remove padding
	return encoded[:len(encoded)-2]
}
