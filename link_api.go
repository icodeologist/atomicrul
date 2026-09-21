package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	linkCodeMinLength = 3
	linkCodeMaxLength = 64
	generatedCodeSize = 10
)

var errDuplicateLinkCode = errors.New("link code already exists")

var reservedLinkCodes = map[string]struct{}{
	"register":     {},
	"login":        {},
	"links":        {},
	"create":       {},
	"logout":       {},
	"greetme":      {},
	"dashboard":    {},
	"remake_links": {},
}

type createLinkRequest struct {
	Title       string `json:"title"`
	Destination string `json:"destination"`
	Code        string `json:"code"`
}

type linkResponse struct {
	ID          uint      `json:"id"`
	Code        string    `json:"code"`
	Title       string    `json:"title"`
	Destination string    `json:"destination"`
	Active      bool      `json:"active"`
	Clicks      int       `json:"clicks"`
	UserID      uint      `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func CreateLink(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		return
	}

	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}

	var request createLinkRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	if err := decoder.Decode(&request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Request body must be valid JSON.")
		return
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeAPIError(w, http.StatusBadRequest, "Request body must contain one JSON object.")
		return
	}

	destination, err := ValidateAndNormalizeURL(request.Destination)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	customCode := strings.TrimSpace(request.Code)
	if customCode != "" {
		if err := validateLinkCode(customCode); err != nil {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	var link Link
	err = db.Transaction(func(tx *gorm.DB) error {
		for attempts := 0; attempts < 5; attempts++ {
			code := customCode
			if code == "" {
				code, err = generateLinkCode()
				if err != nil {
					return err
				}
			}

			var existing Link
			if queryErr := tx.Where("code = ?", code).First(&existing).Error; queryErr == nil {
				if customCode != "" {
					return errDuplicateLinkCode
				}
				continue
			} else if !errors.Is(queryErr, gorm.ErrRecordNotFound) {
				return queryErr
			}

			candidate := Link{
				Code:        code,
				Title:       request.Title,
				Destination: destination,
				Active:      true,
				UserID:      userID,
			}
			if createErr := tx.Create(&candidate).Error; createErr != nil {
				if customCode != "" && isDuplicateCodeError(createErr) {
					return errDuplicateLinkCode
				}
				if customCode == "" && isDuplicateCodeError(createErr) {
					continue
				}
				return createErr
			}

			version := LinkVersion{
				LinkID:      candidate.ID,
				Destination: destination,
			}
			if err := tx.Create(&version).Error; err != nil {
				return err
			}
			link = candidate
			return nil
		}
		return errors.New("could not generate a unique link code")
	})
	if err != nil {
		if errors.Is(err, errDuplicateLinkCode) {
			writeAPIError(w, http.StatusConflict, "That link code is already in use.")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "Could not create link.")
		return
	}

	writeJson(w, http.StatusCreated, linkResponse{
		ID:          link.ID,
		Code:        link.Code,
		Title:       link.Title,
		Destination: link.Destination,
		Active:      link.Active,
		Clicks:      link.Clicks,
		UserID:      link.UserID,
		CreatedAt:   link.CreatedAt,
		UpdatedAt:   link.UpdatedAt,
	})
}

func validateLinkCode(code string) error {
	if len(code) < linkCodeMinLength || len(code) > linkCodeMaxLength {
		return fmt.Errorf("code must be between %d and %d characters", linkCodeMinLength, linkCodeMaxLength)
	}
	if _, reserved := reservedLinkCodes[strings.ToLower(code)]; reserved {
		return errors.New("code is reserved")
	}
	for _, character := range code {
		if (character < 'a' || character > 'z') &&
			(character < 'A' || character > 'Z') &&
			(character < '0' || character > '9') &&
			character != '-' && character != '_' {
			return errors.New("code may contain only letters, numbers, hyphens, and underscores")
		}
	}
	return nil
}

func generateLinkCode() (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, generatedCodeSize)
	for index := range code {
		value, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		code[index] = alphabet[value.Int64()]
	}
	return string(code), nil
}

func isDuplicateCodeError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate key") && strings.Contains(message, "code") ||
		strings.Contains(message, "unique constraint failed") && strings.Contains(message, "code")
}
