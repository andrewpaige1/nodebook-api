package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/andrewpaige1/nodebook-api/config"
	"github.com/andrewpaige1/nodebook-api/models"
)

func GetUserFlashcardSets(w http.ResponseWriter, r *http.Request) {
	// Extract nickname from URL
	nickname := r.PathValue("nickname")
	if nickname == "" {
		http.Error(w, "Nickname is required", http.StatusBadRequest)
		return
	}

	// Find the user by nickname
	var user models.User
	if err := config.Database.Where("nickname = ?", nickname).First(&user).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Find all flashcard sets for this user
	var flashcardSets []models.FlashcardSet
	setsResult := config.Database.
		Where("user_id = ?", user.ID).
		Preload("Flashcards"). // Optional: preload flashcards if needed
		Find(&flashcardSets)

	if setsResult.Error != nil {
		http.Error(w, setsResult.Error.Error(), http.StatusInternalServerError)
		return
	}

	// If no sets found, return an empty array instead of null
	if len(flashcardSets) == 0 {
		flashcardSets = []models.FlashcardSet{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(flashcardSets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func GetFlashcardSet(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid flashcard set ID", http.StatusBadRequest)
		return
	}

	var flashcardSet models.FlashcardSet
	result := config.Database.
		Preload("User").
		Preload("Flashcards").
		First(&flashcardSet, uint(id))

	if result.Error != nil {
		http.Error(w, "Flashcard set not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(flashcardSet)
}

func GetUserFlashcardSetByTitle(w http.ResponseWriter, r *http.Request) {

	nickname := r.PathValue("nickname")
	setTitle := r.PathValue("title")

	if nickname == "" || setTitle == "" {
		http.Error(w, "Both nickname and set title are required", http.StatusBadRequest)
		return
	}

	// Find the user by nickname
	var user models.User
	if err := config.Database.Where("nickname = ?", nickname).First(&user).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Find the specific flashcard set for this user
	var flashcardSet models.FlashcardSet
	result := config.Database.
		Where("user_id = ? AND title = ?", user.ID, setTitle).
		Preload("Flashcards").
		First(&flashcardSet)

	if result.Error != nil {
		http.Error(w, "Flashcard set not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(flashcardSet); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
