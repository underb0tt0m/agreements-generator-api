package token_manager

import (
	"time"

	"agreements-generator/internal/config"
	"agreements-generator/internal/domain"
	"agreements-generator/internal/encoder"
	"agreements-generator/internal/logging"

	"github.com/golang-jwt/jwt/v5"
)

type TokenManager interface {
	Create(data any) (string, error)
	Validate(token string) error
	Parse(tokenString string, obj interface{}) error
	GetPrefix() string
}

type tokenMaker struct {
	logger        logging.Logger
	encoder       encoder.Encoder
	ttl           time.Duration
	signingMethod jwt.SigningMethod
	secret        []byte
	prefix        string
}

func New(l logging.Logger, e encoder.Encoder, ttl time.Duration, signM string, secret string, prefix string) TokenManager {
	var method jwt.SigningMethod
	switch signM {
	case config.ES256:
		method = jwt.SigningMethodES256
	case config.HS256:
		method = jwt.SigningMethodHS256
	case config.RS256:
		method = jwt.SigningMethodRS256
	default:
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
}

func (t *tokenMaker) Create(data any) (string, error) {
	bytes, err := t.encoder.Marshal(data)
	if err != nil {
		return "", domain.ErrInternal.Wrap("can't marshal token data", err)
	}

	claims := jwt.MapClaims{}
	if err = t.encoder.Unmarshal(bytes, &claims); err != nil {
		return "", domain.ErrInternal.Wrap("can't unmarshal token data into claims", err)
	}

	claims["iat"] = jwt.NewNumericDate(time.Now())
	claims["exp"] = jwt.NewNumericDate(time.Now().Add(t.ttl))

	token := jwt.NewWithClaims(t.signingMethod, claims)
	signedToken, err := token.SignedString(t.secret)
	if err != nil {
		t.logger.Debug("can't sign token", logging.FieldError, err)
		return "", domain.ErrInternal.Wrap("can't sign token", err)
	}

	return signedToken, nil
}

func (t *tokenMaker) Validate(tokenString string) error {
	_, err := jwt.Parse(tokenString, t.createKeyFunc(t.secret))
	if err != nil {
		return domain.ErrInvalidToken
	}

	return nil
}

func (t *tokenMaker) Parse(tokenString string, obj interface{}) error {
	token, err := jwt.Parse(tokenString, t.createKeyFunc(t.secret))
	if err != nil {
		return domain.ErrInvalidToken.Wrap("can't parse token into token object", err)
	}

	if !token.Valid {
		return domain.ErrInvalidToken.Wrap("token is invalid", nil)
	}

	bytes, err := t.encoder.Marshal(token.Claims)
	if err != nil {
		return domain.ErrInternal.Wrap("can't marshal token", err)
	}

	if err = t.encoder.Unmarshal(bytes, obj); err != nil {
		t.logger.Debug("can't parse token into input object", logging.FieldError, err)
		return domain.ErrInvalidToken.Wrap("can't parse token into input object", err)
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
