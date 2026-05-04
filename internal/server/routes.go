package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (s *Server) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	r.Route("/api", func(r chi.Router) {
		r.Get("/event", s.handleEvent)
		r.Get("/path", s.handlePath)
		r.Get("/agent", s.handleAgents)
		r.Get("/models", s.handleModels)
		r.Get("/config", s.handleConfig)
		r.Get("/mode", s.handleMode)

		r.Post("/models/preference", s.handleSetModelPreference)
		r.Delete("/models/preference/{id}", s.handleDeleteModelPreference)

		r.Get("/theme", s.handleGetTheme)
		r.Post("/theme", s.handleSetTheme)
		r.Delete("/theme/{directory}", s.handleDeleteTheme)

		r.Route("/session", func(r chi.Router) {
			r.Get("/", s.handleListSessions)
			r.Post("/", s.handleCreateSession)

			r.Route("/{sessionID}", func(r chi.Router) {
				r.Get("/", s.handleGetSession)
				r.Patch("/", s.handleUpdateSession)
				r.Delete("/", s.handleDeleteSession)
				r.Post("/abort", s.handleAbortSession)
				r.Post("/prompt", s.handlePrompt)
				r.Get("/message", s.handleGetMessages)
				r.Post("/permission/{permissionID}", s.handlePermissionReply)
			})
		})

		r.Route("/plans", func(r chi.Router) {
			r.Get("/", s.handleListPlans)
			r.Post("/", s.handleCreatePlan)
			r.Route("/{planID}", func(r chi.Router) {
				r.Get("/", s.handleGetPlan)
				r.Patch("/", s.handleUpdatePlan)
				r.Delete("/", s.handleDeletePlan)
				r.Post("/lock", s.handleLockPlan)
				r.Post("/abort", s.handleAbortPlan)
				r.Post("/prompt", s.handlePlanPrompt)
				r.Get("/message", s.handleGetPlanMessages)
				r.Get("/export", s.handleExportPlan)
				r.Get("/tasks", s.handleListTasks)
				r.Post("/tasks", s.handleCreateTasks)
			})
		})

		r.Route("/tasks", func(r chi.Router) {
			r.Route("/{taskID}", func(r chi.Router) {
				r.Get("/", s.handleGetTask)
				r.Patch("/", s.handleUpdateTask)
				r.Post("/start", s.handleStartTask)
				r.Post("/complete", s.handleCompleteTask)
				r.Post("/fail", s.handleFailTask)
				r.Post("/retry", s.handleRetryTask)
			})
		})

		r.Get("/vcs", s.handleVCS)
	})

	// Serve embedded web UI (or placeholder for dev)
	s.serveStatic(r)

	return r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, x-ogcode-directory")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}