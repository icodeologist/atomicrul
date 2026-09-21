package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type rollbackLinkRequest struct {
	Note string `json:"note"`
}

type rollbackLinkResponse struct {
	Link    linkResponse        `json:"link"`
	Version linkVersionResponse `json:"version"`
}

func RollbackLink(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	if r.Method != http.MethodPost {
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

	vars := mux.Vars(r)
	linkID, linkErr := strconv.ParseUint(vars["id"], 10, 0)
	versionID, versionErr := strconv.ParseUint(vars["versionID"], 10, 0)
	if linkErr != nil || versionErr != nil || linkID == 0 || versionID == 0 {
		writeJson(w, http.StatusNotFound, apiError{Err: "Link or version not found."})
		return
	}

	var request rollbackLinkRequest
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

	var link Link
	var version LinkVersion
	err := db.Transaction(func(tx *gorm.DB) error {
		var current Link
		result := tx.Where("id = ? AND user_id = ?", uint(linkID), userID).First(&current)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return errLinkNotFound
		}
		if result.Error != nil {
			return result.Error
		}

		var selected LinkVersion
		result = tx.Where("id = ? AND link_id = ?", uint(versionID), current.ID).First(&selected)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return errLinkNotFound
		}
		if result.Error != nil {
			return result.Error
		}

		version = LinkVersion{
			LinkID:      current.ID,
			Destination: selected.Destination,
			Note:        note,
		}
		if err := tx.Create(&version).Error; err != nil {
			return err
		}
		if err := tx.Model(&current).Update("destination", selected.Destination).Error; err != nil {
			return err
		}
		return tx.First(&link, current.ID).Error
	})
	if err != nil {
		if errors.Is(err, errLinkNotFound) {
			writeJson(w, http.StatusNotFound, apiError{Err: "Link or version not found."})
			return
		}
		writeJson(w, http.StatusInternalServerError, apiError{Err: "Could not roll back link."})
		return
	}

	writeJson(w, http.StatusOK, rollbackLinkResponse{
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
