package main

import (
	"controle-arcondicionado/internal/config"
	"controle-arcondicionado/internal/thinq"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var logger *log.Logger

func initLogger() (*os.File, error) {
	logFile, err := os.OpenFile("economizador.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger = log.New(multiWriter, "", log.Ldate|log.Ltime)

	return logFile, nil
}

func logMsg(format string, args ...interface{}) {
	logger.Printf(format, args...)
}

func logFatal(format string, args ...interface{}) {
	logger.Fatalf(format, args...)
}

func main() {
	// Initialize logger
	logFile, err := initLogger()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logFile.Close()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logFatal("Failed to load configuration: %v", err)
	}

	logMsg("=== LG ThinQ Energy Saver ===")
	logMsg("Minimum Temperature: %d°C", cfg.MinTemperature)
	logMsg("Country Code: %s", cfg.CountryCode)
	logMsg("Client ID: %s", cfg.ClientID)

	// Create ThinQ client
	client := thinq.NewClient(cfg.ThinQPAT, cfg.CountryCode, cfg.ClientID)

	// Get MQTT broker
	logMsg("Getting MQTT broker information...")
	mqttServer, err := client.GetMQTTRoute()
	if err != nil {
		logFatal("Failed to get MQTT route: %v", err)
	}
	logMsg("MQTT Server: %s", mqttServer)

	// Get device list first
	logMsg("Fetching devices...")
	devices, err := client.GetDeviceList()
	if err != nil {
		logFatal("Failed to get device list: %v", err)
	}
	logMsg("Found %d device(s)", len(devices))

	// Subscribe to events for each device
	logMsg("Subscribing to device events and push notifications...")
	for i, device := range devices {
		logMsg("[%d/%d] Subscribing to: %s", i+1, len(devices), device.Alias)

		// Subscribe to events
		if err := client.SubscribeToDeviceEvents(device.DeviceID); err != nil {
			logMsg("Warning: Failed to subscribe to events for %s: %v", device.Alias, err)
		}

		// Subscribe to push notifications
		if err := client.SubscribeToPushNotifications(device.DeviceID); err != nil {
			logMsg("Warning: Failed to subscribe to push for %s: %v", device.Alias, err)
		}
	}
	logMsg("Subscription complete!")

	// Fix devices that are already running below the minimum before the
	// first MQTT event arrives
	checkInitialStates(client, devices, cfg.MinTemperature)

	// Get MQTT credentials
	logMsg("Obtaining MQTT credentials...")
	credentials, err := client.GetMQTTCredentials()
	if err != nil {
		logFatal("Failed to get MQTT credentials: %v", err)
	}
	logMsg("Received certificate and %d subscription topic(s)", len(credentials.Subscriptions))

	// Setup TLS configuration
	tlsConfig, err := createTLSConfig(credentials)
	if err != nil {
		logFatal("Failed to create TLS config: %v", err)
	}

	// Setup MQTT options with message handler
	messageHandler := createMessageHandler(client, devices, cfg.MinTemperature)
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("ssl://%s", mqttServer))
	opts.SetClientID(cfg.ClientID)
	opts.SetTLSConfig(tlsConfig)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetDefaultPublishHandler(messageHandler)
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		logMsg("Connection lost: %v", err)
	})
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		logMsg("Connected to MQTT broker!")

		// Subscribe to all topics
		for _, topic := range credentials.Subscriptions {
			logMsg("Subscribing to: %s", topic)
			if token := client.Subscribe(topic, 1, nil); token.Wait() && token.Error() != nil {
				logMsg("Failed to subscribe to %s: %v", topic, token.Error())
			}
		}
		logMsg("Energy Saver Active! Minimum allowed: %d°C (press Ctrl+C to stop)...", cfg.MinTemperature)
	})

	// Create and start MQTT client
	mqttClient := mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		logFatal("Failed to connect to MQTT broker: %v", token.Error())
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	// Unsubscribe from all topics
	logMsg("Unsubscribing from topics...")
	for _, topic := range credentials.Subscriptions {
		if token := mqttClient.Unsubscribe(topic); token.Wait() && token.Error() != nil {
			logMsg("Failed to unsubscribe from %s: %v", topic, token.Error())
		}
	}

	logMsg("Disconnecting from MQTT broker...")
	mqttClient.Disconnect(250)
	logMsg("Energy Saver stopped. Goodbye!")
}

