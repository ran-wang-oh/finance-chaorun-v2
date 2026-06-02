package api

import (
	"net/http"

	"finance.chao.run/v2/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func Routes(svc *service.Service) http.Handler {
	return routesWithHandler(NewHandler(svc))
}

func routesWithHandler(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Get("/healthz", h.Healthz)
	r.Get("/readyz", h.Readyz)

	r.Route("/v1", func(r chi.Router) {
		r.Get("/capabilities", h.ListCapabilities)
		r.Post("/context", h.GetContext)

		r.Route("/capabilities/{capability_id}", func(r chi.Router) {
			r.Post("/preview", h.Preview)
			r.Post("/validate", h.Validate)
			r.Post("/execute", h.Execute)
		})

		r.Get("/resources/{resource_uri}", h.GetResource)
	})

	return r
}
