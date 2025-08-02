package middlewares

import (
	"log"
	"net/http"
	"os"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/froz42/kerbernetes/internal/security"
	"github.com/jcmturner/goidentity/v6"
	"github.com/jcmturner/gokrb5/v8/keytab"
	"github.com/jcmturner/gokrb5/v8/service"
	"github.com/jcmturner/gokrb5/v8/spnego"
)

func SPNEGO(
	keytabPath string,
) HumaMiddleware {
	kt, err := keytab.Load(keytabPath)
	if err != nil {
		log.Fatalf("Failed to load keytab: %v", err)
	}
	l := log.New(os.Stderr, "", log.LstdFlags)
	log.Printf("Keytab loaded from %s", keytabPath)
	return func(ctx huma.Context, next func(huma.Context)) {
		r, w := humachi.Unwrap(ctx)

		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			creds := goidentity.FromHTTPRequestContext(r)
			principal := creds.UserName()

			ctx = huma.WithValue(ctx, security.PrincipalFromContextKey, principal)
			next(ctx)
		})

		authHandler := spnego.SPNEGOKRB5Authenticate(inner, kt, service.Logger(l), service.DecodePAC(false))
		authHandler.ServeHTTP(w, r)
	}
}
