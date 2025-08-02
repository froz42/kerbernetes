package authctrl

import k8smodels "github.com/froz42/kerbernetes/internal/services/k8s/models"

type kerberosAuthOutput struct {
	Body *k8smodels.Credentials
}
