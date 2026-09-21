package main

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type linkHistoryVersionResponse struct {
	ID          uint      `json:"id"`
	Destination string    `json:"destination"`
	Note        string    `json:"note"`
	CreatedAt   time.Time `json:"created_at"`
	Active      bool      `json:"active"`
}

type linkHistoryResponse struct {
	LinkID   uint                         `json:"link_id"`
	Versions []linkHistoryVersionResponse `json:"versions"`
}

func GetLinkHistory(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	if r.Method != http.MethodGet {
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

	var link Link
	result := db.Select("id", "destination").Where("id = ? AND user_id = ?", uint(id), userID).First(&link)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		writeJson(w, http.StatusNotFound, apiError{Err: "Link not found."})
		return
	}
	if result.Error != nil {
		writeJson(w, http.StatusInternalServerError, apiError{Err: "Could not load link history."})
		return
	}

	var versions []LinkVersion
	result = db.Where("link_id = ?", link.ID).Order("created_at DESC, id DESC").Find(&versions)
	if result.Error != nil {
		writeJson(w, http.StatusInternalServerError, apiError{Err: "Could not load link history."})
		return
	}

	responseVersions := make([]linkHistoryVersionResponse, 0, len(versions))
	activeMarked := false
	for _, version := range versions {
		active := !activeMarked && version.Destination == link.Destination
		if active {
			activeMarked = true
		}
		responseVersions = append(responseVersions, linkHistoryVersionResponse{
			ID:          version.ID,
			Destination: version.Destination,
			Note:        version.Note,
			CreatedAt:   version.CreatedAt,
			Active:      active,
		})
	}

	writeJson(w, http.StatusOK, linkHistoryResponse{
		LinkID:   link.ID,
		Versions: responseVersions,
	})
}
