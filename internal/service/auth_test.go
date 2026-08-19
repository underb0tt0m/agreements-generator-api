package service

import (
	"context"
	"errors"
	"testing"

	"agreements-generator/internal/domain"
	"agreements-generator/internal/dto"
	"agreements-generator/internal/mocks"

	"go.uber.org/mock/gomock"
)

func TestAuth_Register(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	storage := mocks.NewMockUserStorage(ctrl)
	tokenMng := mocks.NewMockTokenManager(ctrl)
	hasher := mocks.NewMockHasher(ctrl)

	type fields struct {
		storage  *mocks.MockUserStorage
		tokenMng *mocks.MockTokenManager
		hasher   *mocks.MockHasher
	}

	type args struct {
		ctx        context.Context
		userData   dto.RegisterRequest
		setupMocks func(s *mocks.MockUserStorage, t *mocks.MockTokenManager, h *mocks.MockHasher)
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		expectedErr error
	}{
		{
			name: "success",
			fields: fields{
				storage:  storage,
				tokenMng: tokenMng,
				hasher:   hasher,
			},
			args: args{
				ctx: ctx,
				userData: dto.RegisterRequest{
					Name:     "success",
					Login:    "success",
					Password: "success_success",
				},
				setupMocks: func(s *mocks.MockUserStorage, t *mocks.MockTokenManager, h *mocks.MockHasher) {
					h.
						EXPECT().
						Hash(gomock.Any()).
						Return([]byte{}, nil)
					s.
						EXPECT().
						Register(gomock.Any(), gomock.Any()).
						Return(1, nil)
					t.
						EXPECT().
						Create(gomock.Any()).
						Return("success", nil)
				},
			},
			expectedErr: nil,
		},

		{
			name: "short password",
			fields: fields{
				storage:  storage,
				tokenMng: tokenMng,
				hasher:   hasher,
			},
			args: args{
				ctx: ctx,
				userData: dto.RegisterRequest{
					Name:     "short_p",
					Login:    "short_p",
					Password: "short_p",
				},
			},
			expectedErr: domain.ErrBadRequest,
		},

		{
			name: "empty login",
			fields: fields{
				storage:  storage,
				tokenMng: tokenMng,
				hasher:   hasher,
			},
			args: args{
				ctx: ctx,
				userData: dto.RegisterRequest{
					Name:     "empty login",
					Login:    "",
					Password: "empty login",
				},
			},
			expectedErr: domain.ErrBadRequest,
		},

		{
			name: "hasher error",
			fields: fields{
				storage:  storage,
				tokenMng: tokenMng,
				hasher:   hasher,
			},
			args: args{
				ctx: ctx,
				userData: dto.RegisterRequest{
					Name:     "hasher error",
					Login:    "hasher error",
					Password: "hasher error",
				},
				setupMocks: func(s *mocks.MockUserStorage, t *mocks.MockTokenManager, h *mocks.MockHasher) {
					h.
						EXPECT().
						Hash(gomock.Any()).
						Return([]byte{}, domain.ErrInternal)
				},
			},
			expectedErr: domain.ErrInternal,
		},

		{
			name: "storage error",
			fields: fields{
				storage:  storage,
				tokenMng: tokenMng,
				hasher:   hasher,
			},
			args: args{
				ctx: ctx,
				userData: dto.RegisterRequest{
					Name:     "storage error",
					Login:    "storage error",
					Password: "storage error",
				},
				setupMocks: func(s *mocks.MockUserStorage, t *mocks.MockTokenManager, h *mocks.MockHasher) {
					h.
						EXPECT().
						Hash(gomock.Any()).
						Return([]byte{}, nil)
					s.
						EXPECT().
						Register(gomock.Any(), gomock.Any()).
						Return(0, domain.ErrStorageBadRequest)
				},
			},
			expectedErr: domain.ErrStorageBadRequest,
		},

		{
			name: "tokenMng error",
			fields: fields{
				storage:  storage,
				tokenMng: tokenMng,
				hasher:   hasher,
			},
			args: args{
				ctx: ctx,
				userData: dto.RegisterRequest{
					Name:     "tokenMng error",
					Login:    "tokenMng error",
					Password: "tokenMng error",
				},
				setupMocks: func(s *mocks.MockUserStorage, t *mocks.MockTokenManager, h *mocks.MockHasher) {
					h.
						EXPECT().
						Hash(gomock.Any()).
						Return([]byte{}, nil)
					s.
						EXPECT().
						Register(gomock.Any(), gomock.Any()).
						Return(1, nil)
					t.
						EXPECT().
						Create(gomock.Any()).
						Return("", domain.ErrInternal)
				},
			},
			expectedErr: domain.ErrInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.setupMocks != nil {
				tt.args.setupMocks(tt.fields.storage, tt.fields.tokenMng, tt.fields.hasher)
			}

			s := NewAuth(tt.fields.storage, tt.fields.tokenMng, tt.fields.hasher)

			token, err := s.Register(tt.args.ctx, tt.args.userData)

			if (err != nil) != (tt.expectedErr != nil) {
				t.Errorf("Register(): error presence mismatch: got %v, want %v", err, tt.expectedErr)
			}

			if err != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("Register(): got %v, want %v", err, tt.expectedErr)
			}

			if err == nil && token == "" {
				t.Errorf("Register(): got empty token without error")
			}
		})
	}
}

