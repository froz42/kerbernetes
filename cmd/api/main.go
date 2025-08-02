package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/froz42/kerbernetes/internal/controllers"
	"github.com/froz42/kerbernetes/internal/openapi"
	"github.com/froz42/kerbernetes/internal/services"
	configservice "github.com/froz42/kerbernetes/internal/services/config"
	"github.com/go-chi/chi/v5"
	"github.com/samber/do"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	_ "github.com/danielgtaylor/huma/v2/formats/cbor"
)

func main() {
	apiBootstrap()
}

// apiBootstrap initializes the API server
func apiBootstrap() {
	injector := do.New()
	err := services.InitServices(injector)
	if err != nil {
		log.Fatalf("Failed to initialize services: %v", err)
	}

	config := do.MustInvoke[configservice.ConfigService](injector).GetConfig()

	router := chi.NewRouter()

	router.Route(config.APIPrefix, apiMux(injector))

	log.Printf("Starting API server on port %d", config.HTTPPort)
	err = http.ListenAndServe(fmt.Sprintf(":%d", config.HTTPPort), router)
	if err != nil {
		log.Fatalf("Failed to start API server: %v", err)
	}
}

// apiMux returns a function that initializes the API routes
func apiMux(
	injector *do.Injector,
) func(chi.Router) {
	return func(router chi.Router) {
		config := do.MustInvoke[configservice.ConfigService](injector).GetConfig()
		humaConfig := huma.DefaultConfig("Kerbetes API", "dev")
		humaConfig = openapi.WithOverviewDoc(humaConfig)
		humaConfig = openapi.WithServers(humaConfig, config)
		api := humachi.New(router, humaConfig)
		err := controllers.ControllersInit(api, injector)
		if err != nil {
			log.Fatalf("Failed to initialize controllers: %v", err)
		}
		router.Get("/", openapi.ScalarDocHandler(config))
		router.Get("/docs", notFoundHandler)
		log.Printf("API initialized with prefix: %s", config.APIPrefix)
	}
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "404 page not found", http.StatusNotFound)
}
