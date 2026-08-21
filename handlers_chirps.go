package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
	"sort"

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

func (cfg *apiConfig) getAuthorChirps(writer http.ResponseWriter, req *http.Request, authorID string) []ValidChirpResponse {
	authorUUID, err := uuid.Parse(authorID)
	if err != nil {
		respondWithError(writer, 400, "invalid uuid")
	}
	chirps, err := cfg.queries.GetAuthorChirps(req.Context(), uuid.NullUUID{
		UUID: authorUUID,
		Valid: true,
	})
	if err == sql.ErrNoRows {
		respondWithError(writer, 404, "author doesn't exist or has no chirps")
		return nil
	}
	if err != nil {
		respondWithError(writer, 500, "internal server error")
		log.Printf("couldn't get author chirps: %v", err)
		return nil
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
	return jsonableChirps
}

func (cfg *apiConfig) getAllChirpsHandler(writer http.ResponseWriter, req *http.Request) {
	authorID := req.URL.Query().Get("author_id")
	sortParam := req.URL.Query().Get("sort")
	if sortParam == "" {
		sortParam = "asc"
	}
	if authorID != "" {
		chirps := cfg.getAuthorChirps(writer, req, authorID)
		if sortParam != "desc" {
			respondWithJSON(writer, 200, chirps)
			return
		}
		sort.Slice(chirps, func(i, j int) bool { return chirps[i].CreatedAt.After(chirps[j].CreatedAt) })
		respondWithJSON(writer, 200, chirps)
		return
	}
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
	if sortParam != "desc" {
		respondWithJSON(writer, 200, jsonableChirps)
		return
	}
	sort.Slice(jsonableChirps, func(i, j int) bool { return jsonableChirps[i].CreatedAt.After(jsonableChirps[j].CreatedAt) })
	respondWithJSON(writer, 200, jsonableChirps)
}

func (cfg *apiConfig) getOneChirpHandler(writer http.ResponseWriter, req *http.Request) {
	stringIdOfChirp := req.PathValue("chirpID")
	idOfChirp, err := uuid.Parse(stringIdOfChirp)
	if err != nil {
		respondWithError(writer, 400, "invalid uuid for chirp")
		log.Printf("error parsing uuid: %v", err)
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

func (cfg *apiConfig) DeleteChirpHandler(writer http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(writer, 401, "not found")
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(writer, 401, "not found")
		return
	}

	stringIdOfChirp := req.PathValue("chirpID")
	idOfChirp, err := uuid.Parse(stringIdOfChirp)
	if err != nil {
		respondWithError(writer, 400, "invalid uuid for chirp")
		log.Printf("Error parsing UUID: %v", err)
		return
	}

	chirp, err := cfg.queries.GetOneChirp(req.Context(), idOfChirp)
	if err == sql.ErrNoRows {
		respondWithError(writer, 404, "not found")
		return
	}
	if err != nil {
		respondWithError(writer, 500, "internal server error")
		return
	}
	if userID != chirp.UserID.UUID {
		respondWithError(writer, 403, "forbidden")
		return
	}
	err = cfg.queries.DeleteAChirp(req.Context(), chirp.ID)
	if err != nil {
		respondWithError(writer, 500, "internal server error")
		log.Printf("couldn't delete chirp: %v", err)
		return
	}
	respondWithJSON(writer, 204, nil)
}
