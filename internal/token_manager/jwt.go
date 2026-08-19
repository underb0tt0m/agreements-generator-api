package token_manager

import (
	"fmt"
	"time"

	"agreements-generator/internal/config"
	"agreements-generator/internal/domain"
	"agreements-generator/internal/encoder"
	"agreements-generator/internal/logger"

	"github.com/golang-jwt/jwt/v5"
)

//go:generate mockgen -source=jwt.go -destination=../mocks/jwt.go -package=mocks
type TokenManager interface {
	Create(data any) (string, error)
	Validate(token string) error
	Parse(tokenString string, obj interface{}) error
	GetPrefix() string
}

type tokenMaker struct {
	logger        logger.Logger
	encoder       encoder.Encoder
	ttl           time.Duration
	signingMethod jwt.SigningMethod
	secret        []byte
	prefix        string
}

func New(l logger.Logger, e encoder.Encoder, ttl time.Duration, signM string, secret string, prefix string) TokenManager {
	var method jwt.SigningMethod
	switch signM {
	case config.ES256:
		method = jwt.SigningMethodES256
	case config.HS256:
		method = jwt.SigningMethodHS256
	case config.RS256:
		method = jwt.SigningMethodRS256
	default:
		l.Warn(fmt.Sprintf("unknown signing method %s, use default HS256", signM))
		method = jwt.SigningMethodHS256
	}

	return &tokenMaker{
		logger:        l,
		encoder:       e,
		ttl:           ttl,
		signingMethod: method,
		secret:        []byte(secret),
		prefix:        prefix,
	}
}

type UserClaims struct {
	Login string
	ID    int
}

func (t *tokenMaker) Create(data any) (string, error) {
	bytes, err := t.encoder.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("can't marshal data: %v, %w", err, domain.ErrInternal)
	}

	claims := jwt.MapClaims{}
	if err = t.encoder.Unmarshal(bytes, &claims); err != nil {
		return "", fmt.Errorf("can't unmarshal data: %v, %w", err, domain.ErrInternal)
	}

	claims["iat"] = jwt.NewNumericDate(time.Now())
	claims["exp"] = jwt.NewNumericDate(time.Now().Add(t.ttl))

	token := jwt.NewWithClaims(t.signingMethod, claims)
	signedToken, err := token.SignedString(t.secret)
	if err != nil {
		t.logger.Debug("can't sign token", logger.FieldError, err)
		return "", fmt.Errorf("can't sign token: %v, %w", err, domain.ErrInternal)
	}

	return signedToken, nil
}

func (t *tokenMaker) Validate(tokenString string) error {
	_, err := jwt.Parse(tokenString, t.createKeyFunc(t.secret))
	if err != nil {
		return fmt.Errorf("invalid token: %v, %w", err, domain.ErrInvalidToken)
	}

	return nil
}

func (t *tokenMaker) Parse(tokenString string, obj interface{}) error {
	token, err := jwt.Parse(tokenString, t.createKeyFunc(t.secret))
	if err != nil {
		return fmt.Errorf("invalid token: %v, %w", err, domain.ErrInvalidToken)
	}

	if !token.Valid {
		return domain.ErrInvalidToken
	}

	bytes, err := t.encoder.Marshal(token.Claims)
	if err != nil {
		return err
	}

	if err = t.encoder.Unmarshal(bytes, obj); err != nil {
		t.logger.Debug("can't parse token into input object", logger.FieldError, err)
		return fmt.Errorf("invalid token: %v, %w", err, domain.ErrInvalidToken)
	}

	return nil
}

func (t *tokenMaker) GetPrefix() string {
	return t.prefix
}

func (t *tokenMaker) createKeyFunc(secret []byte) jwt.Keyfunc {
	return func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != t.signingMethod.Alg() {
			return "", domain.ErrWrongSigningMethod
		}

		return secret, nil
	}
}
