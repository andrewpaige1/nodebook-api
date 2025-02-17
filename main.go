package main

import (
	"log"
	"net/http"
	"os"

	"github.com/andrewpaige1/nodebook-api/auth"
	"github.com/andrewpaige1/nodebook-api/config"
	"github.com/andrewpaige1/nodebook-api/handlers"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func init() {
	// Load .env file if not in production environment
	if os.Getenv("RENDER") == "" {
		err := godotenv.Load()
		if err != nil {
			log.Printf("Warning: .env file not found, environment variables might not be loaded: %v", err)
		}
	}
}

func main() {
	// Initialize database connection
	config.Connect()
	// Create a new ServeMux
	mux := http.NewServeMux()

	// User routes
	mux.HandleFunc("GET /app/users", auth.AuthMiddleware(handlers.GetUsers))
	mux.HandleFunc("POST /app/users", handlers.AddUser)

	// Flashcard Set routes
	mux.HandleFunc("GET /app/users/{nickname}/flashcard-sets", auth.AuthMiddleware(handlers.GetUserFlashcardSets)) // all flashcards
	mux.HandleFunc("GET /app/users/{nickname}/sets/{title}", handlers.GetUserFlashcardSetByTitle)                  // singular flashcard set

	mux.HandleFunc("POST /app/createSet", auth.AuthMiddleware(handlers.CreateSetWithCards))
	mux.HandleFunc("POST /app/updateSet", auth.AuthMiddleware(handlers.UpdateSetWithCards))
	mux.HandleFunc("POST /app/deleteSet", auth.AuthMiddleware(handlers.DeleteUserFlashcardSet))

	mux.HandleFunc("GET /app/{nickname}/sets/mindmap/{setName}", auth.AuthMiddleware(handlers.GetMindMap))
	mux.HandleFunc("GET /app/{nickname}/{setName}/mindmaps", auth.AuthMiddleware(handlers.GetMindMapForSets))
	mux.HandleFunc("GET /app/{nickname}/mindmap/state/{title}", auth.AuthMiddleware((handlers.GetMindMapState)))
	mux.HandleFunc("POST /app/mindmap/create", auth.AuthMiddleware((handlers.CreateMindMap)))
	mux.HandleFunc("POST /app/mindmap/checkDup", auth.AuthMiddleware((handlers.CheckDup)))
	mux.HandleFunc("POST /app/mindmap/delete", auth.AuthMiddleware((handlers.DeleteMindMap)))
	mux.HandleFunc("POST /app/mindmap/updateConnections", auth.AuthMiddleware((handlers.UpdateConnections)))
	mux.HandleFunc("POST /app/mindmap/updateNodeLayout", auth.AuthMiddleware((handlers.UpdateNodeLayout)))

	// Configure CORS with specific options
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "https://thenodebook.vercel.app", "https://www.mindthred.com"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Requested-With", "Accept", "Origin"},
		AllowCredentials: true,
		MaxAge:           86400,
	}).Handler(mux)

	// Server configuration

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // fallback port for local development
	}
	serverAddr := "0.0.0.0:" + port

	http.ListenAndServe(serverAddr, corsHandler)
}
