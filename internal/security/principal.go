package security

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
)

const PrincipalFromContextKey = "principal"

func GetPrincipalFromContext(ctx context.Context) (string, error) {
	principal, ok := ctx.Value(PrincipalFromContextKey).(string)
	if !ok || principal == "" {
		return "", huma.Error403Forbidden("unauthorized")
	}
	return principal, nil
}
