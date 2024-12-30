package main

import (
	"net/http"
	"os"

	"github.com/andrewpaige1/nodebook-api/auth"
	"github.com/andrewpaige1/nodebook-api/config"
	"github.com/andrewpaige1/nodebook-api/handlers"
	"github.com/rs/cors"
)

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
