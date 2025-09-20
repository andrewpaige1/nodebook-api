package main

import (
	"log"
	"net/http"
	"os"

	"github.com/andrewpaige1/nodebook-api/config"
	"github.com/andrewpaige1/nodebook-api/handlers"
	"github.com/andrewpaige1/nodebook-api/middleware"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func init() {
	// Load .env file if not in production environment
	if os.Getenv("RAILWAY_ENVIRONMENT_NAME") == "" {
		err := godotenv.Load()
		if err != nil {
			log.Printf("Warning: .env file not found, environment variables might not be loaded: %v", err)
		}
	}
}

func main() {
	// Initialize database connection
	config.Connect()
	authMiddleware := middleware.EnsureValidToken()

	DBHandler := &handlers.DBHandler{DB: config.Database}
	mux := http.NewServeMux()

	// Set
	mux.HandleFunc("GET /api/sets/{setID}", DBHandler.GetSetByID)
	mux.HandleFunc("POST /api/sets", middleware.SyncUserMiddleware(DBHandler.CreateFlashCardSet))
	mux.HandleFunc("PUT /api/sets/{setID}", middleware.SyncUserMiddleware(DBHandler.UpdateSetByID))
	mux.HandleFunc("DELETE /api/sets/{setID}", middleware.SyncUserMiddleware(DBHandler.DeleteSetByID))

	// User sets
	mux.HandleFunc("GET /api/users/{nickname}/sets", DBHandler.GetSetsForUser)
	mux.HandleFunc("GET /api/users/{nickname}/mindmaps", DBHandler.GetMindMapsForUser)

	// Mind map
	mux.HandleFunc("GET /api/sets/{setID}/mindmaps/{mindMapID}", DBHandler.GetMindMapByID)
	mux.HandleFunc("GET /api/sets/{setID}/mindmaps", DBHandler.GetMindMapsForSet)
	mux.HandleFunc("POST /api/sets/{setID}/mindmaps", middleware.SyncUserMiddleware(DBHandler.CreateMindMap))
	mux.HandleFunc("PUT /api/sets/{setID}/mindmaps/{mindMapID}", middleware.SyncUserMiddleware(DBHandler.UpdateMindMapByID))
	mux.HandleFunc("DELETE /api/sets/{setID}/mindmaps/{mindMapID}", middleware.SyncUserMiddleware(DBHandler.DeleteMindMapByID))
	mux.HandleFunc("PUT /api/sets/{setID}/mindmaps/{mindMapID}/connections", DBHandler.UpdateMindMapConnections)
	mux.HandleFunc("PUT /api/sets/{setID}/mindmaps/{mindMapID}/layouts", DBHandler.UpdateMindMapLayouts)

	// Blocks
	mux.HandleFunc("GET /api/blocks/leaderboard/{setID}", DBHandler.GetBlocksLeaderboard)
	mux.HandleFunc("POST /api/blocks/score/{setID}", DBHandler.CreateBlockScore)

	// Flashcard
	mux.HandleFunc("POST /api/sets/{setID}/flashcards/", middleware.SyncUserMiddleware(DBHandler.CreateFlashCard))
	mux.HandleFunc("GET /api/sets/{setID}/flashcards/{flashcardID}", middleware.SyncUserMiddleware(DBHandler.GetFlashcardByID))
	mux.HandleFunc("GET /api/sets/{setID}/flashcards", DBHandler.GetFlashcardsForSet)
	mux.HandleFunc("PUT /api/sets/{setID}/flashcards/{flashcardID}", middleware.SyncUserMiddleware(DBHandler.UpdateFlashCardByID))
	mux.HandleFunc("DELETE /api/sets/{setID}/flashcards/{flashcardID}", middleware.SyncUserMiddleware(DBHandler.DeleteFlashCardByID))

	// Configure CORS with specific options
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "https://thenodebook.vercel.app", "https://www.mindthred.com"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Requested-With", "Accept", "Origin"},
		AllowCredentials: true,
		MaxAge:           86400,
	}).Handler(authMiddleware(mux))

	// Server configuration

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // fallback port for local development
	}
	serverAddr := "0.0.0.0:" + port

	http.ListenAndServe(serverAddr, corsHandler)
}