// createMessageHandler creates a message handler that adjusts temperature
func createMessageHandler(client *thinq.Client, devices []thinq.Device, minTemperature int) mqtt.MessageHandler {
	// Create device alias map for friendly names
	deviceAliases := make(map[string]string)
	for _, device := range devices {
		deviceAliases[device.DeviceID] = device.Alias
	}

	return func(_ mqtt.Client, msg mqtt.Message) {
		var event map[string]interface{}
		if err := json.Unmarshal(msg.Payload(), &event); err != nil {
			return
		}

		// Check if it's a device status event
		pushType, ok := event["pushType"].(string)
		if !ok || pushType != "DEVICE_STATUS" {
			return
		}

		deviceID, ok := event["deviceId"].(string)
		if !ok {
			return
		}

		// Get device alias
		alias := deviceAliases[deviceID]
		if alias == "" {
			alias = deviceID
		}

		// Check report for target temperature
		report, ok := event["report"].(map[string]interface{})
		if !ok {
			return
		}

		enforceMinTemperature(client, alias, deviceID, report, minTemperature)
	}
}

// checkInitialStates applies the temperature limit to every device once at
// startup, so devices already running below the minimum are fixed before the
// first MQTT event arrives.
func checkInitialStates(client *thinq.Client, devices []thinq.Device, minTemperature int) {
	logMsg("Checking current state of each device...")
	for i, device := range devices {
		logMsg("[%d/%d] Checking: %s", i+1, len(devices), device.Alias)

		state, err := client.GetDeviceState(device.DeviceID)
		if err != nil {
			logMsg("Warning: Failed to get state for %s: %v", device.Alias, err)
			continue
		}

		enforceMinTemperature(client, device.Alias, device.DeviceID, state, minTemperature)
	}
}

// enforceMinTemperature raises the target temperature to the configured minimum
// when the device is running below it. Devices that are powered off are skipped,
// because ThinQ rejects temperature commands in POWER OFF (error 2304).
func enforceMinTemperature(client *thinq.Client, alias, deviceID string, state map[string]interface{}, minTemperature int) {
	powerOn, powerKnown := thinq.PowerOn(state)
	if !powerKnown {
		// Pushes may carry partial reports, so ask the device for its state
		fetched, err := client.GetDeviceState(deviceID)
		if err != nil {
			logMsg("[%s] Could not determine power state, skipping adjustment: %v", alias, err)
			return
		}

		state = fetched
		powerOn, powerKnown = thinq.PowerOn(state)
		if !powerKnown {
			logMsg("[%s] Power state not reported by device, skipping adjustment", alias)
			return
		}
	}

	// Nothing to adjust while the device is off
	if !powerOn {
		return
	}

	targetTemp, ok := thinq.TargetTemperature(state)
	if !ok {
		return
	}

	// Only adjust if temperature is below minimum
	if int(targetTemp) >= minTemperature {
		return
	}

	// Adjust temperature to minimum
	logMsg("[%s] Temperature at %.0f°C (below minimum), adjusting to %d°C...",
		alias, targetTemp, minTemperature)

	if err := client.SetTemperature(deviceID, minTemperature); err != nil {
		logMsg("[%s] Failed to adjust temperature: %v", alias, err)
	} else {
		logMsg("[%s] Temperature adjusted successfully!", alias)
	}
}

// createTLSConfig creates TLS configuration from credentials
func createTLSConfig(credentials *thinq.MQTTCredentials) (*tls.Config, error) {
	// Load client certificate
	cert, err := tls.X509KeyPair([]byte(credentials.Certificate), []byte(credentials.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}, nil
}
