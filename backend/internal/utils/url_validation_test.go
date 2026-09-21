package utils

import (
	"strings"
	"testing"
)

func TestValidateAndNormalizeURL(t *testing.T) {
	tests := []struct {
		name      string
		rawURL    string
		want      string
		wantErr   bool
		errorText string
	}{
		{
			name:   "valid HTTP URL",
			rawURL: "http://example.com/path",
			want:   "http://example.com/path",
		},
		{
			name:   "valid HTTPS URL",
			rawURL: " HTTPS://example.com/path ",
			want:   "https://example.com/path",
		},
		{
			name:      "empty URL",
			rawURL:    "   ",
			wantErr:   true,
			errorText: "URL cannot be empty",
		},
		{
			name:      "malformed URL",
			rawURL:    "https://[example.com",
			wantErr:   true,
			errorText: "URL is malformed",
		},
		{
			name:      "unsupported scheme",
			rawURL:    "ftp://example.com/file.txt",
			wantErr:   true,
			errorText: "URL scheme must be http or https",
		},
		{
			name:      "missing host",
			rawURL:    "https:///path",
			wantErr:   true,
			errorText: "URL must include a host",
		},
		{
			name:      "overly long URL",
			rawURL:    "https://example.com/" + strings.Repeat("a", MaxURLLength),
			wantErr:   true,
			errorText: "URL exceeds the maximum length",
		},
		{
			name:   "query string and fragment",
			rawURL: "https://example.com/search?q=go+url#results",
			want:   "https://example.com/search?q=go+url#results",
		},
		{
			name:   "international path",
			rawURL: "https://example.com/путь/naïve",
			want:   "https://example.com/%D0%BF%D1%83%D1%82%D1%8C/na%C3%AFve",
		},
		{
			name:      "embedded credentials",
			rawURL:    "https://user:password@example.com/path",
			wantErr:   true,
			errorText: "URLs with embedded credentials are not allowed",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ValidateAndNormalizeURL(test.rawURL)
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected validation error, got normalized URL %q", got)
				}
				if !strings.Contains(err.Error(), test.errorText) {
					t.Fatalf("expected error containing %q, got %q", test.errorText, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
			if got != test.want {
				t.Fatalf("expected normalized URL %q, got %q", test.want, got)
			}
		})
	}
}
