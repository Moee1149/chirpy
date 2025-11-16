package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Moee1149/chirpy/internal/auth"
	"github.com/Moee1149/chirpy/internal/database"
)

type users struct {
	ID         string `json:"id"`
	EMAIL      string `json:"email"`
	UPDATED_AT string `json:"updated_at"`
	CREATED_AT string `json:"created_at"`
}

func (cfg *apiConfig) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	params := parameters{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}
	if params.Email == "" {
		responsdWithError(w, 400, "missing email field")
		return
	}
	if params.Password == "" {
		responsdWithError(w, 400, "missing password field")
		return
	}
	hashPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		log.Fatalf("Error hashing password: %v", err)
		responsdWithError(w, 500, "Error creating user")
		return
	}
	usersParams := database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashPassword,
	}
	user, err := cfg.dbQueries.CreateUser(r.Context(), usersParams)
	if err != nil {
		responsdWithError(w, 500, fmt.Sprintf("Error creating user %v", err))
		return
	}
	usr := users{
		ID:         user.ID.String(),
		EMAIL:      user.Email,
		CREATED_AT: user.CreatedAt.String(),
		UPDATED_AT: user.UpdatedAt.String(),
	}
	respondWithJSON(w, 201, usr)
}

func (cfg *apiConfig) handleUserLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	params := parameters{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		responsdWithError(w, 500, fmt.Sprintf("Error decoding body: %v", err))
		return
	}
	user, err := cfg.dbQueries.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			responsdWithError(w, 401, "Incorrect email or password")
			return
		}
		respondWithJSON(w, 500, fmt.Sprintf("Database Error: %v", err))
		return
	}
	passwordMatch, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil {
		respondWithJSON(w, 500, fmt.Sprintf("Database Error: %v", err))
		return
	}
	if !passwordMatch {
		responsdWithError(w, 401, "Incorrect email or password")
		return
	}
	usr := users{
		ID:         user.ID.String(),
		EMAIL:      user.Email,
		CREATED_AT: user.CreatedAt.String(),
		UPDATED_AT: user.UpdatedAt.String(),
	}
	respondWithJSON(w, 200, usr)
}
