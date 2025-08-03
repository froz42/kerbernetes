package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/danielgtaylor/huma/v2"
	"github.com/froz42/kerbernetes/internal/controllers"
	"github.com/froz42/kerbernetes/internal/openapi"
	"github.com/froz42/kerbernetes/internal/services"
	configsvc "github.com/froz42/kerbernetes/internal/services/config"
	"github.com/go-chi/chi/v5"
	"github.com/samber/do"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	_ "github.com/danielgtaylor/huma/v2/formats/cbor"
	"github.com/go-chi/httplog/v3"
)

var Version = "dev"

func main() {
	apiBootstrap()
}

// apiBootstrap initializes the API server
func apiBootstrap() {
	logFormat := httplog.SchemaECS.Concise(false)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: logFormat.ReplaceAttr,
	})).With(
		slog.String("app", "kerbernetes-api"),
		slog.String("version", Version),
	)

	injector := do.New()
	do.ProvideValue(injector, logger)
	err := services.InitServices(injector)

	if err != nil {
		logger.Error("Failed to initialize services", "error", err)
		os.Exit(1)
	}

	config := do.MustInvoke[configsvc.ConfigService](injector).GetConfig()

	router := chi.NewRouter()

	router.Use(httplog.RequestLogger(logger, &httplog.Options{
		Level:         slog.LevelInfo,
		Schema:        httplog.SchemaECS,
		RecoverPanics: true,
	}))

	router.Route(config.APIPrefix, apiMux(injector))

	logger.Info("Started API server", "port", config.HTTPPort, "prefix", config.APIPrefix)
	err = http.ListenAndServe(fmt.Sprintf(":%d", config.HTTPPort), router)
	if err != nil {
		logger.Error("Failed to start API server", "error", err)
		os.Exit(1)
	}
}

// apiMux returns a function that initializes the API routes
func apiMux(
	injector *do.Injector,
) func(chi.Router) {
	logger := do.MustInvoke[*slog.Logger](injector)
	return func(router chi.Router) {
		config := do.MustInvoke[configsvc.ConfigService](injector).GetConfig()
		humaConfig := huma.DefaultConfig("Kerbetes API", "dev")
		humaConfig = openapi.WithOverviewDoc(humaConfig)
		humaConfig = openapi.WithServers(humaConfig, config)
		api := humachi.New(router, humaConfig)
		err := controllers.ControllersInit(api, injector)
		if err != nil {
			logger.Error("Failed to initialize controllers", "error", err)
			os.Exit(1)
		}
		router.Get("/", openapi.ScalarDocHandler(config))
		router.Get("/docs", notFoundHandler)
	}
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "404 page not found", http.StatusNotFound)
}
