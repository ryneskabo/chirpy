package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/ryneskabo/chirpy/internal/auth"
	"github.com/ryneskabo/chirpy/internal/database"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
}

func (cfg *apiConfig) userCreationHandler(writer http.ResponseWriter, req *http.Request) {
	type UserCreationRequest struct {
		Email    string `json:"email"`
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
		Email:          userReq.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		respondWithError(writer, 500, "Could not create user")
		return
	}
	createdUser := User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	}
	respondWithJSON(writer, 201, createdUser)
}

func (cfg *apiConfig) loginHandler(writer http.ResponseWriter, req *http.Request) {
	type LoginRequest struct {
		Email            string `json:"email"`
		Password         string `json:"password"`
		ExpiresInSeconds *int   `json:"expires_in_seconds"`
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
	if err != nil {
		respondWithError(writer, 500, "Internal Server error")
		log.Printf("failed to get user from email %v", err)
		return
	}

	var secondsUntilExpiration int
	if loginReq.ExpiresInSeconds == nil {
		secondsUntilExpiration = 3600
	} else if *loginReq.ExpiresInSeconds > 3600 {
		secondsUntilExpiration = 3600
	} else {
		secondsUntilExpiration = *loginReq.ExpiresInSeconds
	}
	jwt, err := auth.MakeJWT(user.ID, cfg.jwtSecret, time.Second*time.Duration(secondsUntilExpiration))
	if err != nil {
		respondWithError(writer, 500, "Internal Server error")
		log.Printf("couldn't make jwt: %v", err)
		return
	}
	jsonableUser := User{
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		ID:        user.ID,
		Token:     jwt,
	}
	respondWithJSON(writer, 200, jsonableUser)
}
