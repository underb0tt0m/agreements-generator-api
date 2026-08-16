package token_manager

import (
	"errors"
	"testing"
	"time"

	"agreements-generator/internal/config"
	"agreements-generator/internal/domain"
	logger_package "agreements-generator/internal/logger"
	"agreements-generator/internal/mocks"

	"go.uber.org/mock/gomock"
)

func TestTokenMaker_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	logger := logger_package.NewNoop()
	encoder := mocks.NewMockEncoder(ctrl)

	type fields struct {
		logger  logger_package.Logger
		encoder *mocks.MockEncoder
	}

	type args struct {
		data          any
		signingMethod string
		setupMocks    func(e *mocks.MockEncoder)
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
				logger:  logger,
				encoder: encoder,
			},
			args: args{
				data: UserClaims{
					Login: "success",
				},
				signingMethod: config.HS256,
				setupMocks: func(e *mocks.MockEncoder) {
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return([]byte{}, nil)
					e.
						EXPECT().
						Unmarshal(gomock.Any(), gomock.Any()).
						Return(nil)
				},
			},
			expectedErr: nil,
		},

		{
			name: "marshalling error",
			fields: fields{
				logger:  logger,
				encoder: encoder,
			},
			args: args{
				data: UserClaims{
					Login: "marshalling error",
				},
				signingMethod: config.HS256,
				setupMocks: func(e *mocks.MockEncoder) {
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return([]byte{}, errors.New("some error"))

				},
			},
			expectedErr: domain.ErrInternal,
		},

		{
			name: "unmarshalling error",
			fields: fields{
				logger:  logger,
				encoder: encoder,
			},
			args: args{
				data: UserClaims{
					Login: "unmarshalling error",
				},
				signingMethod: config.HS256,
				setupMocks: func(e *mocks.MockEncoder) {
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return([]byte{}, nil)
					e.
						EXPECT().
						Unmarshal(gomock.Any(), gomock.Any()).
						Return(errors.New("some error"))
				},
			},
			expectedErr: domain.ErrInternal,
		},

		{
			name: "signing error",
			fields: fields{
				logger:  logger,
				encoder: encoder,
			},
			args: args{
				data: UserClaims{
					Login: "signing error",
				},
				signingMethod: config.RS256,
				setupMocks: func(e *mocks.MockEncoder) {
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return([]byte{}, nil)
					e.
						EXPECT().
						Unmarshal(gomock.Any(), gomock.Any()).
						Return(nil)
				},
			},
			expectedErr: domain.ErrInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.setupMocks != nil {
				tt.args.setupMocks(tt.fields.encoder)
			}

			tokenMng := New(tt.fields.logger, tt.fields.encoder, time.Second, tt.args.signingMethod, "123", "Bearer")

			token, err := tokenMng.Create(tt.args.data)

			if (err != nil) != (tt.expectedErr != nil) {
				t.Errorf("Create(): error presence mismatch: got %v, want %v", err, tt.expectedErr)
			}

			if err != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("Create(): got %v, want %v", err, tt.expectedErr)
			}

			if err == nil && token == "" {
				t.Errorf("Create(): got empty token without error")
			}
		})
	}
}

func TestTokenMaker_Validate(t *testing.T) {
	ctrl := gomock.NewController(t)
	logger := logger_package.NewNoop()
	encoder := mocks.NewMockEncoder(ctrl)

	type fields struct {
		logger  logger_package.Logger
		encoder *mocks.MockEncoder
	}

	type args struct {
		ttl        time.Duration
		setupMocks func(e *mocks.MockEncoder)
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
				logger:  logger,
				encoder: encoder,
			},
			args: args{
				ttl: time.Minute,
				setupMocks: func(e *mocks.MockEncoder) {
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return([]byte{}, nil)
					e.
						EXPECT().
						Unmarshal(gomock.Any(), gomock.Any()).
						Return(nil)
				},
			},
			expectedErr: nil,
		},

		{
			name: "invalid token",
			fields: fields{
				logger:  logger,
				encoder: encoder,
			},
			args: args{
				ttl: time.Millisecond,
				setupMocks: func(e *mocks.MockEncoder) {
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return([]byte{}, nil)
					e.
						EXPECT().
						Unmarshal(gomock.Any(), gomock.Any()).
						Return(nil)
				},
			},
			expectedErr: domain.ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.setupMocks != nil {
				tt.args.setupMocks(tt.fields.encoder)
			}

			tokenMng := New(tt.fields.logger, tt.fields.encoder, tt.args.ttl, config.HS256, "123", "Bearer")

			token, _ := tokenMng.Create(UserClaims{Login: "login"})

			err := tokenMng.Validate(token)

			if (err != nil) != (tt.expectedErr != nil) {
				t.Errorf("Validate(): error presence mismatch: got %v, want %v", err, tt.expectedErr)
			}

			if err != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("Validate(): got %v, want %v", err, tt.expectedErr)
			}
		})
	}
}

func TestTokenMaker_Parse(t *testing.T) {
	ctrl := gomock.NewController(t)
	logger := logger_package.NewNoop()
	encoder := mocks.NewMockEncoder(ctrl)

	type fields struct {
		logger  logger_package.Logger
		encoder *mocks.MockEncoder
	}

	type args struct {
		genSecret   string
		checkSecret string
		setupMocks  func(e *mocks.MockEncoder)
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		expectedErr error
	}{
		{
			name: "Valid Token",
			fields: fields{
				logger:  logger,
				encoder: encoder,
			},
			args: args{
				genSecret:   "123",
				checkSecret: "123",
				setupMocks: func(e *mocks.MockEncoder) {
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return([]byte{}, nil).
						AnyTimes()
					e.
						EXPECT().
						Unmarshal(gomock.Any(), gomock.Any()).
						Return(nil).
						AnyTimes()
				},
			},
			expectedErr: nil,
		},

		{
			name: "Invalid Token",
			fields: fields{
				logger:  logger,
				encoder: encoder,
			},
			args: args{
				genSecret:   "123",
				checkSecret: "456",
				setupMocks: func(e *mocks.MockEncoder) {
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return([]byte{}, nil).
						AnyTimes()
					e.
						EXPECT().
						Unmarshal(gomock.Any(), gomock.Any()).
						Return(nil).
						AnyTimes()
				},
			},
			expectedErr: domain.ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.setupMocks != nil {
				tt.args.setupMocks(tt.fields.encoder)
			}

			genTokenMng := New(tt.fields.logger, tt.fields.encoder, time.Second, config.HS256, tt.args.genSecret, "Bearer")
			checkTokenMng := New(tt.fields.logger, tt.fields.encoder, time.Second, config.HS256, tt.args.checkSecret, "Bearer")

			genData := UserClaims{
				Login: "login",
			}
			parseData := struct {
				login string
			}{}

			token, _ := genTokenMng.Create(genData)
			err := checkTokenMng.Parse(token, &parseData)

			if (err != nil) != (tt.expectedErr != nil) {
				t.Errorf("Parse(): error presence mismatch: got %v, want %v", err, tt.expectedErr)
			}

			if err != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("Parse(): got %v, want %v", err, tt.expectedErr)
			}
		})
	}
}
