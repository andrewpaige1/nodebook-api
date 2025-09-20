package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/andrewpaige1/nodebook-api/models"
	"github.com/andrewpaige1/nodebook-api/utils"
)

func (db *DBHandler) GetBlocksLeaderboard(w http.ResponseWriter, r *http.Request) {

	auth0ID, ok := utils.GetAuth0ID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}
	// fetches set using the public id
	publicSetID := r.PathValue("setID")
	if publicSetID == "" {
		http.Error(w, "set ID is required", http.StatusBadRequest)
		return
	}

	var set models.FlashcardSet

	result := db.Preload("User").Where("public_id = ?", publicSetID).First(&set)
	if result.Error != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if !set.IsPublic && set.User.Auth0ID != auth0ID {
		http.Error(w, "Set is not public", http.StatusForbidden)
		return
	}

	var blockScores []models.BlocksScore
	result = db.Preload("User").Where("flashcard_set_id = ?", set.ID).Order("time_seconds asc").Find(&blockScores)
	if result.Error != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blockScores)
}

func (db *DBHandler) CreateBlockScore(w http.ResponseWriter, r *http.Request) {
	auth0ID, ok := utils.GetAuth0ID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// --- (This part is correct) ---
	publicSetID := r.PathValue("setID")
	if publicSetID == "" {
		http.Error(w, "set ID is required", http.StatusBadRequest)
		return
	}

	var set models.FlashcardSet
	result := db.Preload("User").Where("public_id = ?", publicSetID).First(&set)
	if result.Error != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if !set.IsPublic && set.User.Auth0ID != auth0ID {
		http.Error(w, "Set is not public", http.StatusForbidden)
		return
	}

	// --- FIX STARTS HERE ---

	// NEW: Find the internal user based on the Auth0 ID from the token
	var user models.User
	if err := db.Where("auth0_id = ?", auth0ID).First(&user).Error; err != nil {
		// This handles the case where a user exists in Auth0 but not in your DB yet.
		// You might have a different way of handling this.
		http.Error(w, "User not found in database", http.StatusNotFound)
		return
	}

	type BlockPayload struct {
		CorrectAttempts uint
		TotalAttempts   uint
		Time            uint
	}

	var blockReq BlockPayload

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&blockReq); err != nil {
		http.Error(w, "Invalid request body: %v", http.StatusBadRequest)
		return
	}

	blockScore := models.BlocksScore{
		UserID:          user.ID,
		FlashcardSetID:  set.ID,
		TimeSeconds:     int(blockReq.Time),
		CorrectAttempts: int(blockReq.CorrectAttempts),
		TotalAttempts:   int(blockReq.TotalAttempts),
	}

	if err := db.Create(&blockScore).Error; err != nil {
		http.Error(w, "Failed to create block score", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(blockScore)
}
