package main

import (
	"fmt"
	"net/http"
	"strings"
)

func (cfg *apiConfig) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	hits := cfg.fileServerHits.Load()
	html := fmt.Sprintf("<html> <body> <h1>Welcome, Chirpy Admin</h1> <p>Chirpy has been visited %d times!</p> </body> </html>", hits)
	w.Write([]byte(html))
}

func (cfg *apiConfig) handleReset(platform string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits.Store(0)
		if strings.ToLower(platform) != "dev" {
			responsdWithError(w, 403, "Forbidden")
			return
		}
		cfg.dbQueries.DropTable(r.Context())
		type response struct {
			Message string `json:"message"`
		}
		msg := response{
			Message: "Delete User Table",
		}
		respondWithJSON(w, 200, msg)
	}
}
