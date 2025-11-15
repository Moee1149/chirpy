package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"

	"github.com/Moee1149/chirpy/internal/database"
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

func main() {
	godotenv.Load()

	dbUrl := os.Getenv("DB_URL")
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
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hits reset" + strconv.FormatInt(int64(apiConfig.fileServerHits.Load()), 10)))
	})

	mux.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {
		type paramters struct {
			Body string `json:"body"`
		}
		params := paramters{}
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&params)
		if err != nil {
			log.Printf("Error decoding parameters: %s", err)
			w.WriteHeader(500)
			return
		}
		if len(params.Body) > 140 {
			responsdWithError(w, 400, "The chirpy is too long")
			return
		}
		type validResponse struct {
			CleanedBody string `json:"cleaned_body"`
		}
		//check for profane words
		cleanedBody := validateBadWords(params.Body)
		respBody := validResponse{
			CleanedBody: cleanedBody,
		}
		respondWithJSON(w, 200, respBody)
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

	fmt.Printf("Server running on port %v\n", server.Addr)

	err = server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
