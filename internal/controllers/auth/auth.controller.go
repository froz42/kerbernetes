package authctrl

import (
	"context"
	"errors"
	"log"

	"github.com/danielgtaylor/huma/v2"
	"github.com/froz42/kerbernetes/internal/middlewares"
	"github.com/froz42/kerbernetes/internal/security"
	authsvc "github.com/froz42/kerbernetes/internal/services/auth"
	configservice "github.com/froz42/kerbernetes/internal/services/config"
	k8ssvc "github.com/froz42/kerbernetes/internal/services/k8s"
	"github.com/samber/do"
)

type authController struct {
	authSvc authsvc.AuthService
	k8sSvc  k8ssvc.K8sService
	config  configservice.Config
}

func Init(api huma.API, injector *do.Injector) {
	authController := &authController{
		authSvc: do.MustInvoke[authsvc.AuthService](injector),
		k8sSvc:  do.MustInvoke[k8ssvc.K8sService](injector),
		config:  do.MustInvoke[configservice.ConfigService](injector).GetConfig(),
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
) (*kerberosAuthOutput, error) {
	principal, err := security.GetPrincipalFromContext(ctx)
	if err != nil {
		return nil, err
	}
	log.Printf("Authenticated principal: %s", principal)
	creds, err := ctrl.k8sSvc.AuthAccount(ctx, principal)
	if err != nil {
		log.Printf("Failed to authenticate user on Kubernetes: %v", err)
		return nil, huma.Error401Unauthorized("Unauthorized", errors.New("failed to authenticate user on Kubernetes"))
	}
	return &kerberosAuthOutput{
		Body: creds,
	}, nil
}
