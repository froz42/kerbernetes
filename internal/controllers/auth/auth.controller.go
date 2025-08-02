package authcontroller

import (
	"context"
	"log"

	"github.com/danielgtaylor/huma/v2"
	"github.com/froz42/kerbernetes/internal/middlewares"
	"github.com/froz42/kerbernetes/internal/security"
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
	huma.Register(api, huma.Operation{
		Method:  "GET",
		Path:    "/auth/kerberos",
		Summary: "Kerberos auth",
		Description: `This endpoint is used to handle the Kerberos authentication for workstations.
		\u26a0\ufe0f **You are probably not interested in this endpoint as it is should be only used by the frontend**.`,
		Tags:        []string{"Authentification"},
		OperationID: "getKerberosAuth",
		Middlewares: huma.Middlewares{middlewares.SPNEGO(ctrl.config.KeytabPath)},
	}, ctrl.getKerberosAuth)
}

func (ctrl *authController) getKerberosAuth(
	ctx context.Context,
	input *struct{},
) (*struct{}, error) {
	principal, err := security.GetPrincipalFromContext(ctx)
	if err != nil {
		return nil, err
	}
	log.Printf("Authenticated principal: %s", principal)
	return nil, nil
}
