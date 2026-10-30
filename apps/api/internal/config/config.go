package config

import (
	"fmt"
	"os"
	"strconv"

	"infu-foundry/apps/api/internal/media"
)

type Config struct {
	Port          string
	Environment   string
	DatabaseURL   string
	GatewaySecret string
	R2            media.R2Config
}

func Load() (Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return Config{}, fmt.Errorf("PORT must be between 1 and 65535")
	}
	environment := os.Getenv("APP_ENV")
	if environment == "" {
		environment = "development"
	}
	if environment != "development" && environment != "production" {
		return Config{}, fmt.Errorf("APP_ENV must be development or production")
	}
	return Config{
		Port:          port,
		Environment:   environment,
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		GatewaySecret: os.Getenv("GATEWAY_SECRET"),
		R2: media.R2Config{
			AccountID:       os.Getenv("R2_ACCOUNT_ID"),
			Bucket:          os.Getenv("R2_BUCKET"),
			AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
		},
	}, nil
}
