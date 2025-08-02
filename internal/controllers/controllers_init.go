package controllers

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/autopatch"
	authcontroller "github.com/froz42/kerbernetes/internal/controllers/auth"
	"github.com/samber/do"
)

// Every controller must have a function that initializes it
type controllerInitFunc func(api huma.API, injector *do.Injector)

// controllersList returns a list of all controllers init functions
func controllersList() []controllerInitFunc {
	return []controllerInitFunc{
		authcontroller.Init,
	}
}

// ControllersInit initializes all controllers
func ControllersInit(api huma.API, injector *do.Injector) error {
	controllerList := controllersList()
	for _, controller := range controllerList {
		controller(api, injector)
	}

	autopatch.AutoPatch(api)
	return nil
}
