package api_v1

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"agreements-generator/internal/domain"
	"agreements-generator/internal/encoder"
	"agreements-generator/internal/logger"
	"agreements-generator/internal/token_manager"
)

type user struct {
	ID    int    `json:"id"`
	Login string `json:"login"`
}

func MWAuth(tokenMaker token_manager.TokenManager, enc encoder.Encoder, log logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawToken := r.Header.Get("Authorization")
			if rawToken == "" {
				writeError(w, domain.ErrUnauthorized, enc, log)
				return
			}

			token, ok := strings.CutPrefix(rawToken, tokenMaker.GetPrefix())
			if !ok {
				writeError(
					w,
					fmt.Errorf("token without prefix: %w", domain.ErrUnauthorized),
					enc,
					log,
				)
				return
			}

			token = strings.TrimSpace(token)
			if err := tokenMaker.Validate(token); err != nil {
				log.Debug("validation is failed", logger.FieldError, err)
				writeError(
					w,
					fmt.Errorf("invalid token: %v, %w", err, domain.ErrInvalidToken),
					enc,
					log,
				)
				return
			}

			usr := user{}
			if err := tokenMaker.Parse(token, &usr); err != nil {
				log.Debug("can't parse token", logger.FieldError, err)
				writeError(
					w,
					fmt.Errorf("invalid token: %v, %w", err, domain.ErrInvalidToken),
					enc,
					log,
				)
				return
			}

			log.Debug(fmt.Sprintf("set context variables. login: %v, userID: %v", usr.Login, usr.ID))

			ctx := context.WithValue(r.Context(), domain.LoginKey, usr.Login)
			ctx = context.WithValue(ctx, domain.UserIDKey, usr.ID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
