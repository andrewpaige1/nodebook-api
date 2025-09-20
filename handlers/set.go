package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/andrewpaige1/nodebook-api/models"
	"github.com/andrewpaige1/nodebook-api/utils"
)

// /api/sets/{setID}

func (db *DBHandler) GetSetByID(w http.ResponseWriter, r *http.Request) {
	setID := r.PathValue("setID")
	var set models.FlashcardSet
	// Preload the User to access Auth0ID without a separate query
	if err := db.Preload("User").Preload("Flashcards").Where("public_id = ?", setID).First(&set).Error; err != nil {
		log.Printf("GetSetByID: Set not found for public_id=%s: %v", setID, err)
		http.Error(w, fmt.Sprintf("Set with ID %s not found", setID), http.StatusNotFound)
		return
	}

	// Lazy migration for public_id
	auth0ID, ok := utils.GetAuth0ID(r)
	isOwner := ok && set.User.Auth0ID == auth0ID

	type SetResponse struct {
		models.FlashcardSet
		IsOwner bool `json:"IsOwner"`
	}

	response := SetResponse{
		FlashcardSet: set,
		IsOwner:      isOwner,
	}

	if set.IsPublic {
		log.Printf("GetSetByID: Returning public set %s", setID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(response)
		return
	}

	if !ok || set.User.Auth0ID != auth0ID {
		log.Printf("GetSetByID: Forbidden access for set %s by auth0ID=%s", setID, auth0ID)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

// POST /api/set
func (db *DBHandler) CreateFlashCardSet(w http.ResponseWriter, r *http.Request) {
	// Get Auth0 ID from JWT/context
	auth0ID, ok := utils.GetAuth0ID(r)
	if !ok {
		log.Printf("CreateFlashCardSet: Unauthorized request")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Look up the user in your database
	var user models.User
	if err := db.Where("auth0_id = ?", auth0ID).First(&user).Error; err != nil {
		log.Printf("CreateFlashCardSet: User not found for auth0ID=%s: %v", auth0ID, err)
		// Avoid exposing internal IDs
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Decode the request body
	type CreateSetRequest struct {
		Title    string `json:"Title"`
		IsPublic bool   `json:"IsPublic"`
	}
	var req CreateSetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("CreateFlashCardSet: Invalid request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	publicID, err := gonanoid.New()
	if err != nil {
		log.Printf("CreateFlashCardSet: Failed to generate publicID: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create the set
	set := models.FlashcardSet{
		Title:    req.Title,
		UserID:   user.ID,
		IsPublic: req.IsPublic,
		PublicID: publicID,
	}

	// Save to DB
	if err := db.Create(&set).Error; err != nil {
		log.Printf("CreateFlashCardSet: Failed to create set: %v", err)
		http.Error(w, "Failed to create set", http.StatusInternalServerError)
		return
	}

	log.Printf("CreateFlashCardSet: Successfully created set with publicID=%s for userID=%d", publicID, user.ID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(set)
}

func (db *DBHandler) UpdateSetByID(w http.ResponseWriter, r *http.Request) {
	setID := r.PathValue("setID")
	auth0ID, ok := utils.GetAuth0ID(r)
	if !ok {
		log.Printf("UpdateSetByID: Unauthorized request")
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	var set models.FlashcardSet
	// Preload the User to get owner info in one query
	if err := db.Preload("User").Where("public_id = ?", setID).First(&set).Error; err != nil {
		log.Printf("UpdateSetByID: Set not found for public_id=%s: %v", setID, err)
		http.Error(w, fmt.Sprintf("Set with ID %s not found", setID), http.StatusNotFound)
		return
	}

	if auth0ID != set.User.Auth0ID {
		log.Printf("UpdateSetByID: Unauthorized update attempt by auth0ID=%s for setID=%s", auth0ID, setID)
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Decode the update request body
	type FlashcardUpdate struct {
		ID           uint   `json:"ID"`
		Term         string `json:"Term"`
		Solution     string `json:"Solution"`
		Concept      string `json:"Concept"`
		ShouldDelete bool   `json:"shouldDelete"`
		ShouldUpdate bool   `json:"shouldUpdate"`
		ShouldCreate bool   `json:"shouldCreate"`
	}
	type UpdateSetRequest struct {
		Title      *string            `json:"title,omitempty"`
		IsPublic   *bool              `json:"isPublic,omitempty"`
		Flashcards *[]FlashcardUpdate `json:"Flashcards,omitempty"`
	}

	var req UpdateSetRequest
	decoder := json.NewDecoder(r.Body)
	//decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		log.Printf("UpdateSetByID: Invalid request body: %v", err)
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	b, _ := json.MarshalIndent(req, "", "  ")
	fmt.Println(string(b))
	// Update fields if provided
	updated := false
	if req.Title != nil && set.Title != *req.Title {
		set.Title = *req.Title
		updated = true
	}
	if req.IsPublic != nil && set.IsPublic != *req.IsPublic {
		set.IsPublic = *req.IsPublic
		updated = true
	}

	// Support shouldDelete, shouldUpdate, shouldCreate flags for flashcards
	if req.Flashcards != nil {
		for _, fc := range *req.Flashcards {
			if fc.ID != 0 {
				if fc.ShouldDelete {
					// Delete flashcard
					if err := db.Where("id = ? AND set_id = ?", fc.ID, set.ID).Delete(&models.Flashcard{}).Error; err != nil {
						log.Printf("UpdateSetByID: Failed to delete flashcard id=%d for setID=%s: %v", fc.ID, setID, err)
					}
					continue
				}
				if fc.ShouldUpdate {
					// Update existing flashcard
					var flashcard models.Flashcard
					if err := db.Where("id = ? AND set_id = ?", fc.ID, set.ID).First(&flashcard).Error; err != nil {
						log.Printf("UpdateSetByID: Flashcard not found id=%d for setID=%s", fc.ID, setID)
						continue
					}
					flashcard.Term = fc.Term
					flashcard.Solution = fc.Solution
					flashcard.Concept = fc.Concept
					if err := db.Save(&flashcard).Error; err != nil {
						log.Printf("UpdateSetByID: Failed to update flashcard id=%d for setID=%s: %v", fc.ID, setID, err)
					}
				}
				// If neither shouldDelete nor shouldUpdate, do nothing
			} else {
				if fc.ShouldDelete {
					// No-op: cannot delete a non-existent flashcard
					continue
				}
				if fc.ShouldCreate {
					// Create new flashcard with nanoid public_id
					publicID, err := gonanoid.New()
					if err != nil {
						log.Printf("UpdateSetByID: Failed to generate public_id for new flashcard: %v", err)
						continue
					}
					newFlashcard := models.Flashcard{
						Term:     fc.Term,
						Solution: fc.Solution,
						Concept:  fc.Concept,
						SetID:    set.ID,
						PublicID: publicID,
					}
					if err := db.Create(&newFlashcard).Error; err != nil {
						log.Printf("UpdateSetByID: Failed to create new flashcard for setID=%s: %v", setID, err)
					}
				}
				// If neither shouldDelete nor shouldCreate, do nothing
			}
		}
	}

	// Save changes only if something changed
	fmt.Println("UPDATED", updated)
	if updated {
		if err := db.Save(&set).Error; err != nil {
			log.Printf("UpdateSetByID: Failed to update setID=%s: %v", setID, err)
			http.Error(w, fmt.Sprintf("Failed to update set with ID %s", setID), http.StatusInternalServerError)
			return
		}
	}

	log.Printf("UpdateSetByID: Successfully updated setID=%s", setID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(set)
}

func (db *DBHandler) DeleteSetByID(w http.ResponseWriter, r *http.Request) {
	setID := r.PathValue("setID")
	auth0ID, ok := utils.GetAuth0ID(r)
	if !ok {
		log.Printf("DeleteSetByID: Unauthorized request")
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	var set models.FlashcardSet
	// Preload User for the ownership check
	if err := db.Preload("User").Where("public_id = ?", setID).First(&set).Error; err != nil {
		http.Error(w, fmt.Sprintf("Set not found for public_id=%s", setID), http.StatusNotFound)
		return
	}

	if auth0ID != set.User.Auth0ID {
		log.Printf("DeleteSetByID: Unauthorized delete attempt by auth0ID=%s for setID=%s", auth0ID, setID)
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Associated flashcards should be deleted by a DB cascade or handled here
	result := db.Delete(&set)
	if result.Error != nil {
		log.Printf("DeleteSetByID: Failed to delete setID=%s: %v", setID, result.Error)
		http.Error(w, fmt.Sprintf("Failed to delete set with ID %s", setID), http.StatusInternalServerError)
		return
	}
	if result.RowsAffected == 0 {
		// This case should ideally be caught by the .First() call above, but it's good for safety.
		log.Printf("DeleteSetByID: Set not found for setID=%s during delete operation", setID)
		http.Error(w, fmt.Sprintf("Set not found for public_id=%s", setID), http.StatusNotFound)
		return
	}

	log.Printf("DeleteSetByID: Successfully deleted setID=%s", setID)
	w.WriteHeader(http.StatusNoContent)
}

func (db *DBHandler) GetSetsForUser(w http.ResponseWriter, r *http.Request) {
	nickname := r.PathValue("nickname")
	if nickname == "" {
		log.Printf("GetSetsForUser: Nickname is required")
		http.Error(w, "Nickname is required", http.StatusBadRequest)
		return
	}

	var user models.User
	if err := db.Where("nickname = ?", nickname).First(&user).Error; err != nil {
		log.Printf("GetSetsForUser: User not found for nickname=%s: %v", nickname, err)
		http.Error(w, fmt.Sprintf("User not found for nickname=%s", nickname), http.StatusNotFound)
		return
	}

	auth0ID, ok := utils.GetAuth0ID(r)

	var sets []models.FlashcardSet
	query := db.Preload("Flashcards").Where("user_id = ?", user.ID)

	if ok && user.Auth0ID == auth0ID {
		//log.Printf("GetSetsForUser: Returning all sets for owner userID=%d", user.ID)
	} else {
		query = query.Where("is_public = ?", true)
		log.Printf("GetSetsForUser: Returning public sets for userID=%d", user.ID)
	}

	if err := query.Find(&sets).Error; err != nil {
		log.Printf("GetSetsForUser: Failed to fetch sets for userID=%d: %v", user.ID, err)
		http.Error(w, fmt.Sprintf("Failed to fetch sets for user %s: %v", nickname, err), http.StatusInternalServerError)
		return
	}
	// Lazy migration for public_id on each set
	for i := range sets {
		if sets[i].PublicID == "" {
			newID, err := gonanoid.New()
			if err == nil {
				sets[i].PublicID = newID
				if err := db.Model(&sets[i]).Update("public_id", newID).Error; err != nil {
					log.Printf("GetSetsForUser: Failed to update public_id for setID=%d: %v", sets[i].ID, err)
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sets)
}
