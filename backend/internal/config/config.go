package config

import (
	"fmt"
	"strconv"
	"strings"
)

const defaultHTTPPort = "3000"

type AppConfig struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	SecretKey  string
	AppEnv     string
	HTTPPort   string
}

func LoadConfig(getenv func(string) string) (AppConfig, error) {
	config := AppConfig{
		DBHost:     envWithFallback(getenv, "ATOMICURL_DB_HOST", "HOST"),
		DBPort:     envWithFallback(getenv, "ATOMICURL_DB_PORT", "PORT"),
		DBUser:     envWithFallback(getenv, "ATOMICURL_DB_USER", "USER"),
		DBPassword: envWithFallback(getenv, "ATOMICURL_DB_PASSWORD", "PASSWORD"),
		DBName:     envWithFallback(getenv, "ATOMICURL_DB_NAME", "DBNAME"),
		SecretKey:  envWithFallback(getenv, "ATOMICURL_SECRET_KEY", "SECRETKEY"),
		AppEnv:     envWithFallback(getenv, "ATOMICURL_APP_ENV", "APP_ENV"),
		HTTPPort:   getenv("ATOMICURL_HTTP_PORT"),
	}
	if config.HTTPPort == "" {
		config.HTTPPort = defaultHTTPPort
	}

	missing := make([]string, 0)
	for name, value := range map[string]string{
		"database host":     config.DBHost,
		"database port":     config.DBPort,
		"database user":     config.DBUser,
		"database password": config.DBPassword,
		"database name":     config.DBName,
		"secret key":        config.SecretKey,
	} {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return AppConfig{}, fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}
	if err := validatePort("database port", config.DBPort); err != nil {
		return AppConfig{}, err
	}
	if err := validatePort("HTTP port", config.HTTPPort); err != nil {
		return AppConfig{}, err
	}
	if len(config.SecretKey) < 32 {
		return AppConfig{}, fmt.Errorf("secret key must be at least 32 characters long")
	}
	return config, nil
}

func envWithFallback(getenv func(string) string, preferred, legacy string) string {
	if value := getenv(preferred); value != "" {
		return value
	}
	return getenv(legacy)
}

func validatePort(name, value string) error {
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("%s must be a valid TCP port", name)
	}
	return nil
}
