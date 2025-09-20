package middleware

import (
	"context"
	"log"
	"net/http"

	"github.com/andrewpaige1/nodebook-api/config"
	"github.com/andrewpaige1/nodebook-api/models"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
)

type contextKey string

// SyncUserMiddleware ensures the Auth0 user exists in the DB and attaches it to context
func SyncUserMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
		if !ok || claims.RegisteredClaims.Subject == "" {
			http.Error(w, "No Auth0 subject found", http.StatusUnauthorized)
			return
		}

		auth0ID := claims.RegisteredClaims.Subject
		nickname := ""

		type Auth0Payload struct {
			Auth0ID  string `json:"sub"`
			Nickname string `json:"nickname"`
		}

		auth0Payload := Auth0Payload{
			Auth0ID:  auth0ID,
			Nickname: nickname,
		}

		var user models.User
		result := config.Database.Where("auth0_id = ?", auth0ID).First(&user)
		if result.Error != nil {
			// User does not exist, create a new one
			user = models.User{
				Auth0ID:  auth0Payload.Auth0ID,
				Nickname: auth0Payload.Nickname,
			}
			createResult := config.Database.Create(&user)
			if createResult.Error != nil {
				http.Error(w, "Failed to create user", http.StatusInternalServerError)
				log.Println("Database creation error:", createResult.Error)
				return
			}
			log.Printf("Created new user: %s\n", user.Nickname)
		} else {
			// User exists, update details if necessary
			updated := false
			if user.Nickname != auth0Payload.Nickname && auth0Payload.Nickname != "" {
				user.Nickname = auth0Payload.Nickname
				updated = true
			}
			if updated {
				saveResult := config.Database.Save(&user)
				if saveResult.Error != nil {
					http.Error(w, "Failed to update user", http.StatusInternalServerError)
					log.Println("Database update error:", saveResult.Error)
					return
				}
				//log.Printf("Updated user: %s\n", user.Nickname)
			}
		}

		// Add user to context for downstream handlers
		ctx := r.Context()
		const userContextKey contextKey = "user"
		ctx = context.WithValue(ctx, userContextKey, &user)
		r = r.WithContext(ctx)

		// Call the next handler
		next.ServeHTTP(w, r)
	}
}
