package middleware

import (
	"net/http"
	"strings"

	"github.com/rs/cors"
)

// CORS returns middleware that allows browser clients (e.g. the React example)
// to call this API cross-origin. No-op when allowedOrigins is empty.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	origins := make([]string, 0, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if o = strings.TrimSpace(o); o != "" {
			origins = append(origins, o)
		}
	}
	if len(origins) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	c := cors.New(cors.Options{
		AllowedOrigins: origins,
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"PAYMENT-SIGNATURE",
			"PAYMENT-REQUIRED",
			"PAYMENT-RESPONSE",
			"X-Payment",
			"X-Payment-Response",
		},
		ExposedHeaders: []string{
			"PAYMENT-SIGNATURE",
			"PAYMENT-REQUIRED",
			"PAYMENT-RESPONSE",
			"X-Payment",
			"X-Payment-Response",
		},
		MaxAge: 86400,
	})
	return c.Handler
}
