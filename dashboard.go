package main

import (
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"
)

type dashboardLinkResponse struct {
	ID           uint      `json:"id"`
	Title        string    `json:"title"`
	Code         string    `json:"code"`
	ShortURL     string    `json:"short_url"`
	Destination  string    `json:"destination"`
	Active       bool      `json:"active"`
	Clicks       int       `json:"clicks"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	VersionCount int       `json:"version_count"`
}

type dashboardResponse struct {
	Links []dashboardLinkResponse `json:"links"`
}

func ShowUserDashBoard(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	if r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		return
	}

	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}

	var links []Link
	result := db.Preload("Versions").
		Where("user_id = ?", userID).
		Order("updated_at DESC, id DESC").
		Find(&links)
	if result.Error != nil {
		writeAPIError(w, http.StatusInternalServerError, "Could not load dashboard links.")
		return
	}

	response := dashboardResponse{Links: make([]dashboardLinkResponse, 0, len(links))}
	for _, link := range links {
		response.Links = append(response.Links, dashboardLinkResponse{
			ID:           link.ID,
			Title:        link.Title,
			Code:         link.Code,
			ShortURL:     dashboardShortURL(r, link.Code),
			Destination:  link.Destination,
			Active:       link.Active,
			Clicks:       link.Clicks,
			CreatedAt:    link.CreatedAt,
			UpdatedAt:    link.UpdatedAt,
			VersionCount: len(link.Versions),
		})
	}

	writeJson(w, http.StatusOK, response)
}

func dashboardShortURL(r *http.Request, code string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host
	if host == "" {
		host = "localhost:3000"
	}
	return fmt.Sprintf("%s://%s/%s", scheme, host, code)
}
