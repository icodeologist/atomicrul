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
		writeJson(w, http.StatusMethodNotAllowed, apiError{Err: "Method not allowed."})
		return
	}

	session, ok := getSession(w, r)
	if !ok {
		return
	}
	if session.Values["authenticated"] != true {
		writeJson(w, http.StatusUnauthorized, apiError{Err: "Please log in."})
		return
	}
	userID, ok := sessionUserID(session.Values["userid"])
	if !ok {
		writeJson(w, http.StatusUnauthorized, apiError{Err: "Please log in."})
		return
	}

	id, err := strconv.ParseUint(mux.Vars(r)["id"], 10, 0)
	if err != nil || id == 0 {
		writeJson(w, http.StatusNotFound, apiError{Err: "Link not found."})
		return
	}

	var request updateLinkRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	if err := decoder.Decode(&request); err != nil {
		writeJson(w, http.StatusBadRequest, apiError{Err: "Request body must be valid JSON."})
		return
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeJson(w, http.StatusBadRequest, apiError{Err: "Request body must contain one JSON object."})
		return
	}

	note := strings.TrimSpace(request.Note)
	if note == "" {
		writeJson(w, http.StatusBadRequest, apiError{Err: "Note is required."})
		return
	}
	destination, err := ValidateAndNormalizeURL(request.Destination)
	if err != nil {
		writeJson(w, http.StatusBadRequest, apiError{Err: err.Error()})
		return
	}

	var link Link
	var version LinkVersion
	err = db.Transaction(func(tx *gorm.DB) error {
		var current Link
		result := tx.Where("id = ? AND user_id = ?", uint(id), userID).First(&current)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return errLinkNotFound
		}
		if result.Error != nil {
			return result.Error
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
			writeJson(w, http.StatusNotFound, apiError{Err: "Link not found."})
		case errors.Is(err, errSameDestination):
			writeJson(w, http.StatusBadRequest, apiError{Err: "Destination is unchanged."})
		default:
			writeJson(w, http.StatusInternalServerError, apiError{Err: "Could not update link."})
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
