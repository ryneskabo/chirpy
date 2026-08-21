package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/ryneskabo/chirpy/internal/auth"
)


func (cfg *apiConfig) RevokeTokenHandler(writer http.ResponseWriter, req *http.Request) {
	tokenToBeRevoked, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(writer, 400, fmt.Sprintf("%v", err))
	}
	
	rows, err := cfg.queries.RevokeRefreshToken(req.Context(), tokenToBeRevoked)
	if err != nil {
		respondWithError(writer, 500, "Internal Server error")
		log.Printf("error revoking refresh token: %v", err)
		return
	} 
	if rows == 0 {
		respondWithError(writer, 404, "refresh token not found")
		return
	}
	respondWithJSON(writer, 204, nil)
}
