package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/danielgtaylor/huma/v2"
	"github.com/froz42/kerbernetes/internal/controllers"
	"github.com/froz42/kerbernetes/internal/openapi"
	"github.com/froz42/kerbernetes/internal/services"
	envsvc "github.com/froz42/kerbernetes/internal/services/env"
	ldapgroupbindingssvc "github.com/froz42/kerbernetes/internal/services/k8s/ldapgroupbindings"
	"github.com/go-chi/chi/v5"
	"github.com/samber/do"

	"github.com/MatusOllah/slogcolor"
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
	logger := slog.New(slogcolor.NewHandler(os.Stderr, slogcolor.DefaultOptions)).With(
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

	svc := do.MustInvoke[ldapgroupbindingssvc.LdapGroupBindingService](injector)

	go func() {
		err = svc.Start(context.Background())
		if err != nil {
			logger.Error("Failed to start LDAP Group Bindings service", "error", err)
			os.Exit(1)
		}
	}()

	env := do.MustInvoke[envsvc.EnvSvc](injector).GetEnv()

	router := chi.NewRouter()

	router.Use(httplog.RequestLogger(logger, &httplog.Options{
		Level:         slog.LevelInfo,
		Schema:        httplog.SchemaECS,
		RecoverPanics: true,
	}))

	router.Route(env.APIPrefix, apiMux(injector))

	logger.Info("Started API server", "port", env.HTTPPort, "prefix", env.APIPrefix)
	err = http.ListenAndServe(fmt.Sprintf(":%d", env.HTTPPort), router)
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
		config := do.MustInvoke[envsvc.EnvSvc](injector).GetEnv()
		humaConfig := huma.DefaultConfig("Kerbernetes API", "dev")
		humaConfig = openapi.WithOverviewDoc(humaConfig)
		humaConfig = openapi.WithServers(humaConfig, config)
		api := humachi.New(router, humaConfig)
		err := controllers.ControllersInit(api, injector)
		if err != nil {
			logger.Error("Failed to initialize controllers", "error", err)
			os.Exit(1)
		}
		router.Get("/docs", notFoundHandler)
	}
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "404 page not found", http.StatusNotFound)
}
