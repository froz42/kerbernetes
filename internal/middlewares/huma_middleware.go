package middlewares

import "github.com/danielgtaylor/huma/v2"

type HumaMiddleware func(ctx huma.Context, next func(huma.Context))