func TestAuth_LogIn(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	storage := mocks.NewMockUserStorage(ctrl)
	tokenMng := mocks.NewMockTokenManager(ctrl)
	hasher := mocks.NewMockHasher(ctrl)

	type fields struct {
		storage  *mocks.MockUserStorage
		tokenMng *mocks.MockTokenManager
		hasher   *mocks.MockHasher
	}

	type args struct {
		ctx        context.Context
		userData   dto.LogInRequest
		setupMocks func(s *mocks.MockUserStorage, t *mocks.MockTokenManager, h *mocks.MockHasher)
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		expectedErr error
	}{
		{
			name: "success",
			fields: fields{
				storage:  storage,
				tokenMng: tokenMng,
				hasher:   hasher,
			},
			args: args{
				ctx: ctx,
				userData: dto.LogInRequest{
					Login:    "success",
					Password: "success",
				},
				setupMocks: func(s *mocks.MockUserStorage, t *mocks.MockTokenManager, h *mocks.MockHasher) {
					s.
						EXPECT().
						LogIn(gomock.Any(), gomock.Any()).
						Return(1, []byte{}, nil)
					h.
						EXPECT().
						Compare(gomock.Any(), gomock.Any()).
						Return(nil)
					t.
						EXPECT().
						Create(gomock.Any()).
						Return("success", nil)
				},
			},
			expectedErr: nil,
		},

		{
			name: "storage error",
			fields: fields{
				storage:  storage,
				tokenMng: tokenMng,
				hasher:   hasher,
			},
			args: args{
				ctx: ctx,
				userData: dto.LogInRequest{
					Login:    "storage error",
					Password: "storage error",
				},
				setupMocks: func(s *mocks.MockUserStorage, t *mocks.MockTokenManager, h *mocks.MockHasher) {
					s.
						EXPECT().
						LogIn(gomock.Any(), gomock.Any()).
						Return(0, nil, domain.ErrStorageBadRequest)
				},
			},
			expectedErr: domain.ErrStorageBadRequest,
		},

		{
			name: "hasher error",
			fields: fields{
				storage:  storage,
				tokenMng: tokenMng,
				hasher:   hasher,
			},
			args: args{
				ctx: ctx,
				userData: dto.LogInRequest{
					Login:    "hasher error",
					Password: "hasher error",
				},
				setupMocks: func(s *mocks.MockUserStorage, t *mocks.MockTokenManager, h *mocks.MockHasher) {
					s.
						EXPECT().
						LogIn(gomock.Any(), gomock.Any()).
						Return(1, []byte{}, nil)
					h.
						EXPECT().
						Compare(gomock.Any(), gomock.Any()).
						Return(domain.ErrHashComparing)
				},
			},
			expectedErr: domain.ErrHashComparing,
		},

		{
			name: "tokenMng error",
			fields: fields{
				storage:  storage,
				tokenMng: tokenMng,
				hasher:   hasher,
			},
			args: args{
				ctx: ctx,
				userData: dto.LogInRequest{
					Login:    "tokenMng error",
					Password: "tokenMng error",
				},
				setupMocks: func(s *mocks.MockUserStorage, t *mocks.MockTokenManager, h *mocks.MockHasher) {
					s.
						EXPECT().
						LogIn(gomock.Any(), gomock.Any()).
						Return(1, []byte{}, nil)
					h.
						EXPECT().
						Compare(gomock.Any(), gomock.Any()).
						Return(nil)
					t.
						EXPECT().
						Create(gomock.Any()).
						Return("", domain.ErrInternal)
				},
			},
			expectedErr: domain.ErrInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.setupMocks != nil {
				tt.args.setupMocks(tt.fields.storage, tt.fields.tokenMng, tt.fields.hasher)
			}

			s := NewAuth(tt.fields.storage, tt.fields.tokenMng, tt.fields.hasher)

			token, err := s.LogIn(tt.args.ctx, tt.args.userData)

			if (err != nil) != (tt.expectedErr != nil) {
				t.Errorf("LogIn(): error presence mismatch: got %v, want %v", err, tt.expectedErr)
			}

			if err != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("LogIn(): got %v, want %v", err, tt.expectedErr)
			}

			if err == nil && token == "" {
				t.Errorf("LogIn(): got empty token without error")
			}
		})
	}
}
