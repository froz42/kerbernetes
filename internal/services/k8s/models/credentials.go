package k8smodels

type Credentials struct {
	Kind       string  `json:"kind" description:"Kind of the the object, should be ExecCredential"`
	ApiVersion string  `json:"apiVersion" description:"API version of the object, should be client.authentication.k8s.io/v1beta1"`
	Status     *Status `json:"status,omitempty" description:"Status of the credentials, contains the token to use for authentication"`
}

type Status struct {
	Token               string `json:"token" description:"The token to use for authentication"`
	ExpirationTimestamp string `json:"expirationTimestamp,omitempty" description:"The expiration timestamp of the token, if available"`
}
