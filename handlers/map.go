package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/andrewpaige1/nodebook-api/models"
	"github.com/andrewpaige1/nodebook-api/utils"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

// GET /api/sets/{setID}/mindmaps
func (db *DBHandler) GetMindMapsForSet(w http.ResponseWriter, r *http.Request) {
	setID := r.PathValue("setID")
	if setID == "" {
		http.Error(w, "Set ID is required", http.StatusBadRequest)
		return
	}

	var set models.FlashcardSet
	if err := db.Preload("User").Where("public_id = ?", setID).First(&set).Error; err != nil {
		http.Error(w, "Set not found", http.StatusNotFound)
		return
	}

	auth0ID, ok := utils.GetAuth0ID(r)
	var mindMaps []models.MindMap

	query := db.Preload("Connections").Preload("Connections.Source").Preload("Connections.Target").Where("set_id = ?", set.ID)

	if !(ok && set.User.Auth0ID == auth0ID) {
		// Only show public mindmaps if not owner
		query = query.Where("is_public = ?", true)
	}
	if err := query.Find(&mindMaps).Error; err != nil {
		http.Error(w, "Failed to fetch mind maps", http.StatusInternalServerError)
		return
	}

	type MindMapFull struct {
		models.MindMap
		NodeLayouts []models.MindMapNodeLayout `json:"nodeLayouts"`
	}

	var result []MindMapFull
	for i := range mindMaps {
		var layouts []models.MindMapNodeLayout
		if err := db.Where("mind_map_id = ?", mindMaps[i].ID).Find(&layouts).Error; err != nil {

			http.Error(w, "Failed to fetch node layouts", http.StatusInternalServerError)
			return
		}
		result = append(result, MindMapFull{
			MindMap:     mindMaps[i],
			NodeLayouts: layouts,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GET /api/sets/{setID}/mindmaps/{mindMapID}
func (db *DBHandler) GetMindMapByID(w http.ResponseWriter, r *http.Request) {
	setID := r.PathValue("setID")
	mindMapID := r.PathValue("mindMapID")
	if setID == "" || mindMapID == "" {
		http.Error(w, "Set ID and MindMap ID are required", http.StatusBadRequest)
		return
	}
	var set models.FlashcardSet
	if err := db.Preload("User").Where("public_id = ?", setID).First(&set).Error; err != nil {
		http.Error(w, "Set not found", http.StatusNotFound)
		return
	}
	var mindMap models.MindMap
	if err := db.Preload("Connections").Preload("Connections.Source").Preload("Connections.Target").Where("public_id = ? AND set_id = ?", mindMapID, set.ID).First(&mindMap).Error; err != nil {
		http.Error(w, "MindMap not found in set", http.StatusNotFound)
		return
	}
	var nodeLayouts []models.MindMapNodeLayout
	if err := db.Where("mind_map_id = ?", mindMap.ID).Find(&nodeLayouts).Error; err != nil {
		http.Error(w, "Failed to fetch node layouts", http.StatusInternalServerError)
		return
	}
	type MindMapFull struct {
		models.MindMap
		NodeLayouts []models.MindMapNodeLayout `json:"nodeLayouts"`
	}
	response := MindMapFull{
		MindMap:     mindMap,
		NodeLayouts: nodeLayouts,
	}

	if mindMap.IsPublic {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}
	// Private: check authentication and ownership
	auth0ID, ok := utils.GetAuth0ID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if set.User.Auth0ID != auth0ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// POST /api/sets/{setID}/mindmaps
func (db *DBHandler) CreateMindMap(w http.ResponseWriter, r *http.Request) {
	auth0ID, ok := utils.GetAuth0ID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	setID := r.PathValue("setID")
	if setID == "" {
		http.Error(w, "Set ID is required", http.StatusBadRequest)
		return
	}
	var req struct {
		Title       string                      `json:"Title"`
		Connections *[]models.MindMapConnection `json:"Connections,omitempty"`
		NodeLayouts *[]models.MindMapNodeLayout `json:"NodeLayouts,omitempty"`
		IsPublic    bool                        `json:"IsPublic"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	var set models.FlashcardSet
	if err := db.Preload("User").Where("public_id = ?", setID).First(&set).Error; err != nil {
		http.Error(w, "Set not found", http.StatusNotFound)
		return
	}
	if set.User.Auth0ID != auth0ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	publicID, err := gonanoid.New()
	if err != nil {
		http.Error(w, "Failed to generate public_id", http.StatusInternalServerError)
		return
	}
	mindMap := models.MindMap{
		Title:    req.Title,
		SetID:    set.ID,
		UserID:   set.UserID,
		IsPublic: req.IsPublic,
		PublicID: publicID,
	}

	tx := db.Begin()
	if err := tx.Create(&mindMap).Error; err != nil {
		tx.Rollback()
		http.Error(w, "Failed to create mind map", http.StatusInternalServerError)
		return
	}

	// Optionally skip connection creation on initial mindmap creation
	// (No connections will be created)

	// Node layouts are now created via a separate endpoint after mind map creation

	if err := tx.Commit().Error; err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	// Preload associated data for the response
	if err := db.Preload("Connections").Preload("Connections.Source").Preload("Connections.Target").First(&mindMap, mindMap.ID).Error; err != nil {
		http.Error(w, "Error retrieving created mind map", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(mindMap)
}

// PUT /api/sets/{setID}/mindmaps/{mindMapID}
func (db *DBHandler) UpdateMindMapByID(w http.ResponseWriter, r *http.Request) {
	setID := r.PathValue("setID")
	mindMapID := r.PathValue("mindMapID")
	auth0ID, ok := utils.GetAuth0ID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if setID == "" || mindMapID == "" {
		http.Error(w, "Set ID and MindMap ID are required", http.StatusBadRequest)
		return
	}
	var set models.FlashcardSet
	if err := db.Where("public_id = ?", setID).First(&set).Error; err != nil {
		http.Error(w, "Set not found", http.StatusNotFound)
		return
	}
	var mindMap models.MindMap
	if err := db.Preload("Connections").Preload("Connections.Source").Preload("Connections.Target").Where("public_id = ? AND set_id = ?", mindMapID, set.ID).First(&mindMap).Error; err != nil {
		http.Error(w, "MindMap not found in set", http.StatusNotFound)
		return
	}
	if set.User.Auth0ID != auth0ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		Title    *string `json:"title,omitempty"`
		IsPublic *bool   `json:"isPublic,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	updated := false
	if req.Title != nil && mindMap.Title != *req.Title {
		mindMap.Title = *req.Title
		updated = true
	}
	if req.IsPublic != nil && mindMap.IsPublic != *req.IsPublic {
		mindMap.IsPublic = *req.IsPublic
		updated = true
	}
	if updated {
		if err := db.Save(&mindMap).Error; err != nil {
			http.Error(w, "Failed to update mind map", http.StatusInternalServerError)
			return
		}
	}
	// Reload connections and node layouts for response
	if err := db.Preload("Connections").Preload("Connections.Source").Preload("Connections.Target").Where("id = ? AND set_id = ?", mindMap.ID, set.ID).First(&mindMap).Error; err != nil {
		http.Error(w, "Failed to reload mind map", http.StatusInternalServerError)
		return
	}
	var nodeLayouts []models.MindMapNodeLayout
	if err := db.Where("mind_map_id = ?", mindMap.ID).Find(&nodeLayouts).Error; err != nil {
		http.Error(w, "Failed to fetch node layouts", http.StatusInternalServerError)
		return
	}
	type MindMapFull struct {
		models.MindMap
		NodeLayouts []models.MindMapNodeLayout `json:"nodeLayouts"`
	}
	response := MindMapFull{
		MindMap:     mindMap,
		NodeLayouts: nodeLayouts,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// DELETE /api/sets/{setID}/mindmaps/{mindMapID}
func (db *DBHandler) DeleteMindMapByID(w http.ResponseWriter, r *http.Request) {
	setID := r.PathValue("setID")
	mindMapID := r.PathValue("mindMapID")
	auth0ID, ok := utils.GetAuth0ID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if setID == "" || mindMapID == "" {
		http.Error(w, "Set ID and MindMap ID are required", http.StatusBadRequest)
		return
	}
	var set models.FlashcardSet
	if err := db.Where("public_id = ?", setID).First(&set).Error; err != nil {
		http.Error(w, "Set not found", http.StatusNotFound)
		return
	}
	var mindMap models.MindMap
	if err := db.Preload("Connections").Preload("Connections.Source").Preload("Connections.Target").Where("public_id = ? AND set_id = ?", mindMapID, set.ID).First(&mindMap).Error; err != nil {
		http.Error(w, "MindMap not found in set", http.StatusNotFound)
		return
	}

	var user models.User
	if err := db.Where("id = ?", set.UserID).First(&user).Error; err != nil {
		http.Error(w, "User not found for mindmap", http.StatusNotFound)
	}
	if user.Auth0ID != auth0ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if err := db.Delete(&mindMap).Error; err != nil {
		http.Error(w, "Failed to delete mind map", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/users/{nickname}/mindmaps
func (db *DBHandler) GetMindMapsForUser(w http.ResponseWriter, r *http.Request) {
	nickname := r.PathValue("nickname")
	if nickname == "" {
		http.Error(w, "Nickname is required", http.StatusBadRequest)
		return
	}

	var user models.User
	if err := db.Where("nickname = ?", nickname).First(&user).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	auth0ID, ok := utils.GetAuth0ID(r)

	var mindMaps []models.MindMap
	query := db.Preload("Connections").Preload("Connections.Source").Preload("Connections.Target")
	query = query.Where("user_id = ?", user.ID)

	if !(ok && user.Auth0ID == auth0ID) {
		query = query.Where("is_public = ?", true)
	}

	if err := query.Find(&mindMaps).Error; err != nil {
		http.Error(w, "Failed to fetch mind maps", http.StatusInternalServerError)
		return
	}

	type MindMapFull struct {
		models.MindMap
		NodeLayouts []models.MindMapNodeLayout `json:"nodeLayouts"`
	}
	var result []MindMapFull

	for i := range mindMaps {
		// Lazy migration for public_id
		if mindMaps[i].PublicID == "" {
			newID, err := gonanoid.New()
			if err == nil {
				mindMaps[i].PublicID = newID
				db.Model(&mindMaps[i]).Update("public_id", newID)
			}
		}
		// Load node layouts
		var layouts []models.MindMapNodeLayout
		if err := db.Where("mind_map_id = ?", mindMaps[i].ID).Find(&layouts).Error; err != nil {
			http.Error(w, "Failed to fetch node layouts", http.StatusInternalServerError)
			return
		}
		result = append(result, MindMapFull{
			MindMap:     mindMaps[i],
			NodeLayouts: layouts,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// PUT /api/sets/{setID}/mindmaps/{mindMapID}/layouts
func (db *DBHandler) UpdateMindMapLayouts(w http.ResponseWriter, r *http.Request) {
	setID := r.PathValue("setID")
	mindMapID := r.PathValue("mindMapID")
	auth0ID, ok := utils.GetAuth0ID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if setID == "" || mindMapID == "" {
		http.Error(w, "Set ID and MindMap ID are required", http.StatusBadRequest)
		return
	}
	var set models.FlashcardSet
	if err := db.Where("public_id = ?", setID).First(&set).Error; err != nil {
		http.Error(w, "Set not found", http.StatusNotFound)
		return
	}
	var mindMap models.MindMap
	if err := db.Where("public_id = ? AND set_id = ?", mindMapID, set.ID).First(&mindMap).Error; err != nil {
		http.Error(w, "MindMap not found in set", http.StatusNotFound)
		return
	}

	var user models.User
	if err := db.Where("id = ?", set.UserID).First(&user).Error; err != nil {
		http.Error(w, "User not found for mind map", http.StatusNotFound)
		return
	}

	if user.Auth0ID != auth0ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	// Request struct matching frontend payload
	type NodeLayoutRequest struct {
		SetID       string
		PublicID    string
		FlashcardID uint    `json:"FlashcardID"`
		XPosition   float64 `json:"XPosition"`
		YPosition   float64 `json:"YPosition"`
		Data        string  `json:"Data"`
	}
	var reqLayouts []NodeLayoutRequest
	if err := json.NewDecoder(r.Body).Decode(&reqLayouts); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	//fmt.Println("req layouts", reqLayouts)
	// Delete existing layouts for this mindmap
	if err := db.Where("mind_map_id = ?", mindMap.ID).Delete(&models.MindMapNodeLayout{}).Error; err != nil {
		http.Error(w, "Failed to clear old node layouts", http.StatusInternalServerError)
		return
	}
	// Insert new layouts
	for _, req := range reqLayouts {
		layout := models.MindMapNodeLayout{
			MindMapID:   mindMap.ID,
			FlashcardID: req.FlashcardID,
			XPosition:   req.XPosition,
			YPosition:   req.YPosition,
			Data:        req.Data,
		}
		if err := db.Create(&layout).Error; err != nil {
			http.Error(w, "Failed to save node layout", http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// PUT /api/sets/{setID}/mindmaps/{mindMapID}/connections
func (db *DBHandler) UpdateMindMapConnections(w http.ResponseWriter, r *http.Request) {
	setID := r.PathValue("setID")
	mindMapID := r.PathValue("mindMapID")
	auth0ID, ok := utils.GetAuth0ID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if setID == "" || mindMapID == "" {
		http.Error(w, "Set ID and MindMap ID are required", http.StatusBadRequest)
		return
	}
	var set models.FlashcardSet
	if err := db.Preload("User").Where("public_id = ?", setID).First(&set).Error; err != nil {
		http.Error(w, "Set not found", http.StatusNotFound)
		return
	}
	var mindMap models.MindMap
	if err := db.Where("public_id = ? AND set_id = ?", mindMapID, set.ID).First(&mindMap).Error; err != nil {
		http.Error(w, "MindMap not found in set", http.StatusNotFound)
		return
	}
	if set.User.Auth0ID != auth0ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	var connections []models.MindMapConnection
	if err := json.NewDecoder(r.Body).Decode(&connections); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	// Delete existing connections for this mindmap
	if err := db.Where("mind_map_id = ?", mindMap.ID).Delete(&models.MindMapConnection{}).Error; err != nil {
		http.Error(w, "Failed to clear old connections", http.StatusInternalServerError)
		return
	}
	// Insert new connections
	for _, conn := range connections {
		conn.MindMapID = mindMap.ID
		if err := db.Create(&conn).Error; err != nil {
			http.Error(w, "Failed to save connection", http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}
