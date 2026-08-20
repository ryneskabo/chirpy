package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ryneskabo/chirpy/internal/auth"
	"github.com/ryneskabo/chirpy/internal/database"
)

type ValidChirpResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func replaceProfaneWords(chirp string) string {
	profaneWords := []string{
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
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	decoder := json.NewDecoder(req.Body)
	chirpReq := chirpRequest{}
	err := decoder.Decode(&chirpReq)
	if err != nil {
		respondWithError(writer, 400, "Request did not match expected constraints")
		return
	}
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(writer, 401, "unauthorized: bearer token not present or unable to be parsed")
		return
	}
	userId, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(writer, 401, "unauthorized")
		return
	}

	if len(chirpReq.Body) > 140 {
		respondWithError(writer, 400, "Chirp is too long")
		return
	}
	cleanedChirp := replaceProfaneWords(chirpReq.Body)

	requestedChirp, err := cfg.queries.CreateChirp(req.Context(), database.CreateChirpParams{
		Body:   cleanedChirp,
		UserID: uuid.NullUUID{UUID: userId, Valid: true},
	})
	if err != nil {
		respondWithError(writer, 500, "could not create chirp")
		return
	}
	validChirp := ValidChirpResponse{
		ID:        requestedChirp.ID,
		CreatedAt: requestedChirp.CreatedAt,
		UpdatedAt: requestedChirp.UpdatedAt,
		Body:      cleanedChirp,
		UserID:    userId,
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
		jsonableChirp := ValidChirpResponse{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID.UUID,
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
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID.UUID,
	}
	respondWithJSON(writer, 200, jsonableChirp)
}
