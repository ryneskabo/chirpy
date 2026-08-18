package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
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

func (cfg *apiConfig) HitsHandler(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
	hitsString := fmt.Sprintf("Hits: %d", cfg.fileserverHits.Load())
	_, err := writer.Write([]byte(hitsString))
	if err != nil {
		err := fmt.Errorf("Error in hits handler write: %w", err)
		fmt.Println(err)
	}
}

func (cfg *apiConfig) ResetHitsHandler(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
	cfg.fileserverHits.Store(0)
}

func main() { 
	const port = "8080"

	serveMux := http.NewServeMux()
	strippedPathHandler := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	apiCfg := apiConfig{}
	serveMux.Handle("/app/", apiCfg.middlewareMetricsInc(strippedPathHandler))

	// Handles Readiness Endpoint, shows if server is up and running
	serveMux.HandleFunc("/healthz", ReadinessHandler)

	// Handles Hits Metrics Endpoint, shows how many hits since last reset
	serveMux.HandleFunc("/metrics", apiCfg.HitsHandler)

	// Handles reset Metrics Endpoint, resets the number of hits
	serveMux.HandleFunc("/reset", apiCfg.ResetHitsHandler)

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
