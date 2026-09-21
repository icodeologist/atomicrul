package main

import (
	"strings"
	"testing"
)

func TestLoadConfigUsesClearNamesAndDefaults(t *testing.T) {
	values := map[string]string{
		"ATOMICURL_DB_HOST":     "db.internal",
		"ATOMICURL_DB_PORT":     "5432",
		"ATOMICURL_DB_USER":     "atomicurl",
		"ATOMICURL_DB_PASSWORD": "not-a-real-password",
		"ATOMICURL_DB_NAME":     "atomicurl",
		"ATOMICURL_SECRET_KEY":  strings.Repeat("s", 32),
		"ATOMICURL_APP_ENV":     "production",
	}

	config, err := LoadConfig(func(name string) string { return values[name] })
	if err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
	if config.DBHost != "db.internal" || config.HTTPPort != defaultHTTPPort || config.AppEnv != "production" {
		t.Fatalf("unexpected config: %+v", config)
	}
}

func TestLoadConfigSupportsLegacyNames(t *testing.T) {
	values := map[string]string{
		"HOST":      "localhost",
		"PORT":      "5432",
		"USER":      "postgres",
		"PASSWORD":  "password",
		"DBNAME":    "atomicurl",
		"SECRETKEY": strings.Repeat("k", 32),
	}

	config, err := LoadConfig(func(name string) string { return values[name] })
	if err != nil {
		t.Fatalf("expected legacy config to remain supported, got %v", err)
	}
	if config.DBUser != "postgres" || config.DBName != "atomicurl" {
		t.Fatalf("unexpected legacy config: %+v", config)
	}
}

func TestLoadConfigRejectsMissingRequiredValues(t *testing.T) {
	_, err := LoadConfig(func(string) string { return "" })
	if err == nil {
		t.Fatal("expected missing configuration to be rejected")
	}
	if !strings.Contains(err.Error(), "missing required configuration") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestLoadConfigRejectsInvalidPortsAndShortSecrets(t *testing.T) {
	base := map[string]string{
		"ATOMICURL_DB_HOST":     "localhost",
		"ATOMICURL_DB_PORT":     "5432",
		"ATOMICURL_DB_USER":     "postgres",
		"ATOMICURL_DB_PASSWORD": "password",
		"ATOMICURL_DB_NAME":     "atomicurl",
		"ATOMICURL_SECRET_KEY":  strings.Repeat("k", 32),
	}
	base["ATOMICURL_HTTP_PORT"] = "not-a-port"
	if _, err := LoadConfig(func(name string) string { return base[name] }); err == nil {
		t.Fatal("expected invalid HTTP port to be rejected")
	}

	base["ATOMICURL_HTTP_PORT"] = "3000"
	base["ATOMICURL_SECRET_KEY"] = "short"
	if _, err := LoadConfig(func(name string) string { return base[name] }); err == nil {
		t.Fatal("expected short secret to be rejected")
	}
}
