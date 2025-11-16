package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Moee1149/chirpy/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handleCreateChirps(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body    string `json:"body"`
		User_Id string `json:"user_id"`
	}

	params := parameters{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		respondWithJSON(w, 500, fmt.Sprintf("Error decoding json: %v", err))
		return
	}
	if len(params.Body) > 140 {
		responsdWithError(w, 400, "The chirpy is too long")
		return
	}
	//check for profane words
	cleanedBody := validateBadWords(params.Body)
	userId, err := uuid.Parse(params.User_Id)
	if err != nil {
		responsdWithError(w, 400, "Invalid user_id format")
		return
	}
	chirpsParams := database.CreateChirpyParams{
		Body:   cleanedBody,
		UserID: userId,
	}

	chirp, err := cfg.dbQueries.CreateChirpy(r.Context(), chirpsParams)
	if err != nil {
		responsdWithError(w, 500, fmt.Sprintf("Error adding chirps: %v", err))
		return
	}
	resp := chirpSchema{
		ID:        chirp.ID.String(),
		CreateAt:  chirp.CreatedAt.UTC().Format("2006-01-0215:04:05Z"),
		UpdatedAt: chirp.UpdatedAt.UTC().Format("2006-01-0215:04:05Z"),
		Body:      chirp.Body,
		UserId:    chirp.UserID.String(),
	}
	respondWithJSON(w, 201, resp)
}

func (cfg *apiConfig) handleGetChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.dbQueries.GetAllChirps(r.Context())
	if err != nil {
		responsdWithError(w, 500, fmt.Sprintf("Error getting chirps: %v", err))
		return
	}
	respondWithJSON(w, 200, chirps)
}

func (cfg *apiConfig) handleGetChirpsById(w http.ResponseWriter, r *http.Request) {
	pathValue := r.PathValue("chirpID")
	chirpId, err := uuid.Parse(pathValue)
	if err != nil {
		responsdWithError(w, 400, "Invalid user_id format")
		return
	}
	chirp, err := cfg.dbQueries.GetChirpsById(r.Context(), chirpId)
	if err != nil {
		responsdWithError(w, 400, fmt.Sprintf("Error getting chirps: %v", err))
		return
	}
	resp := chirpSchema{
		ID:        chirp.ID.String(),
		CreateAt:  chirp.CreatedAt.UTC().Format("2006-01-0215:04:05Z"),
		UpdatedAt: chirp.UpdatedAt.UTC().Format("2006-01-0215:04:05Z"),
		Body:      chirp.Body,
		UserId:    chirp.UserID.String(),
	}
	respondWithJSON(w, 200, resp)
}
