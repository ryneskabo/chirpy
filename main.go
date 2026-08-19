package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"strings"
)

type apiConfig struct {
	fileserverHits atomic.Int32
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
	}
}

func (cfg *apiConfig) ResetMetricsHandler(writer http.ResponseWriter, req *http.Request) {
	writer.WriteHeader(http.StatusOK)
	cfg.fileserverHits.Store(0)
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

func ChirpValidationHandler(writer http.ResponseWriter, req *http.Request) {
	type chirp struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(req.Body)
	chirpReq := chirp{}
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

	type ValidResponse struct {
		CleanedBody string `json:"cleaned_body"`
	}
	respondWithJSON(writer, 200, ValidResponse{
		CleanedBody: cleanedChirp,
	})
}

func main() { 
	const port = "8080"

	serveMux := http.NewServeMux()
	strippedPathHandler := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	apiCfg := apiConfig{}
	serveMux.Handle("/app/", apiCfg.middlewareMetricsInc(strippedPathHandler))

	// Handles Readiness Endpoint, shows if server is up and running
	serveMux.HandleFunc("GET /api/healthz", ReadinessHandler)

	// Handles Hits Metrics Endpoint, shows how many hits since last reset
	serveMux.HandleFunc("GET /admin/metrics", apiCfg.MetricsHandler)

	// Handles reset Metrics Endpoint, resets the number of hits
	serveMux.HandleFunc("POST /admin/reset", apiCfg.ResetMetricsHandler)

	// Validates a Chirp (post)
	serveMux.HandleFunc("POST /api/validate_chirp", ChirpValidationHandler)

	// Initializes server struct and starts it using the serveMux handler
	server := http.Server{
		Handler: serveMux,
		Addr: ":" + port,
	}
	err := server.ListenAndServe()
	if err != nil {
		err := fmt.Errorf("Error in listen and serve %w: ",err)
		fmt.Println(err.Error())
	}
}
