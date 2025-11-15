package main

import (
	"controle-arcondicionado/internal/config"
	"controle-arcondicionado/internal/thinq"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	targetTemperature = 24
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Println("=== LG ThinQ Energy Saver ===")
	fmt.Printf("Target Temperature: %d°C\n", targetTemperature)
	fmt.Printf("Country Code: %s\n", cfg.CountryCode)
	fmt.Printf("Client ID: %s\n\n", cfg.ClientID)

	// Create ThinQ client
	client := thinq.NewClient(cfg.ThinQPAT, cfg.CountryCode, cfg.ClientID)

	// Get MQTT broker
	fmt.Println("Getting MQTT broker information...")
	mqttServer, err := client.GetMQTTRoute()
	if err != nil {
		log.Fatalf("Failed to get MQTT route: %v", err)
	}
	fmt.Printf("MQTT Server: %s\n\n", mqttServer)

	// Get device list first
	fmt.Println("Fetching devices...")
	devices, err := client.GetDeviceList()
	if err != nil {
		log.Fatalf("Failed to get device list: %v", err)
	}
	fmt.Printf("Found %d device(s)\n\n", len(devices))

	// Subscribe to events for each device
	fmt.Println("Subscribing to device events and push notifications...")
	for i, device := range devices {
		fmt.Printf("[%d/%d] Subscribing to: %s\n", i+1, len(devices), device.Alias)

		// Subscribe to events
		if err := client.SubscribeToDeviceEvents(device.DeviceID); err != nil {
			log.Printf("Warning: Failed to subscribe to events for %s: %v", device.Alias, err)
		}

		// Subscribe to push notifications
		if err := client.SubscribeToPushNotifications(device.DeviceID); err != nil {
			log.Printf("Warning: Failed to subscribe to push for %s: %v", device.Alias, err)
		}
	}
	fmt.Println("Subscription complete!\n")

	// Get MQTT credentials
	fmt.Println("Obtaining MQTT credentials...")
	credentials, err := client.GetMQTTCredentials()
	if err != nil {
		log.Fatalf("Failed to get MQTT credentials: %v", err)
	}
	fmt.Printf("Received certificate and %d subscription topic(s)\n\n", len(credentials.Subscriptions))

	// Setup TLS configuration
	tlsConfig, err := createTLSConfig(credentials)
	if err != nil {
		log.Fatalf("Failed to create TLS config: %v", err)
	}

	// Setup MQTT options with message handler
	messageHandler := createMessageHandler(client, devices)
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("ssl://%s", mqttServer))
	opts.SetClientID(cfg.ClientID)
	opts.SetTLSConfig(tlsConfig)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetDefaultPublishHandler(messageHandler)
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		fmt.Printf("Connection lost: %v\n", err)
	})
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		fmt.Println("Connected to MQTT broker!")

		// Subscribe to all topics
		for _, topic := range credentials.Subscriptions {
			fmt.Printf("Subscribing to: %s\n", topic)
			if token := client.Subscribe(topic, 1, nil); token.Wait() && token.Error() != nil {
				log.Printf("Failed to subscribe to %s: %v", topic, token.Error())
			}
		}
		fmt.Printf("\n🌱 Energy Saver Active! Auto-adjusting to %d°C (press Ctrl+C to stop)...\n\n", targetTemperature)
	})

	// Create and start MQTT client
	mqttClient := mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nDisconnecting from MQTT broker...")
	mqttClient.Disconnect(250)
	fmt.Println("Energy Saver stopped. Goodbye!")
}

// createMessageHandler creates a message handler that adjusts temperature
func createMessageHandler(client *thinq.Client, devices []thinq.Device) mqtt.MessageHandler {
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

		temperature, ok := report["temperature"].(map[string]interface{})
		if !ok {
			return
		}

		// Check if target temperature is set and different from our target
		targetTemp, hasTarget := temperature["targetTemperature"].(float64)
		if !hasTarget {
			return
		}

		// If temperature is already at target, skip
		if int(targetTemp) == targetTemperature {
			return
		}

		// Adjust temperature
		fmt.Printf("[%s] 🌡️  Temperature at %.0f°C, adjusting to %d°C...\n",
			time.Now().Format("15:04:05"), targetTemp, targetTemperature)
		fmt.Printf("           Device: %s\n", alias)

		if err := client.SetTemperature(deviceID, targetTemperature); err != nil {
			fmt.Printf("           ❌ Failed: %v\n\n", err)
		} else {
			fmt.Printf("           ✅ Temperature adjusted successfully!\n\n")
		}
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
