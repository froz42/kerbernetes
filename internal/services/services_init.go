package services

import (
	authsvc "github.com/froz42/kerbernetes/internal/services/auth"
	envsvc "github.com/froz42/kerbernetes/internal/services/env"
	k8ssvc "github.com/froz42/kerbernetes/internal/services/k8s"
	ldapclusterrolebindingssvc "github.com/froz42/kerbernetes/internal/services/k8s/ldapclusterrolebindings"
	serviceaccountssvc "github.com/froz42/kerbernetes/internal/services/k8s/serviceaccounts"
	ldapsvc "github.com/froz42/kerbernetes/internal/services/ldap"
	"github.com/samber/do"
)

func InitServices(i *do.Injector) error {
	do.Provide(i, envsvc.NewProvider())
	do.Provide(i, authsvc.NewProvider())
	do.Provide(i, k8ssvc.NewProvider())
	do.Provide(i, ldapsvc.NewProvider())
	do.Provide(i, ldapclusterrolebindingssvc.NewProvider())
	do.Provide(i, serviceaccountssvc.NewProvider())
	return nil
}
