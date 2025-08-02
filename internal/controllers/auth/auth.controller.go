package authcontroller

import (
	"github.com/danielgtaylor/huma/v2"
	authservice "github.com/froz42/kerbernetes/internal/services/auth"
	configservice "github.com/froz42/kerbernetes/internal/services/config"
	"github.com/samber/do"
)

type authController struct {
	authService authservice.AuthService
	config      configservice.Config
}

func Init(api huma.API, injector *do.Injector) {
	authController := &authController{
		authService: do.MustInvoke[authservice.AuthService](injector),
		config:      do.MustInvoke[configservice.ConfigService](injector).GetConfig(),
	}
	authController.Register(api)
}

func (ctrl *authController) Register(api huma.API) {

}
