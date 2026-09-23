package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type sessionFile struct {
	Cookies map[string]string `json:"cookies"`
}

func loadSession(path string) (sessionFile, error) {
	contents, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return sessionFile{Cookies: make(map[string]string)}, nil
	}
	if err != nil {
		return sessionFile{}, fmt.Errorf("read session file: %w", err)
	}

	var session sessionFile
	if err := json.Unmarshal(contents, &session); err != nil {
		return sessionFile{}, fmt.Errorf("decode session file: %w", err)
	}
	if session.Cookies == nil {
		session.Cookies = make(map[string]string)
	}
	return session, nil
}

func saveSession(path string, session sessionFile) error {
	if session.Cookies == nil {
		session.Cookies = make(map[string]string)
	}
	contents, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("encode session file: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create session directory: %w", err)
	}
	if err := os.WriteFile(path, append(contents, '\n'), 0o600); err != nil {
		return fmt.Errorf("write session file: %w", err)
	}
	return nil
}
