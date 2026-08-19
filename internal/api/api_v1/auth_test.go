package api_v1

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"agreements-generator/internal/domain"
	"agreements-generator/internal/dto"
	logger_package "agreements-generator/internal/logger"
	"agreements-generator/internal/mocks"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestAuth_handleRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	logger := logger_package.NewNoop()
	service := mocks.NewMockAuthService(ctrl)
	encoder := mocks.NewMockEncoder(ctrl)

	type fields struct {
		logger  logger_package.Logger
		service *mocks.MockAuthService
		encoder *mocks.MockEncoder
	}

	type args struct {
		mocksSetup func(s *mocks.MockAuthService, e *mocks.MockEncoder)
	}

	tests := []struct {
		name               string
		fields             fields
		args               args
		expectedStatusCode int
		expectedBody       any
	}{
		{
			name: "success",
			fields: fields{
				logger:  logger,
				service: service,
				encoder: encoder,
			},
			args: args{
				mocksSetup: func(s *mocks.MockAuthService, e *mocks.MockEncoder) {
					e.
						EXPECT().
						Unmarshal(gomock.Any(), gomock.Any()).
						DoAndReturn(func(data []byte, v any) error { return nil })

					s.
						EXPECT().
						Register(gomock.Any(), gomock.Any()).
						Return("token123", nil)

					response, _ := json.Marshal(dto.AuthResponse{Token: "token123"})
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return(response, nil)
				},
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       dto.AuthResponse{Token: "token123"},
		},

		{
			name: "unmarshal error",
			fields: fields{
				logger:  logger,
				service: service,
				encoder: encoder,
			},
			args: args{
				mocksSetup: func(s *mocks.MockAuthService, e *mocks.MockEncoder) {
					e.
						EXPECT().
						Unmarshal(gomock.Any(), gomock.Any()).
						Return(errors.New("invalid json"))

					errorResponse, _ := json.Marshal(dto.ErrorResponse{Details: domain.ErrUnprocessableEntity.Msg})
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return(errorResponse, nil)
				},
			},
			expectedStatusCode: http.StatusUnprocessableEntity,
			expectedBody:       dto.ErrorResponse{Details: domain.ErrUnprocessableEntity.Msg},
		},

		{
			name: "service error",
			fields: fields{
				logger:  logger,
				service: service,
				encoder: encoder,
			},
			args: args{
				mocksSetup: func(s *mocks.MockAuthService, e *mocks.MockEncoder) {
					e.
						EXPECT().
						Unmarshal(gomock.Any(), gomock.Any()).
						DoAndReturn(func(data []byte, v any) error { return nil })

					s.
						EXPECT().
						Register(gomock.Any(), gomock.Any()).
						Return("", fmt.Errorf("some error: %w", domain.ErrBadRequest))

					errorResponse, _ := json.Marshal(dto.ErrorResponse{Details: domain.ErrBadRequest.Msg})
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return(errorResponse, nil)
				},
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       dto.ErrorResponse{Details: domain.ErrBadRequest.Msg},
		},

		{
			name: "marshal response error",
			fields: fields{
				logger:  logger,
				service: service,
				encoder: encoder,
			},
			args: args{
				mocksSetup: func(s *mocks.MockAuthService, e *mocks.MockEncoder) {
					e.
						EXPECT().
						Unmarshal(gomock.Any(), gomock.Any()).
						DoAndReturn(func(data []byte, v any) error { return nil })

					s.
						EXPECT().
						Register(gomock.Any(), gomock.Any()).
						Return("token123", nil)

					serviceDTO := dto.AuthResponse{Token: "token123"}
					errorDTO := dto.ErrorResponse{Details: domain.ErrInternal.Msg}
					errorResponse, _ := json.Marshal(errorDTO)

					e.
						EXPECT().
						Marshal(serviceDTO).
						Return(nil, errors.New("marshal failed"))
					e.
						EXPECT().
						Marshal(errorDTO).
						Return(errorResponse, nil)
				},
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody:       dto.ErrorResponse{Details: domain.ErrInternal.Msg},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.mocksSetup != nil {
				tt.args.mocksSetup(tt.fields.service, tt.fields.encoder)
			}

			h := Auth{
				Log:     tt.fields.logger,
				Service: tt.fields.service,
				Encoder: tt.fields.encoder,
			}

			var reqBody bytes.Buffer
			if err := json.NewEncoder(&reqBody).Encode(dto.RegisterRequest{
				Name:     "test",
				Login:    "test",
				Password: "test",
			}); err != nil {
				t.Fatalf("can't encode request body: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/register", &reqBody)
			rr := httptest.NewRecorder()

			h.handleRegister(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			switch expected := tt.expectedBody.(type) {
			case dto.AuthResponse:
				var responseBody dto.AuthResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &responseBody); err != nil {
					t.Errorf("handleRegister(): can't unmarshal response: %v", err)
					return
				}
				assert.Equal(t, expected, responseBody)
			case dto.ErrorResponse:
				var responseBody dto.ErrorResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &responseBody); err != nil {
					t.Errorf("handleRegister(): can't unmarshal response: %v", err)
					return
				}
				assert.Equal(t, expected, responseBody)
			default:
				t.Errorf("handleRegister(): unhandled expected response scheme: %T, can handle %T, %T, %T",
					tt.expectedBody,
					dto.AuthResponse{},
					dto.ErrorResponse{},
					nil,
				)
				return
			}
		})
	}
}

func TestAuth_handleLogIn(t *testing.T) {
	ctrl := gomock.NewController(t)
	logger := logger_package.NewNoop()
	service := mocks.NewMockAuthService(ctrl)
	encoder := mocks.NewMockEncoder(ctrl)

	type fields struct {
		logger  logger_package.Logger
		service *mocks.MockAuthService
		encoder *mocks.MockEncoder
	}

	type args struct {
		mocksSetup func(s *mocks.MockAuthService, e *mocks.MockEncoder)
	}

	tests := []struct {
		name               string
		fields             fields
		args               args
		expectedStatusCode int
		expectedBody       any
	}{
		{
			name: "success",
			fields: fields{
				logger:  logger,
				service: service,
				encoder: encoder,
			},
			args: args{
				mocksSetup: func(s *mocks.MockAuthService, e *mocks.MockEncoder) {
					e.
						EXPECT().
						Unmarshal(gomock.Any(), gomock.Any()).
						DoAndReturn(func(data []byte, v any) error { return nil })

					s.
						EXPECT().
						LogIn(gomock.Any(), gomock.Any()).
						Return("token123", nil)

					response, _ := json.Marshal(dto.AuthResponse{Token: "token123"})
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return(response, nil)
				},
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       dto.AuthResponse{Token: "token123"},
		},

		{
			name: "unmarshal error",
			fields: fields{
				logger:  logger,
				service: service,
				encoder: encoder,
			},
			args: args{
				mocksSetup: func(s *mocks.MockAuthService, e *mocks.MockEncoder) {
					e.
						EXPECT().
						Unmarshal(gomock.Any(), gomock.Any()).
						Return(errors.New("invalid json"))

					errorResponse, _ := json.Marshal(dto.ErrorResponse{Details: domain.ErrUnprocessableEntity.Msg})
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return(errorResponse, nil)
				},
			},
			expectedStatusCode: http.StatusUnprocessableEntity,
			expectedBody:       dto.ErrorResponse{Details: domain.ErrUnprocessableEntity.Msg},
		},

		{
			name: "service error",
			fields: fields{
				logger:  logger,
				service: service,
				encoder: encoder,
			},
			args: args{
				mocksSetup: func(s *mocks.MockAuthService, e *mocks.MockEncoder) {
					e.
						EXPECT().
						Unmarshal(gomock.Any(), gomock.Any()).
						DoAndReturn(func(data []byte, v any) error { return nil })

					s.
						EXPECT().
						LogIn(gomock.Any(), gomock.Any()).
						Return("", fmt.Errorf("some error: %w", domain.ErrBadRequest))

					errorResponse, _ := json.Marshal(dto.ErrorResponse{Details: domain.ErrBadRequest.Msg})
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return(errorResponse, nil)
				},
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       dto.ErrorResponse{Details: domain.ErrBadRequest.Msg},
		},

		{
			name: "marshal response error",
			fields: fields{
				logger:  logger,
				service: service,
				encoder: encoder,
			},
			args: args{
				mocksSetup: func(s *mocks.MockAuthService, e *mocks.MockEncoder) {
					e.
						EXPECT().
						Unmarshal(gomock.Any(), gomock.Any()).
						DoAndReturn(func(data []byte, v any) error { return nil })

					s.
						EXPECT().
						LogIn(gomock.Any(), gomock.Any()).
						Return("token123", nil)

					serviceDTO := dto.AuthResponse{Token: "token123"}
					errorDTO := dto.ErrorResponse{Details: domain.ErrInternal.Msg}
					errorResponse, _ := json.Marshal(errorDTO)

					e.
						EXPECT().
						Marshal(serviceDTO).
						Return(nil, errors.New("marshal failed"))
					e.
						EXPECT().
						Marshal(errorDTO).
						Return(errorResponse, nil)
				},
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody:       dto.ErrorResponse{Details: domain.ErrInternal.Msg},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.mocksSetup != nil {
				tt.args.mocksSetup(tt.fields.service, tt.fields.encoder)
			}

			h := Auth{
				Log:     tt.fields.logger,
				Service: tt.fields.service,
				Encoder: tt.fields.encoder,
			}

			var reqBody bytes.Buffer
			if err := json.NewEncoder(&reqBody).Encode(dto.LogInRequest{
				Login:    "test",
				Password: "test",
			}); err != nil {
				t.Fatalf("can't encode request body: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/login", &reqBody)
			rr := httptest.NewRecorder()

			h.handleLogIn(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			switch expected := tt.expectedBody.(type) {
			case dto.AuthResponse:
				var responseBody dto.AuthResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &responseBody); err != nil {
					t.Errorf("handleLogIn(): can't unmarshal response: %v", err)
					return
				}
				assert.Equal(t, expected, responseBody)
			case dto.ErrorResponse:
				var responseBody dto.ErrorResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &responseBody); err != nil {
					t.Errorf("handleLogIn(): can't unmarshal response: %v", err)
					return
				}
				assert.Equal(t, expected, responseBody)
			default:
				t.Errorf("handleLogIn(): unhandled expected response scheme: %T, can handle %T, %T, %T",
					tt.expectedBody,
					dto.AuthResponse{},
					dto.ErrorResponse{},
					nil,
				)
				return
			}
		})
	}
}
