package api

import (
	"net/http"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/api/handlers"
	authmw "github.com/Sirbuschi2003/ControlPanelVPS/master/internal/api/middleware"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/config"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/services"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(cfg *config.Config, db *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "https://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	authSvc := services.NewAuthService(db, cfg.JWTSecret)
	serverSvc := services.NewServerService(db)

	authHandler := handlers.NewAuthHandler(authSvc)
	serverHandler := handlers.NewServerHandler(serverSvc)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Public routes with rate limiting
	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(10, 60))
		r.Post("/api/auth/login", authHandler.Login)
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authmw.Auth(authSvc))

		r.Get("/api/auth/me", authHandler.Me)

		r.Get("/api/servers", serverHandler.List)
		r.Post("/api/servers", serverHandler.Create)
		r.Get("/api/servers/{id}/metrics", serverHandler.GetMetrics)
	})

	return r
}
