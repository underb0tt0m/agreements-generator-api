package service

import (
	"context"
	"fmt"

	"agreements-generator/internal/domain"
	"agreements-generator/internal/dto"
	"agreements-generator/internal/hasher"
	"agreements-generator/internal/storage"
	"agreements-generator/internal/token_manager"
)

//go:generate mockgen --source=auth.go --destination=../mocks/auth.go --package=mocks --mock_names=Auth=MockAuthService
type Auth interface {
	Register(ctx context.Context, userData dto.RegisterRequest) (string, error)
	LogIn(ctx context.Context, userData dto.LogInRequest) (string, error)
}
type auth struct {
	storage  storage.UserStorage
	tokenMng token_manager.TokenManager
	hasher   hasher.Hasher
}

func NewAuth(s storage.UserStorage, t token_manager.TokenManager, h hasher.Hasher) Auth {
	return &auth{
		storage:  s,
		tokenMng: t,
		hasher:   h,
	}
}

func (a *auth) Register(ctx context.Context, userData dto.RegisterRequest) (string, error) {
	if len(userData.Password) < 8 {
		return "", fmt.Errorf("can't register user, password is too short: %w", domain.ErrBadRequest)
	}

	if userData.Login == "" {
		return "", fmt.Errorf("can't register user, login is empty: %w", domain.ErrBadRequest)
	}

	hashedPassword, err := a.hasher.Hash(userData.Password)
	if err != nil {
		return "", fmt.Errorf("can't hash password of user with login %s: %w", userData.Login, err)
	}

	user := domain.User{
		Name:     userData.Name,
		Login:    userData.Login,
		Password: hashedPassword,
	}

	userID, err := a.storage.Register(ctx, user)
	if err != nil {
		return "", fmt.Errorf("can't register user with login %s: %w", user.Login, err)
	}

	token, err := a.tokenMng.Create(token_manager.UserClaims{
		Login: user.Login,
		ID:    userID,
	})
	if err != nil {
		return "", fmt.Errorf("can't create token for user with login %s: %w", user.Login, err)
	}

	return token, nil
}

func (a *auth) LogIn(ctx context.Context, userData dto.LogInRequest) (string, error) {
	userID, hashedPassword, err := a.storage.LogIn(ctx, userData.Login)
	if err != nil {
		return "", fmt.Errorf("can't get password of user with login %s: %w", userData.Login, err)
	}

	err = a.hasher.Compare(hashedPassword, userData.Password)
	if err != nil {
		return "", fmt.Errorf("error during comparing passwords of user with login %s: %w", userData.Login, err)
	}

	token, err := a.tokenMng.Create(token_manager.UserClaims{
		Login: userData.Login,
		ID:    userID,
	})
	if err != nil {
		return "", fmt.Errorf("can't create token for user with login %s: %w", userData.Login, err)
	}

	return token, nil
}
