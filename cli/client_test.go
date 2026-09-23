package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoginStoresSessionAndCreateUsesIt(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "session.json")
	var sawSession bool
	requestCount := 0
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requestCount++
		switch r.URL.Path {
		case "/login":
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != "password=password&username=denzil" {
				t.Fatalf("unexpected login form: %s", body)
			}
			return response(http.StatusOK, `{"message":"Successfully logged in."}`, "atomicurl=session-value; Path=/"), nil
		case "/links":
			if cookie, err := r.Cookie("atomicurl"); err == nil && cookie.Value == "session-value" {
				sawSession = true
			}
			body, err := json.Marshal(linkResponse{Code: "demo", Destination: "https://example.com"})
			if err != nil {
				return nil, err
			}
			return response(http.StatusCreated, string(body), ""), nil
		default:
			return response(http.StatusNotFound, `{"error":"not found"}`, ""), nil
		}
	})

	client := NewClient("http://atomicurl.test", configPath)
	client.HTTPClient = &http.Client{Transport: transport}
	if err := client.Login(context.Background(), "denzil", "password"); err != nil {
		t.Fatalf("login failed: %v", err)
	}
	link, err := client.CreateLink(context.Background(), "Demo", "https://example.com", "")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if !sawSession || link.Code != "demo" {
		t.Fatalf("session or response not handled: session=%v link=%+v", sawSession, link)
	}
	if requestCount != 2 {
		t.Fatalf("expected two requests, got %d", requestCount)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("session file was not written: %v", err)
	}
}

func TestClientReturnsAPIError(t *testing.T) {
	client := NewClient("http://atomicurl.test", filepath.Join(t.TempDir(), "session.json"))
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return response(http.StatusUnauthorized, `{"error":"Please log in."}`, ""), nil
	})}
	err := client.Logout(context.Background())
	if err == nil || err.Error() != "server returned 401 Unauthorized: Please log in." {
		t.Fatalf("unexpected error: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func response(status int, body, setCookie string) *http.Response {
	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	if setCookie != "" {
		header.Set("Set-Cookie", setCookie)
	}
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
