package security

import (
	"context"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

const ClaimsKey = "claims"

type Claims struct {
	TokenID string
	Scopes  []string
	Subject string
}

type SubjectType string

const (
	SubjectTypeUser   SubjectType = "user"
	SubjectTypeClient SubjectType = "client"
)

func (c *Claims) GetSubjectType() SubjectType {
	if strings.HasPrefix(c.Subject, "clients/") {
		return SubjectTypeClient
	}
	return SubjectTypeUser
}

func (c *Claims) GetUserID() (int, error) {
	if c.GetSubjectType() == SubjectTypeUser {
		return strconv.Atoi(c.Subject)
	}
	return 0, huma.Error403Forbidden("this endpoint is only available for users")
}

func (c *Claims) GetClientID() (string, error) {
	if c.GetSubjectType() == SubjectTypeClient {
		return strings.TrimPrefix(c.Subject, "client;"), nil
	}
	return "", huma.Error403Forbidden("this endpoint is only available for applications")
}

func GetClaimsFromHumaContext(ctx huma.Context) (*Claims, error) {
	return GetClaimsFromContext(ctx.Context())
}

func GetClaimsFromContext(ctx context.Context) (*Claims, error) {
	claims, ok := ctx.Value("claims").(*Claims)
	if !ok {
		return nil, huma.Error403Forbidden("unauthorized")
	}
	return claims, nil
}

func GetUserIDFromContext(ctx context.Context) (int, error) {
	claims, err := GetClaimsFromContext(ctx)
	if err != nil {
		return 0, err
	}
	return claims.GetUserID()
}
