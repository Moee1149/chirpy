package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/Moee1149/chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileServerHits atomic.Int32
	dbQueries      *database.Queries
}

func (cfg *apiConfig) middlewareMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

type chirpSchema struct {
	ID        string `json:"id"`
	CreateAt  string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Body      string `json:"body"`
	UserId    string `json:"UserId"`
}

func main() {
	godotenv.Load()

	dbUrl := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatalf("Error Connection Database: %v", err)
	}
	dbQueries := database.New(db)
	apiConfig := apiConfig{
		dbQueries: dbQueries,
	}
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	mux.Handle("/app/", http.StripPrefix("/app", apiConfig.middlewareMetrics(http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("GET /admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		hits := apiConfig.fileServerHits.Load()
		html := fmt.Sprintf("<html> <body> <h1>Welcome, Chirpy Admin</h1> <p>Chirpy has been visited %d times!</p> </body> </html>", hits)
		w.Write([]byte(html))
	})
	mux.HandleFunc("POST /admin/reset", func(w http.ResponseWriter, r *http.Request) {
		apiConfig.fileServerHits.Store(0)
		if strings.ToLower(platform) != "dev" {
			responsdWithError(w, 403, "Forbidden")
			return
		}
		apiConfig.dbQueries.DropTable(r.Context())
		type response struct {
			Message string `json:"message"`
		}
		msg := response{
			Message: "Delete User Table",
		}
		respondWithJSON(w, 200, msg)
	})
	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Email string `json:"email"`
		}

		params := parameters{}
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&params); err != nil {
			log.Printf("Error decoding parameters: %s", err)
			w.WriteHeader(500)
			return
		}
		if params.Email == "" {
			responsdWithError(w, 400, "missing email filed")
			return
		}
		user, err := apiConfig.dbQueries.CreateUser(r.Context(), params.Email)
		if err != nil {
			responsdWithError(w, 500, fmt.Sprintf("Error creating user %v", err))
			return
		}
		type users struct {
			ID         string `json:"id"`
			EMAIL      string `json:"email"`
			UPDATED_AT string `json:"updated_at"`
			CREATED_AT string `json:"created_at"`
		}
		usr := users{
			ID:         user.ID.String(),
			EMAIL:      user.Email,
			CREATED_AT: user.CreatedAt.String(),
			UPDATED_AT: user.UpdatedAt.String(),
		}
		respondWithJSON(w, 201, usr)
	})
	mux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, r *http.Request) {
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

		chirp, err := apiConfig.dbQueries.CreateChirpy(r.Context(), chirpsParams)
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
	})
	mux.HandleFunc("GET /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		chirps, err := apiConfig.dbQueries.GetAllChirps(r.Context())
		if err != nil {
			responsdWithError(w, 500, fmt.Sprintf("Error getting chirps: %v", err))
			return
		}
		respondWithJSON(w, 200, chirps)
	})
	mux.HandleFunc("GET /api/chirps/{chirpID}", func(w http.ResponseWriter, r *http.Request) {
		pathValue := r.PathValue("chirpID")
		chirpId, err := uuid.Parse(pathValue)
		if err != nil {
			responsdWithError(w, 400, "Invalid user_id format")
			return
		}
		chirp, err := apiConfig.dbQueries.GetChirpsById(r.Context(), chirpId)
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
	})

	fmt.Printf("Server running on port %v\n", server.Addr)

	err = server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
