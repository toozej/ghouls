package ghouls

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/gorilla/csrf"
)

var (
	// csrfKey must be 32-bytes long
	csrfKey  []byte
	localDev bool
)

func setupRouter() (*chi.Mux, error) {
	// rate limiter setup
	rateLimitOpts := httprate.Limit(
		10,             // requests
		10*time.Second, // per duration
		httprate.WithKeyFuncs(httprate.KeyByIP, httprate.KeyByEndpoint),
	)

	// CORS setup
	corsOpts := cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link", "X-CSRF-Token"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}

	// CSRF protection setup
	CSRFMiddleware := csrf.Protect(
		csrfKey,
		csrf.Secure(!localDev),             // false in development only!
		csrf.RequestHeader("X-CSRF-Token"), // Must be in CORS Allowed and Exposed Headers
		csrf.Path("/"),
	)

	// setup router
	r := chi.NewRouter()

	// setup middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(rateLimitOpts)
	r.Use(cors.Handler(corsOpts))
	r.Use(CSRFMiddleware)
	r.Use(middleware.Recoverer)

	return r, nil
}
