package service

import (
	"context"

	"agreements-generator/internal/domain"
	"agreements-generator/internal/dto"
	"agreements-generator/internal/hasher"
	"agreements-generator/internal/storage"
	"agreements-generator/internal/token_manager"
)

type Auth struct {
	storage  storage.UserStorage
	tokenMng token_manager.TokenManager
	hasher   hasher.Hasher
}

func NewAuth(s storage.UserStorage, t token_manager.TokenManager, h hasher.Hasher) *Auth {
	return &Auth{
		storage:  s,
		tokenMng: t,
		hasher:   h,
	}
}

func (a *Auth) Register(ctx context.Context, userData dto.RegisterRequest) (string, error) {
	if len(userData.Password) < 8 {
		return "", domain.ErrBadRequest.Wrap("password is too short", nil)
	}

	if userData.Login == "" {
		return "", domain.ErrBadRequest.Wrap("Login is empty", nil)
	}

	hashedPassword, err := a.hasher.Hash(userData.Password)
	if err != nil {
		return "", domain.CheckAppErr(err)
	}

	user := domain.User{
		Name:     userData.Name,
		Login:    userData.Login,
		Password: hashedPassword,
	}

	if err = a.storage.Register(ctx, user); err != nil {
		return "", domain.CheckAppErr(err)
	}

	token, err := a.tokenMng.Create(token_manager.UserClaims{Login: user.Login})
	if err != nil {
		return "", domain.CheckAppErr(err)
	}

	return token, nil
}

func (a *Auth) LogIn(ctx context.Context, userData dto.LogInRequest) (string, error) {
	hashedPassword, err := a.storage.LogIn(ctx, userData.Login)
	if err != nil {
		return "", domain.CheckAppErr(err)
	}

	err = a.hasher.Compare(hashedPassword, userData.Password)
	if err != nil {
		return "", domain.CheckAppErr(err)
	}

	token, err := a.tokenMng.Create(token_manager.UserClaims{Login: userData.Login})
	if err != nil {
		return "", domain.CheckAppErr(err)
	}

	return token, nil
}
