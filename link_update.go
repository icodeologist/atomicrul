package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

var (
	errLinkNotFound    = errors.New("link not found")
	errSameDestination = errors.New("destination is unchanged")
)

type updateLinkRequest struct {
	Destination string `json:"destination"`
	Note        string `json:"note"`
}

type linkVersionResponse struct {
	ID          uint      `json:"id"`
	LinkID      uint      `json:"link_id"`
	Destination string    `json:"destination"`
	Note        string    `json:"note"`
	CreatedAt   time.Time `json:"created_at"`
}

type updateLinkResponse struct {
	Link    linkResponse        `json:"link"`
	Version linkVersionResponse `json:"version"`
}

func UpdateLink(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	if r.Method != http.MethodPatch {
		writeAPIError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		return
	}

	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(mux.Vars(r)["id"], 10, 0)
	if err != nil || id == 0 {
		writeAPIError(w, http.StatusNotFound, "Link not found.")
		return
	}

	var request updateLinkRequest
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

	note := strings.TrimSpace(request.Note)
	if note == "" {
		writeAPIError(w, http.StatusBadRequest, "Note is required.")
		return
	}
	destination, err := ValidateAndNormalizeURL(request.Destination)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	var link Link
	var version LinkVersion
	err = db.Transaction(func(tx *gorm.DB) error {
		current, findErr := findOwnedLink(tx, userID, uint(id))
		if isLinkNotFound(findErr) {
			return errLinkNotFound
		}
		if findErr != nil {
			return findErr
		}
		if current.Destination == destination {
			return errSameDestination
		}

		version = LinkVersion{
			LinkID:      current.ID,
			Destination: destination,
			Note:        note,
		}
		if err := tx.Create(&version).Error; err != nil {
			return err
		}
		if err := tx.Model(&current).Update("destination", destination).Error; err != nil {
			return err
		}
		if err := tx.First(&link, current.ID).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		switch {
		case errors.Is(err, errLinkNotFound):
			writeAPIError(w, http.StatusNotFound, "Link not found.")
		case errors.Is(err, errSameDestination):
			writeAPIError(w, http.StatusBadRequest, "Destination is unchanged.")
		default:
			writeAPIError(w, http.StatusInternalServerError, "Could not update link.")
		}
		return
	}

	writeJson(w, http.StatusOK, updateLinkResponse{
		Link: linkResponse{
			ID:          link.ID,
			Code:        link.Code,
			Title:       link.Title,
			Destination: link.Destination,
			Active:      link.Active,
			Clicks:      link.Clicks,
			UserID:      link.UserID,
			CreatedAt:   link.CreatedAt,
			UpdatedAt:   link.UpdatedAt,
		},
		Version: linkVersionResponse{
			ID:          version.ID,
			LinkID:      version.LinkID,
			Destination: version.Destination,
			Note:        version.Note,
			CreatedAt:   version.CreatedAt,
		},
	})
}
