package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/andrewpaige1/nodebook-api/config"
	"github.com/andrewpaige1/nodebook-api/models"
)

type MindMapStateResponse struct {
	ID          uint                       `json:"id"`
	Title       string                     `json:"title"`
	IsPublic    bool                       `json:"isPublic"`
	SetID       uint                       `json:"setID"`
	Connections []models.MindMapConnection `json:"connections"`
	NodeLayouts []models.MindMapNodeLayout `json:"nodeLayouts"`
}

func GetMindMapState(w http.ResponseWriter, r *http.Request) {
	// Extract nickname and title from URL
	nickname := r.PathValue("nickname")
	title := r.PathValue("title")

	if nickname == "" || title == "" {
		http.Error(w, "Nickname and mind map title are required", http.StatusBadRequest)
		return
	}

	// Find the user by nickname
	var user models.User
	if err := config.Database.Where("nickname = ?", nickname).First(&user).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Find the mind map and eager load all related data
	var mindMap models.MindMap
	if err := config.Database.
		Preload("Connections").
		Preload("Connections.Source"). // Load source flashcard data
		Preload("Connections.Target"). // Load target flashcard data
		Where("title = ? AND user_id = ?", title, user.ID).
		First(&mindMap).Error; err != nil {
		http.Error(w, "Mind map not found", http.StatusNotFound)
		return
	}

	// Load node layouts separately since they're not directly included in the MindMap model
	var nodeLayouts []models.MindMapNodeLayout
	if err := config.Database.
		Where("mind_map_id = ?", mindMap.ID).
		Find(&nodeLayouts).Error; err != nil {
		http.Error(w, "Error loading node layouts", http.StatusInternalServerError)
		return
	}

	// Construct the response
	response := MindMapStateResponse{
		ID:          mindMap.ID,
		Title:       mindMap.Title,
		IsPublic:    mindMap.IsPublic,
		SetID:       mindMap.SetID,
		Connections: mindMap.Connections,
		NodeLayouts: nodeLayouts,
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Encode and send the response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}

func UpdateConnections(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		MindMapID         uint   `json:"mindMapID"`
		Nickname          string `json:"nickname"`
		SourceID          uint   `json:"source"`
		TargetID          uint   `json:"target"`
		RelationshipLabel string `json:"relationshipLabel"`
	}

	// Decode the request payload
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		log.Printf("Received data: %+v", requestData)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if requestData.Nickname == "" || requestData.MindMapID == 0 || requestData.SourceID == 0 || requestData.TargetID == 0 {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	// Find the user
	var user models.User
	if err := config.Database.Where("nickname = ?", requestData.Nickname).First(&user).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Verify the mind map exists and belongs to the user
	var mindMap models.MindMap
	if err := config.Database.Where("id = ? AND user_id = ?", requestData.MindMapID, user.ID).First(&mindMap).Error; err != nil {
		http.Error(w, "Mind map not found or unauthorized", http.StatusNotFound)
		return
	}

	// Check if the connection already exists
	var connection models.MindMapConnection
	err := config.Database.Where(
		"mind_map_id = ? AND source_id = ? AND target_id = ?",
		requestData.MindMapID,
		requestData.SourceID,
		requestData.TargetID,
	).First(&connection).Error

	if err != nil {
		// Connection doesn't exist, create new one
		newConnection := models.MindMapConnection{
			MindMapID:    requestData.MindMapID,
			SourceID:     requestData.SourceID,
			TargetID:     requestData.TargetID,
			Relationship: requestData.RelationshipLabel,
		}

		if err := config.Database.Create(&newConnection).Error; err != nil {
			http.Error(w, "Error creating the connection", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(newConnection)
		return
	}

	// Connection exists, update it
	if err := config.Database.Model(&connection).Where(
		"mind_map_id = ? AND source_id = ? AND target_id = ?",
		requestData.MindMapID,
		requestData.SourceID,
		requestData.TargetID,
	).Update("relationship", requestData.RelationshipLabel).Error; err != nil {
		http.Error(w, "Error updating the connection", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(connection)
}

func UpdateNodeLayout(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		MindMapID uint   `json:"mindMapID"`
		Nickname  string `json:"nickname"`
		Nodes     []struct {
			FlashcardID uint    `json:"flashcardID"`
			XPosition   float64 `json:"xPosition"`
			YPosition   float64 `json:"yPosition"`
			Data        string  `json:"data"`
		} `json:"nodes"`
	}

	// Decode the request payload
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		log.Printf("Received data: %+v", requestData)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if requestData.Nickname == "" || requestData.MindMapID == 0 || len(requestData.Nodes) == 0 {
		http.Error(w, "Required fields are missing", http.StatusBadRequest)
		return
	}

	// Find the user
	var user models.User
	if err := config.Database.Where("nickname = ?", requestData.Nickname).First(&user).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Verify the mind map exists and belongs to the user
	var mindMap models.MindMap
	if err := config.Database.Where("id = ? AND user_id = ?", requestData.MindMapID, user.ID).First(&mindMap).Error; err != nil {
		http.Error(w, "Mind map not found or unauthorized", http.StatusNotFound)
		return
	}

	// Start a transaction
	tx := config.Database.Begin()
	if tx.Error != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}

	updatedLayouts := make([]models.MindMapNodeLayout, 0, len(requestData.Nodes))

	for _, node := range requestData.Nodes {
		// Check if the node layout already exists
		var nodeLayout models.MindMapNodeLayout
		err := tx.Where(
			"mind_map_id = ? AND flashcard_id = ?",
			requestData.MindMapID,
			node.FlashcardID,
		).First(&nodeLayout).Error

		if err != nil {
			// Node layout doesn't exist, create new one
			newNodeLayout := models.MindMapNodeLayout{
				MindMapID:   requestData.MindMapID,
				FlashcardID: node.FlashcardID,
				XPosition:   node.XPosition,
				YPosition:   node.YPosition,
				Data:        node.Data,
			}

			if err := tx.Create(&newNodeLayout).Error; err != nil {
				tx.Rollback()
				http.Error(w, "Error creating node layout", http.StatusInternalServerError)
				return
			}

			updatedLayouts = append(updatedLayouts, newNodeLayout)
		} else {
			// Node layout exists, update it
			updates := map[string]interface{}{
				"x_position": node.XPosition,
				"y_position": node.YPosition,
				"data":       node.Data,
			}

			if err := tx.Model(&nodeLayout).Updates(updates).Error; err != nil {
				tx.Rollback()
				http.Error(w, "Error updating node layout", http.StatusInternalServerError)
				return
			}

			// Reload the updated node layout
			if err := tx.First(&nodeLayout, nodeLayout.ID).Error; err != nil {
				tx.Rollback()
				http.Error(w, "Error retrieving updated node layout", http.StatusInternalServerError)
				return
			}

			updatedLayouts = append(updatedLayouts, nodeLayout)
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedLayouts)
}

func DeleteMindMap(w http.ResponseWriter, r *http.Request) {

	var requestData struct {
		MindMapID string `json:"title"`
		Nickname  string `json:"nickname"`
	}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&requestData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var user models.User
	if err := config.Database.Where("nickname = ?", requestData.Nickname).First(&user).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var mindMap models.MindMap
	err := config.Database.Where("id = ?", requestData.MindMapID).Delete(&mindMap).Error
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func CheckDup(w http.ResponseWriter, r *http.Request) {

	var requestData struct {
		Title    string `json:"title"`
		Nickname string `json:"nickname"` // Auth0 nickname
		SetName  string `json:"setName"`
	}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&requestData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var user models.User
	if err := config.Database.Where("nickname = ?", requestData.Nickname).First(&user).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Verify the flashcard set exists and belongs to the user
	var flashcardSet models.FlashcardSet
	if err := config.Database.Where("title = ? AND user_id = ?", requestData.SetName, user.ID).First(&flashcardSet).Error; err != nil {
		http.Error(w, "Flashcard set not found or unauthorized", http.StatusNotFound)
		return
	}

	var mindMap models.MindMap
	err := config.Database.Where("title = ? AND set_id = ? AND user_id = ?", requestData.Title, flashcardSet.ID, user.ID).First(&mindMap).Error
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "success",
		})
		return
	}

	response := map[string]interface{}{
		"message": "Mind Map with this title already exists",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusConflict)
	json.NewEncoder(w).Encode(response)
}

func CreateMindMap(w http.ResponseWriter, r *http.Request) {
	// Request struct to match the expected input
	var requestData struct {
		Title       string                     `json:"title"`
		Nickname    string                     `json:"nickname"` // Auth0 nickname
		Data        string                     `json:"data"`
		SetID       uint                       `json:"setID"`
		IsPublic    bool                       `json:"isPublic"`
		Connections []models.MindMapConnection `json:"connections"`
		NodeLayouts []models.MindMapNodeLayout `json:"nodeLayouts"`
	}

	// Decode the incoming JSON
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&requestData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if requestData.Title == "" || requestData.Nickname == "" || requestData.SetID == 0 {
		http.Error(w, "Title, nickname, and setID are required", http.StatusBadRequest)
		return
	}

	// Find user by nickname
	var user models.User
	if err := config.Database.Where("nickname = ?", requestData.Nickname).First(&user).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Verify the flashcard set exists and belongs to the user
	var flashcardSet models.FlashcardSet
	if err := config.Database.Where("id = ? AND user_id = ?", requestData.SetID, user.ID).First(&flashcardSet).Error; err != nil {
		http.Error(w, "Flashcard set not found or unauthorized", http.StatusNotFound)
		return
	}

	// Create the mind map
	mindMap := models.MindMap{
		Title:    requestData.Title,
		SetID:    requestData.SetID,
		UserID:   user.ID,
		IsPublic: requestData.IsPublic,
	}

	// Start a database transaction
	tx := config.Database.Begin()
	if tx.Error != nil {
		http.Error(w, "Could not begin transaction", http.StatusInternalServerError)
		return
	}

	// Create the mind map
	if err := tx.Create(&mindMap).Error; err != nil {
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create connections
	for i := range requestData.Connections {
		// Set the MindMapID for each connection
		requestData.Connections[i].MindMapID = mindMap.ID

		// Validate the connection
		if requestData.Connections[i].SourceID == 0 || requestData.Connections[i].TargetID == 0 {
			tx.Rollback()
			http.Error(w, "Each connection must have a source and target flashcard", http.StatusBadRequest)
			return
		}

		// Verify that both source and target flashcards exist in the set
		var count int64
		tx.Model(&models.Flashcard{}).Where(
			"id IN (?, ?) AND set_id = ?",
			requestData.Connections[i].SourceID,
			requestData.Connections[i].TargetID,
			flashcardSet.ID,
		).Count(&count)

		if count != 2 {
			tx.Rollback()
			http.Error(w, "Invalid source or target flashcard", http.StatusBadRequest)
			return
		}

		// Create the connection
		if err := tx.Create(&requestData.Connections[i]).Error; err != nil {
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Create node layouts
	for i := range requestData.NodeLayouts {
		// Set the MindMapID for each layout
		requestData.NodeLayouts[i].MindMapID = mindMap.ID

		// Validate the layout
		if requestData.NodeLayouts[i].FlashcardID == 0 {
			tx.Rollback()
			http.Error(w, "Each node layout must reference a flashcard", http.StatusBadRequest)
			return
		}

		// Verify that the flashcard exists in the set
		var exists bool
		tx.Model(&models.Flashcard{}).
			Where("id = ? AND set_id = ?", requestData.NodeLayouts[i].FlashcardID, flashcardSet.ID).
			Select("1").
			Scan(&exists)

		if !exists {
			tx.Rollback()
			http.Error(w, "Invalid flashcard reference in node layout", http.StatusBadRequest)
			return
		}

		// Create the node layout
		if err := tx.Create(&requestData.NodeLayouts[i]).Error; err != nil {
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		http.Error(w, "Could not commit transaction", http.StatusInternalServerError)
		return
	}

	// Preload associated data for the response
	if err := config.Database.Preload("Connections").Preload("Connections.Source").Preload("Connections.Target").First(&mindMap, mindMap.ID).Error; err != nil {
		http.Error(w, "Error retrieving created mind map", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(mindMap)
}

func GetMindMapForSets(w http.ResponseWriter, r *http.Request) {
	nickname := r.PathValue("nickname")
	setName := r.PathValue("setName")

	// Find user
	var user models.User
	if err := config.Database.Where("nickname = ?", nickname).First(&user).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Find the flashcard set by title and user
	var set models.FlashcardSet
	if err := config.Database.Where("title = ? AND user_id = ?", setName, user.ID).First(&set).Error; err != nil {
		http.Error(w, "Set not found", http.StatusNotFound)
		return
	}

	var mindMaps []models.MindMap
	if err := config.Database.Where("set_id = ?", set.ID).Find(&mindMaps).Error; err != nil {
		http.Error(w, "Error retrieving mind maps", http.StatusInternalServerError)
		return
	}

	fmt.Println(mindMaps)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mindMaps)

}

func GetMindMap(w http.ResponseWriter, r *http.Request) {
	nickname := r.PathValue("nickname")
	setName := r.PathValue("setName")

	// Find user
	var user models.User
	if err := config.Database.Where("nickname = ?", nickname).First(&user).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Find the flashcard set by title and user
	var set models.FlashcardSet
	if err := config.Database.Where("title = ? AND user_id = ?", setName, user.ID).First(&set).Error; err != nil {
		http.Error(w, "Set not found", http.StatusNotFound)
		return
	}

	// Attempt to load the mind map
	var mindmap models.MindMap
	if err := config.Database.
		Preload("Nodes").
		Preload("Edges").
		Where("set_id = ? AND user_id = ?", set.ID, user.ID).
		First(&mindmap).Error; err != nil {
		// If we can’t find it, return 404
		http.Error(w, "Mindmap not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mindmap)
}
