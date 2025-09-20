package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/andrewpaige1/nodebook-api/models"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"gorm.io/gorm"

	"github.com/andrewpaige1/nodebook-api/utils"
)

type DBHandler struct {
	*gorm.DB
}

func (db *DBHandler) GetFlashcardByID(w http.ResponseWriter, r *http.Request) {

	flashcardID := r.PathValue("flashcardID")
	if flashcardID == "" {
		http.Error(w, "Flashcard ID is required", http.StatusBadRequest)
		return
	}

	var flashcard models.Flashcard

	result := db.Where("public_id = ?", flashcardID).First(&flashcard)

	if result.Error != nil {
		http.Error(w, "Flashcard set not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(flashcard); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

func (db *DBHandler) CreateFlashCard(w http.ResponseWriter, r *http.Request) {

	auth0ID, ok := utils.GetAuth0ID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var user models.User
	setID := r.PathValue("setID")
	var set models.FlashcardSet

	if err := db.Where("public_id = ?", setID).First(&set).Error; err != nil {
		http.Error(w, "Set not found", http.StatusNotFound)
		return
	}

	if err := db.Where("id = ?", set.UserID).First(&user).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if auth0ID != user.Auth0ID {
		http.Error(w, "Status Forbidden", http.StatusForbidden)
		return
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	type FlashcardRequestData struct {
		Term         string
		Solution     string
		LearningGoal string `json:"concept"`
	}

	var FlashcardRequest FlashcardRequestData

	err := decoder.Decode(&FlashcardRequest)

	if err != nil {
		http.Error(w, "Could not decode request", http.StatusInternalServerError)
	}

	publicID, err := gonanoid.New()

	if err != nil {
		http.Error(w, "Failed to generate ID", http.StatusInternalServerError)
		return
	}

	flashcard := models.Flashcard{
		Term:     FlashcardRequest.Term,
		Solution: FlashcardRequest.Solution,
		Concept:  FlashcardRequest.LearningGoal,
		PublicID: publicID,
		SetID:    set.ID,
	}

	if err := db.Create(&flashcard).Error; err != nil {

		http.Error(w, "Failed to create flashcard", http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(flashcard)
}

func (db *DBHandler) UpdateFlashCardByID(w http.ResponseWriter, r *http.Request) {
	setID := r.PathValue("setID")
	flashcardID := r.PathValue("flashcardID")

	auth0ID, ok := utils.GetAuth0ID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var set models.FlashcardSet
	if err := db.Where("public_id = ?", setID).First(&set).Error; err != nil {
		http.Error(w, "Set not found", http.StatusNotFound)
		return
	}

	var owner models.User
	if err := db.Where("id = ?", set.UserID).First(&owner).Error; err != nil {
		http.Error(w, "Owner not found", http.StatusInternalServerError)
		return
	}

	if owner.Auth0ID != auth0ID {
		http.Error(w, "Forbidden: You do not own this set", http.StatusForbidden)
		return
	}

	// Find the flashcard
	var flashcard models.Flashcard
	if err := db.Where("public_id = ? AND set_id = ?", flashcardID, set.ID).First(&flashcard).Error; err != nil {
		http.Error(w, "Flashcard not found", http.StatusNotFound)
		return
	}

	// Decode the update data
	type FlashcardUpdateRequest struct {
		Term     *string `json:"term,omitempty"`
		Solution *string `json:"solution,omitempty"`
		Concept  *string `json:"concept,omitempty"`
	}
	var req FlashcardUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update fields if provided
	if req.Term != nil {
		flashcard.Term = *req.Term
	}
	if req.Solution != nil {
		flashcard.Solution = *req.Solution
	}
	if req.Concept != nil {
		flashcard.Concept = *req.Concept
	}

	// Save the updated flashcard
	if err := db.Save(&flashcard).Error; err != nil {
		http.Error(w, "Failed to update flashcard", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(flashcard)
}

func (db *DBHandler) DeleteFlashCardByID(w http.ResponseWriter, r *http.Request) {
	setID := r.PathValue("setID")
	flashcardID := r.PathValue("flashcardID")

	auth0ID, ok := utils.GetAuth0ID(r)
	if !ok {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	var set models.FlashcardSet
	if err := db.Where("public_id = ?", setID).First(&set).Error; err != nil {
		http.Error(w, "Could not find flashcard set", http.StatusInternalServerError)
		return
	}

	var setOwner models.User
	if err := db.Where("id = ?", set.UserID).First(&setOwner).Error; err != nil {
		http.Error(w, "Could not find flashcard set owner", http.StatusInternalServerError)
		return
	}

	if auth0ID != setOwner.Auth0ID {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	result := db.Where("public_id = ?", flashcardID).Delete(&models.Flashcard{})
	if result.Error != nil {
		http.Error(w, "Failed to delete flashcard", http.StatusInternalServerError)
		return
	}
	if result.RowsAffected == 0 {
		http.Error(w, "Flashcard not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (db *DBHandler) GetFlashcardsForSet(w http.ResponseWriter, r *http.Request) {
	setID := r.PathValue("setID")

	var set models.FlashcardSet
	if err := db.Where("public_id = ?", setID).First(&set).Error; err != nil {
		http.Error(w, "Set not found", http.StatusNotFound)
		return
	}

	var user models.User
	if err := db.Where("id = ?", set.UserID).First(&user).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if !set.IsPublic {
		// If not public, check authentication and ownership
		auth0ID, ok := utils.GetAuth0ID(r)
		if !ok || user.Auth0ID != auth0ID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	var flashcards []models.Flashcard
	if err := db.Where("set_id = ?", set.ID).Find(&flashcards).Error; err != nil {
		http.Error(w, "Failed to fetch flashcards", http.StatusInternalServerError)
		return
	}

	// Lazy migration: generate and save public_id if missing
	for i := range flashcards {
		if flashcards[i].PublicID == "" {
			publicID, err := gonanoid.New()
			if err == nil {
				flashcards[i].PublicID = publicID
				db.Model(&flashcards[i]).Update("public_id", publicID)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(flashcards)
}
