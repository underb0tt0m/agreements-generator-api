package api_v1

import (
	"context"
	"net/http"
	"strings"

	"agreements-generator/internal/domain"
	"agreements-generator/internal/encoder"
	"agreements-generator/internal/logging"
	"agreements-generator/internal/token_manager"
)

type user struct {
	Login string `yaml:"login"`
}

func MWAuth(tokenMaker token_manager.TokenManager, enc encoder.Encoder, log logging.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawToken := r.Header.Get("Authorization")
			if rawToken == "" {
				writeError(w, domain.ErrUnauthorized, enc, log)
				return
			}

			token, ok := strings.CutPrefix(rawToken, tokenMaker.GetPrefix())
			if !ok {
				writeError(w, domain.ErrUnauthorized.Wrap("token without prefix", nil), enc, log)
				return
			}

			token = strings.TrimSpace(token)
			if err := tokenMaker.Validate(token); err != nil {
				log.Debug("validation is failed", logging.FieldError, err)
				writeError(w, domain.ErrUnauthorized.Wrap("invalid token", nil), enc, log)
				return
			}

			usr := user{}
			if err := tokenMaker.Parse(token, &usr); err != nil {
				log.Debug("can't parse token", logging.FieldError, err)
				writeError(w, domain.ErrUnauthorized.Wrap("invalid token", nil), enc, log)
				return
			}

			ctx := context.WithValue(r.Context(), domain.LoginKey, usr.Login)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
