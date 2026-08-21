package main

import (
	"log"
	"net/http"
	"time"

	"github.com/ryneskabo/chirpy/internal/auth"
)


func (cfg *apiConfig) RefreshTokenHandler(writer http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(writer, 401, "unauthorized")
		return
	}

	dbToken, err := cfg.queries.GetRefreshToken(req.Context(), token)
	if err != nil {
		respondWithError(writer, 401, "unauthorized")
		return
	}
	if dbToken.RevokedAt.Valid {
		respondWithError(writer, 401, "unauthorized")
		return
	} 

	type RefreshTokenResponse struct {
		Token string `json:"token"`
	}
	newToken, err := auth.MakeJWT(dbToken.UserID.UUID, cfg.jwtSecret, time.Hour)
	if err != nil {
		respondWithError(writer, 500, "Internal Server Error")
		log.Printf("couldn't make jwt: %v", err)
		return
	}
	
	newTokenResponse := RefreshTokenResponse{
		Token: newToken,
	}
	respondWithJSON(writer, 200, newTokenResponse)
}
