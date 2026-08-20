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
	"github.com/joho/godotenv"
	"github.com/google/uuid"
	"time"
	_ "github.com/lib/pq"
	"github.com/ryneskabo/chirpy/internal/database"
	"github.com/ryneskabo/chirpy/internal/auth"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	queries database.Queries
	platform string
}
type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type ValidChirpResponse struct {
	ID uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body string `json:"body"`
	UserID uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func ReadinessHandler(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(http.StatusOK)

	// Writes out OK to body of response
	_, err := writer.Write([]byte("OK"))
	if err != nil {
		err := fmt.Errorf("Error in readiness handler body write: %w", err)
		fmt.Println(err.Error())
		return
	}
}

func (cfg *apiConfig) MetricsHandler(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Set("Content-Type", "text/html")
	writer.WriteHeader(http.StatusOK)
	hitsString := fmt.Sprintf(`
		<html>
			<body>
				<h1>Welcome, Chirpy Admin</h1>
				<p>Chirpy has been visited %d times!</p>
			</body>
		</html>`, cfg.fileserverHits.Load())
	_, err := writer.Write([]byte(hitsString))
	if err != nil {
		err := fmt.Errorf("Error in hits handler write: %w", err)
		fmt.Println(err)
		return
	}
}

func (cfg *apiConfig) ResetHandler(writer http.ResponseWriter, req *http.Request) {
	if cfg.platform != "dev" {
		respondWithError(writer, 403, "Forbidden")
		return
	}
	cfg.fileserverHits.Store(0)
	err := cfg.queries.DeleteAllUsers(req.Context())
	if err != nil {
		respondWithError(writer, 500, "Could not delete users")
		return
	}
	writer.WriteHeader(http.StatusOK)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type errorResponse struct {
		 ErrorResponse string `json:"error"`
	}
	errBody := errorResponse{
		ErrorResponse: msg,
	}
	respondWithJSON(w, code, errBody)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func replaceProfaneWords(chirp string) string {
	profaneWords := []string {
		"kerfuffle",
		"sharbert",
		"fornax",
	}
	words := strings.Split(chirp, " ")
	
	for i, word := range words {
		for _, profaneWord := range profaneWords {
			if strings.ToLower(word) == strings.ToLower(profaneWord) {
				words[i] = "****"
			}
		}
	}
	return strings.Join(words, " ")
}

func (cfg *apiConfig) ChirpHandler(writer http.ResponseWriter, req *http.Request) {
	type chirpRequest struct {
		Body string `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	decoder := json.NewDecoder(req.Body)
	chirpReq := chirpRequest{}
	err := decoder.Decode(&chirpReq)
	if err != nil {
		respondWithError(writer, 400, "Request did not match expected constraints")
		return
	}
	if len(chirpReq.Body) > 140 {
		respondWithError(writer, 400, "Chirp is too long")
		return
	}
	cleanedChirp := replaceProfaneWords(chirpReq.Body)

	requestedChirp, err := cfg.queries.CreateChirp(req.Context(), database.CreateChirpParams{
		Body: cleanedChirp,
		UserID: uuid.NullUUID{UUID: chirpReq.UserID, Valid: true},
	})
	if err != nil {
		respondWithError(writer, 500, "could not create chirp")
		return
	}
	validChirp := ValidChirpResponse {
		ID: requestedChirp.ID,
		CreatedAt: requestedChirp.CreatedAt,
		UpdatedAt: requestedChirp.UpdatedAt,
		Body: cleanedChirp,
		UserID: requestedChirp.UserID.UUID,
	}
	respondWithJSON(writer, 201, validChirp)
}

func (cfg *apiConfig) getAllChirpsHandler(writer http.ResponseWriter, req *http.Request) {
	chirps, err := cfg.queries.GetAllChirps(req.Context())
	if err != nil {
		respondWithError(writer, 500, "Internal server error")
		log.Printf("Couldn't retrieve all chirps error: %v", err)
		return
	}

	jsonableChirps := []ValidChirpResponse{}
	for _, chirp := range chirps {
		jsonableChirp := ValidChirpResponse {
			ID: chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body: chirp.Body,
			UserID: chirp.UserID.UUID,
		}
		jsonableChirps = append(jsonableChirps, jsonableChirp)
	}
	respondWithJSON(writer, 200, jsonableChirps)
}

func (cfg *apiConfig) getOneChirpHandler(writer http.ResponseWriter, req *http.Request) {
	stringIdOfChirp := req.PathValue("chirpID")
	idOfChirp, err := uuid.Parse(stringIdOfChirp)
	if err != nil {
		respondWithError(writer, 400, "Invalid UUID for chirp")
		log.Printf("Error parsing UUID: %v", err)
		return
	}
	chirp, err := cfg.queries.GetOneChirp(req.Context(), idOfChirp)
	if err != nil {
		respondWithError(writer, 404, "Chirp not found")
		return
	}
	jsonableChirp := ValidChirpResponse{
		ID: chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body: chirp.Body,
		UserID: chirp.UserID.UUID,
	}
	respondWithJSON(writer, 200, jsonableChirp)
}

func (cfg *apiConfig) userCreationHandler(writer http.ResponseWriter, req *http.Request) {
	type UserCreationRequest struct {
		Email string `json:"email"`
		Password string `json:"password"`
	} 
	decoder := json.NewDecoder(req.Body)
	userReq := UserCreationRequest{}
	err := decoder.Decode(&userReq)
	if err != nil {
		respondWithError(writer, 400, "Did not match constraints: expected email and password")
		return
	}
	if userReq.Password == "" {
		respondWithError(writer, 400, "No Password set, please set a password")
		return
	}

	hashedPassword, err := auth.HashPassword(userReq.Password)
	if err != nil {
		respondWithError(writer, 500, "Internal Server Error")
		log.Print("Error hashing password")
		return
	}
	user, err := cfg.queries.CreateUser(req.Context(), database.CreateUserParams{
		Email: userReq.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		respondWithError(writer, 500, "Could not create user")
		return
	}
	createdUser := User {
		ID: user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email: user.Email,
	}
	respondWithJSON(writer, 201, createdUser)
}

func (cfg *apiConfig) loginHandler(writer http.ResponseWriter, req *http.Request) {
	type LoginRequest struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(req.Body)
	loginReq := LoginRequest{}
	err := decoder.Decode(&loginReq)
	if err != nil {
		respondWithError(writer, 500, "Internal Server Error")
		log.Printf("Error decoding login request: %s", err)
		return
	}

	hashedPassword, err := cfg.queries.GetPasswordByEmail(req.Context(), loginReq.Email)
	if err != nil {
		respondWithError(writer, 401, "Incorrect email or password")
		log.Printf("Couldn't retrieve password from email.\nemail: %s\nerror: %v", loginReq.Email, err)
		return
	}
	match, err := auth.CheckPasswordHash(loginReq.Password, hashedPassword)
	if err != nil {
		respondWithError(writer, 500, "Internal Server error")
		log.Printf("Error checking password hash: %v", err)
		return
	}
	if !match {
		respondWithError(writer, 401, "Incorrect email or password")
		log.Printf("Non matching password from email: %s", loginReq.Email)
		return
	}
	user, err := cfg.queries.GetUserByEmail(req.Context(), loginReq.Email)
	jsonableUser := User {
		Email: user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		ID: user.ID,
	}
	respondWithJSON(writer, 200, jsonableUser)
}

func main() { 
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	requestPlatform := os.Getenv("PLATFORM")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Printf("Error in opening database: %v", err)
	}
	dbQueries := database.New(db)
	const port = "8080"

	serveMux := http.NewServeMux()
	strippedPathHandler := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	apiCfg := apiConfig{
		queries: *dbQueries,
		platform: requestPlatform,
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

	// Initializes server struct and starts it using the serveMux handler
	server := http.Server{
		Handler: serveMux,
		Addr: ":" + port,
	}
	err = server.ListenAndServe()
	if err != nil {
		err := fmt.Errorf("Error in listen and serve %w: ",err)
		fmt.Println(err.Error())
	}
}
