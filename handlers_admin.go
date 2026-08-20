package main

import (
	"fmt"
	"net/http"
)

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
