package utils

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const MaxURLLength = 2048

// ValidateAndNormalizeURL validates a destination URL and returns its normalized form.
func ValidateAndNormalizeURL(rawURL string) (string, error) {
	value := strings.TrimSpace(rawURL)
	if value == "" {
		return "", errors.New("URL cannot be empty")
	}
	if len(value) > MaxURLLength {
		return "", fmt.Errorf("URL exceeds the maximum length of %d characters", MaxURLLength)
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "", errors.New("URL is malformed")
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", errors.New("URL scheme must be http or https")
	}
	if parsed.Host == "" || parsed.Hostname() == "" {
		return "", errors.New("URL must include a host")
	}
	if parsed.User != nil {
		return "", errors.New("URLs with embedded credentials are not allowed")
	}

	parsed.Scheme = scheme
	return parsed.String(), nil
}
