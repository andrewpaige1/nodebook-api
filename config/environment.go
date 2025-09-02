package config

import "os"

type Environment struct {
	IsDevelopment bool
	Domain        string
	CookieSecure  bool
}

var Env Environment

func init() {
	// Get domain from environment variable
	domain := os.Getenv("COOKIE_DOMAIN")

	// If no domain is set, we're in development
	isDev := domain == ""
	if isDev {
		domain = "localhost"
	}

	Env = Environment{
		IsDevelopment: isDev,
		Domain:        domain,
		CookieSecure:  !isDev,
	}
}
