package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/andrewpaige1/nodebook-api/auth"
	"github.com/andrewpaige1/nodebook-api/config"
	"github.com/andrewpaige1/nodebook-api/models"
)

func GetUsers(w http.ResponseWriter, r *http.Request) {
	var users []models.User

	config.Database.Find(&users)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(users)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func AddUser(w http.ResponseWriter, r *http.Request) {
	// Check if database connection is initialized
	if config.Database == nil {
		http.Error(w, "Database connection not initialized", http.StatusInternalServerError)
		log.Println("Error: Database connection is nil")
		return
	}

	// Read the raw body for logging
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		log.Println("Error reading request body:", err)
		return
	}

	// Log the raw input
	log.Println("Raw Input:", string(body))

	// Reset body for decoding
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	// Decode the user
	user := new(models.User)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println("Decoding error:", err)
		return
	}

	// Log the decoded user
	log.Printf("Decoded User: %+v\n", user)

	// Check if user already exists
	var existingUser models.User
	result := config.Database.Where("nickname = ?", user.Nickname).First(&existingUser)
	if result.Error == nil {
		// User already exists, return 200 status to prevent frontend errors
		tokenString, err := auth.CreateToken(existingUser.Nickname)
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			log.Println("Token generation error:", err)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "auth_token",
			Value:    tokenString,
			Path:     "/",
			HttpOnly: true,
			Domain:   ".mindthred.com",
			Secure:   true, // Use only in HTTPS
			SameSite: http.SameSiteLaxMode,
			MaxAge:   86400, // 24 hours
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "User already exists!",
		})
		log.Printf("User %s already exists\n", existingUser.Nickname)
		log.Println("Set-Cookie Header:", w.Header().Get("Set-Cookie"))

		return
	}

	// Perform the database creation with error checking
	result = config.Database.Create(&user)
	if result.Error != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		log.Println("Database creation error:", result.Error)
		return
	}

	// Check if any rows were actually affected
	if result.RowsAffected == 0 {
		http.Error(w, "No user was created", http.StatusInternalServerError)
		log.Println("No rows affected when creating user")
		return
	}

	tokenString, err := auth.CreateToken(user.Nickname)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		log.Println("Token generation error:", err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    tokenString,
		Path:     "/",
		Domain:   ".mindthred.com",
		HttpOnly: true,
		Secure:   true, // Use only in HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400, // 24 hours
	})

	// Prepare response with user and token
	response := map[string]interface{}{
		"user": user,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
	log.Println("User created successfully")
}
