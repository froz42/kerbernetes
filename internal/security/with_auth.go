package security

func WithAuth(scopes ...string) []map[string][]string {
	return []map[string][]string{
		{
			Oauth2: scopes,
		},
	}
}
