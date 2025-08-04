package authctrl

import (
	"context"
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	"github.com/froz42/kerbernetes/internal/middlewares"
	"github.com/froz42/kerbernetes/internal/security"
	authsvc "github.com/froz42/kerbernetes/internal/services/auth"
	envsvc "github.com/froz42/kerbernetes/internal/services/env"
	"github.com/samber/do"
)

type authController struct {
	authSvc authsvc.AuthService
	env     envsvc.Env
	logger  *slog.Logger
}

func Init(api huma.API, injector *do.Injector) {
	authController := &authController{
		authSvc: do.MustInvoke[authsvc.AuthService](injector),
		env:     do.MustInvoke[envsvc.EnvSvc](injector).GetEnv(),
		logger:  do.MustInvoke[*slog.Logger](injector),
	}
	authController.Register(api)
}

func (ctrl *authController) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		Method:      "GET",
		Path:        "/auth/kerberos",
		Summary:     "Kerberos auth",
		Description: `This endpoint is used to handle the Kerberos authentication.`,
		Tags:        []string{"Authentification"},
		OperationID: "getKerberosAuth",
		Middlewares: huma.Middlewares{middlewares.SPNEGO(ctrl.logger, ctrl.env.KeytabPath)},
	}, ctrl.getKerberosAuth)
}

func (ctrl *authController) getKerberosAuth(
	ctx context.Context,
	input *struct{},
) (*kerberosAuthOutput, error) {
	principal, err := security.GetPrincipalFromContext(ctx)
	if err != nil {
		return nil, err
	}
	creds, err := ctrl.authSvc.AuthAccount(ctx, principal)
	if err != nil {
		return nil, err
	}
	return &kerberosAuthOutput{
		Body: creds,
	}, nil
}
