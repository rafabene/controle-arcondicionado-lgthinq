package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

type Config struct {
	ThinQPAT       string
	CountryCode    string
	ClientID       string
	MinTemperature int
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	minTemp := 21 // Default minimum temperature
	if tempStr := os.Getenv("MIN_TEMPERATURE"); tempStr != "" {
		if temp, err := strconv.Atoi(tempStr); err == nil {
			minTemp = temp
		}
	}
	if minTemp < 21 {
		minTemp = 21
	}

	cfg := &Config{
		ThinQPAT:       os.Getenv("THINQ_PAT"),
		CountryCode:    os.Getenv("THINQ_COUNTRY_CODE"),
		ClientID:       os.Getenv("THINQ_CLIENT_ID"),
		MinTemperature: minTemp,
	}

	if cfg.ThinQPAT == "" {
		return nil, fmt.Errorf("THINQ_PAT is required")
	}

	if cfg.CountryCode == "" {
		cfg.CountryCode = "BR" // Default to Brazil
	}

	if cfg.ClientID == "" {
		// Generate a unique client ID if not provided
		cfg.ClientID = generateClientID()
	}

	return cfg, nil
}

func generateClientID() string {
	// AWS IoT Thing names must match pattern: [a-zA-Z0-9:_-]+
	// Generate UUID and format it properly
	id := uuid.New().String()
	// Replace dots and other invalid chars with hyphens
	re := regexp.MustCompile(`[^a-zA-Z0-9:_-]`)
	validID := re.ReplaceAllString(id, "-")
	return fmt.Sprintf("go-thinq-%s", validID)
}
