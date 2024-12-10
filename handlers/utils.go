package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/andrewpaige1/nodebook-api/config"
	"github.com/andrewpaige1/nodebook-api/models"
)

func CreateSetWithCards(w http.ResponseWriter, r *http.Request) {

	// Simplified request struct
	var requestData struct {
		Name     string             `json:"name"`
		Nickname string             `json:"nickname"` // Auth0 nickname
		Cards    []models.Flashcard `json:"cards"`
	}

	// Decode the incoming JSON
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&requestData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if requestData.Name == "" || requestData.Nickname == "" {
		http.Error(w, "Set name and nickname are required", http.StatusBadRequest)
		return
	}

	// Find user by nickname
	var user models.User
	if err := config.Database.Where("nickname = ?", requestData.Nickname).First(&user).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Create the flashcard set
	flashcardSet := models.FlashcardSet{
		Title:  requestData.Name,
		UserID: user.ID,
	}

	// Start a database transaction
	tx := config.Database.Begin()
	if tx.Error != nil {
		http.Error(w, "Could not begin transaction", http.StatusInternalServerError)
		return
	}

	// Create the flashcard set
	if err := tx.Create(&flashcardSet).Error; err != nil {
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Associate and create flashcards
	for i := range requestData.Cards {
		// Set the SetID for each flashcard
		requestData.Cards[i].SetID = flashcardSet.ID

		// Validate each flashcard
		if requestData.Cards[i].Term == "" || requestData.Cards[i].Solution == "" {
			tx.Rollback()
			http.Error(w, "Each flashcard must have a term and solution", http.StatusBadRequest)
			return
		}

		// Create the flashcard
		if err := tx.Create(&requestData.Cards[i]).Error; err != nil {
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		http.Error(w, "Could not commit transaction", http.StatusInternalServerError)
		return
	}

	// Preload associated data for the response
	if err := config.Database.Preload("Flashcards").First(&flashcardSet, flashcardSet.ID).Error; err != nil {
		http.Error(w, "Error retrieving created set", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(flashcardSet)
}
