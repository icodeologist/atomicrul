package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	BaseURL    string
	ConfigPath string
	HTTPClient *http.Client
}

type registerResponse struct {
	Message string `json:"message"`
}

type linkResponse struct {
	ID          uint   `json:"id"`
	Code        string `json:"code"`
	Title       string `json:"title"`
	Destination string `json:"destination"`
}

type apiError struct {
	Message string `json:"error"`
}

func NewClient(baseURL, configPath string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		ConfigPath: configPath,
		HTTPClient: http.DefaultClient,
	}
}

func (c *Client) Register(ctx context.Context, username, email, password string) error {
	form := url.Values{
		"username": {username},
		"email":    {email},
		"password": {password},
	}
	return c.do(ctx, http.MethodPost, "/register", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded", nil)
}

func (c *Client) Login(ctx context.Context, username, password string) error {
	form := url.Values{
		"username": {username},
		"password": {password},
	}
	return c.do(ctx, http.MethodPost, "/login", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded", nil)
}

func (c *Client) Logout(ctx context.Context) error {
	if err := c.do(ctx, http.MethodPost, "/logout", nil, "", nil); err != nil {
		return err
	}
	return saveSession(c.ConfigPath, sessionFile{Cookies: map[string]string{}})
}

func (c *Client) CreateLink(ctx context.Context, title, destination, code string) (linkResponse, error) {
	body, err := json.Marshal(struct {
		Title       string `json:"title,omitempty"`
		Destination string `json:"destination"`
		Code        string `json:"code,omitempty"`
	}{
		Title:       title,
		Destination: destination,
		Code:        code,
	})
	if err != nil {
		return linkResponse{}, fmt.Errorf("encode link request: %w", err)
	}

	var response linkResponse
	if err := c.do(ctx, http.MethodPost, "/links", bytes.NewReader(body), "application/json", &response); err != nil {
		return linkResponse{}, err
	}
	return response, nil
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader, contentType string, output any) error {
	request, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}

	session, err := loadSession(c.ConfigPath)
	if err != nil {
		return err
	}
	for name, value := range session.Cookies {
		request.AddCookie(&http.Cookie{Name: name, Value: value})
	}

	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer response.Body.Close()

	for _, cookie := range response.Cookies() {
		if cookie.MaxAge < 0 || cookie.Value == "" {
			delete(session.Cookies, cookie.Name)
			continue
		}
		session.Cookies[cookie.Name] = cookie.Value
	}
	if err := saveSession(c.ConfigPath, session); err != nil {
		return err
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		var apiResponse apiError
		if json.Unmarshal(responseBody, &apiResponse) == nil && apiResponse.Message != "" {
			return fmt.Errorf("server returned %s: %s", response.Status, apiResponse.Message)
		}
		return fmt.Errorf("server returned %s", response.Status)
	}
	if output == nil || len(responseBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(responseBody, output); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *Client) ShortURL(link linkResponse) string {
	return c.BaseURL + "/" + link.Code
}

func validateClient(client *Client) error {
	if client == nil || client.BaseURL == "" {
		return errors.New("base URL is required")
	}
	parsed, err := url.Parse(client.BaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("base URL must include a scheme and host, for example http://localhost:3000")
	}
	return nil
}
