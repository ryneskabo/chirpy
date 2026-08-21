package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/ryneskabo/chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	queries        database.Queries
	platform       string
	jwtSecret      string
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	requestPlatform := os.Getenv("PLATFORM")
	jwtSecret := os.Getenv("JWT_SECRET")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Printf("Error in opening database: %v", err)
	}
	dbQueries := database.New(db)
	const port = "8080"

	serveMux := http.NewServeMux()
	strippedPathHandler := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	apiCfg := apiConfig{
		queries:   *dbQueries,
		platform:  requestPlatform,
		jwtSecret: jwtSecret,
	}
	serveMux.Handle("/app/", apiCfg.middlewareMetricsInc(strippedPathHandler))

	// Handles Readiness Endpoint, shows if server is up and running
	serveMux.HandleFunc("GET /api/healthz", ReadinessHandler)

	// Handles Hits Metrics Endpoint, shows how many hits since last reset
	serveMux.HandleFunc("GET /admin/metrics", apiCfg.MetricsHandler)

	// Handles reset Metrics Endpoint, resets the number of hits
	serveMux.HandleFunc("POST /admin/reset", apiCfg.ResetHandler)

	// Validates and adds Chirp (post) to database
	serveMux.HandleFunc("POST /api/chirps", apiCfg.ChirpHandler)

	// Creates a user
	serveMux.HandleFunc("POST /api/users", apiCfg.userCreationHandler)

	// Gets all chirps
	serveMux.HandleFunc("GET /api/chirps", apiCfg.getAllChirpsHandler)

	// Gets One chirp
	serveMux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.getOneChirpHandler)

	// Handles Logins
	serveMux.HandleFunc("POST /api/login", apiCfg.loginHandler)

	// Handles Refreshing JWT
	serveMux.HandleFunc("POST /api/refresh", apiCfg.RefreshTokenHandler)

	// Handles Revoking Refresh Token
	serveMux.HandleFunc("POST /api/revoke", apiCfg.RevokeTokenHandler)

	// Initializes server struct and starts it using the serveMux handler
	server := http.Server{
		Handler: serveMux,
		Addr:    ":" + port,
	}
	err = server.ListenAndServe()
	if err != nil {
		err := fmt.Errorf("Error in listen and serve %w: ", err)
		fmt.Println(err.Error())
	}
}
