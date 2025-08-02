package openapi

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/froz42/kerbernetes/internal/security"
)

var scopesDescription = map[string]string{
	"openid":         "connect to the identity provider",
	"email":          "see your email address",
	"offline_access": "have access to your account even when you are not connected",
	"profile":        "see your profile",
	"security":       "access to the security API",
}

func getOauth2Flows(baseURL string) *huma.OAuthFlows {
	return &huma.OAuthFlows{
		AuthorizationCode: &huma.OAuthFlow{
			TokenURL:         baseURL + "/oauth/token",
			AuthorizationURL: baseURL + "/authorize",
			RefreshURL:       baseURL + "/token",
			Scopes:           scopesDescription,
		},
		ClientCredentials: &huma.OAuthFlow{
			TokenURL: baseURL + "/oauth/token",
			Scopes:   scopesDescription,
		},
	}
}

func WithAuthSchemes(humaConfig huma.Config) huma.Config {
	humaConfig.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		security.Oauth2: {
			Type:        "oauth2",
			Scheme:      "bearer",
			Description: "OAuth2 security scheme",
			Flows:       getOauth2Flows("http://localhost:3000/api/openid"),
		},
	}
	return humaConfig
}
