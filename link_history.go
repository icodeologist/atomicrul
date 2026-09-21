package main

import (
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

	link, findErr := findOwnedLink(db, userID, uint(id))
	if isLinkNotFound(findErr) {
		writeAPIError(w, http.StatusNotFound, "Link not found.")
		return
	}
	if findErr != nil {
		writeAPIError(w, http.StatusInternalServerError, "Could not load link history.")
		return
	}

	var versions []LinkVersion
	result := db.Where("link_id = ?", link.ID).Order("created_at DESC, id DESC").Find(&versions)
	if result.Error != nil {
		writeAPIError(w, http.StatusInternalServerError, "Could not load link history.")
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
