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

func TestAPI_HandleBulkGenerate(t *testing.T) {
	ctrl := gomock.NewController(t)
	logger := logger_package.NewNoop()
	service := mocks.NewMockGeneratorService(ctrl)
	encoder := mocks.NewMockEncoder(ctrl)

	type fields struct {
		logger  logger_package.Logger
		service *mocks.MockGeneratorService
		encoder *mocks.MockEncoder
	}

	type args struct {
		path       string
		mocksSetup func(s *mocks.MockGeneratorService, e *mocks.MockEncoder)
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
				path: "/bulk_generate",
				mocksSetup: func(s *mocks.MockGeneratorService, e *mocks.MockEncoder) {
					s.
						EXPECT().
						BulkGenerate(gomock.Any(), gomock.Any()).
						Return("success", nil)

					response, _ := json.Marshal(dto.BulkGenerateResponse{JobID: "success"})
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return(response, nil)
				},
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       dto.BulkGenerateResponse{JobID: "success"},
		},

		{
			name: "service error",
			fields: fields{
				logger:  logger,
				service: service,
				encoder: encoder,
			},
			args: args{
				path: "/bulk_generate",
				mocksSetup: func(s *mocks.MockGeneratorService, e *mocks.MockEncoder) {
					s.
						EXPECT().
						BulkGenerate(gomock.Any(), gomock.Any()).
						Return("", errors.New("some error"))

					response, _ := json.Marshal(dto.ErrorResponse{Details: domain.ErrInternal.Msg})
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return(response, nil)
				},
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody:       dto.ErrorResponse{Details: domain.ErrInternal.Msg},
		},

		{
			name: "encoder error",
			fields: fields{
				logger:  logger,
				service: service,
				encoder: encoder,
			},
			args: args{
				path: "/bulk_generate",
				mocksSetup: func(s *mocks.MockGeneratorService, e *mocks.MockEncoder) {
					s.
						EXPECT().
						BulkGenerate(gomock.Any(), gomock.Any()).
						Return("encoder error", nil)

					serviceAnsDTO := dto.BulkGenerateResponse{JobID: "encoder error"}
					errorDTO := dto.ErrorResponse{Details: domain.ErrInternal.Msg}
					errorResponse, _ := json.Marshal(dto.ErrorResponse{Details: domain.ErrInternal.Msg})
					e.
						EXPECT().
						Marshal(serviceAnsDTO).
						Return(nil, errors.New("some error"))
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

			h := API{
				Log:     tt.fields.logger,
				Service: tt.fields.service,
				Encoder: tt.fields.encoder,
			}

			var reqBody bytes.Buffer
			if err := json.NewEncoder(&reqBody).Encode("test"); err != nil {
				t.Fatalf("handleBulkGenerate(): can't encode reqest body: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, tt.args.path, &reqBody)
			rr := httptest.NewRecorder()

			h.handleBulkGenerate(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			switch tt.expectedBody.(type) {
			case dto.BulkGenerateResponse:
				var responseBody dto.BulkGenerateResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &responseBody); err != nil {
					t.Errorf("handleBulkGenerate(): can't unmarshal response body: %v", err)
					return
				}
				assert.Equal(t, tt.expectedBody, responseBody)
			case dto.ErrorResponse:
				var responseBody dto.ErrorResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &responseBody); err != nil {
					t.Errorf("handleBulkGenerate(): can't unmarshal response body: %v", err)
					return
				}
				assert.Equal(t, tt.expectedBody, responseBody)
			case nil:
				if rr.Body.String() != "" {
					t.Errorf("handleBulkGenerate(): not nil response body, expected nil")
					return
				}
			default:
				t.Errorf("handleBulkGenerate(): unhandled expected response scheme: %T, can handle %T, %T, %T",
					tt.expectedBody,
					dto.BulkGenerateResponse{},
					dto.ErrorResponse{},
					nil,
				)
				return
			}
		})
	}
}

func TestAPI_HandleGetJobStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	logger := logger_package.NewNoop()
	service := mocks.NewMockGeneratorService(ctrl)
	encoder := mocks.NewMockEncoder(ctrl)

	type fields struct {
		logger  logger_package.Logger
		service *mocks.MockGeneratorService
		encoder *mocks.MockEncoder
	}

	type args struct {
		path       string
		mocksSetup func(s *mocks.MockGeneratorService, e *mocks.MockEncoder)
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
				path: "/get_job_status?id=1",
				mocksSetup: func(s *mocks.MockGeneratorService, e *mocks.MockEncoder) {
					s.
						EXPECT().
						CheckJobStatus(gomock.Any(), gomock.Any()).
						Return(domain.StatusCompleted, nil)

					response, _ := json.Marshal(dto.GetJobStatusResponse{JobID: "1", Status: "completed"})
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return(response, nil)
				},
			},
			expectedStatusCode: http.StatusOK,
			expectedBody: dto.GetJobStatusResponse{
				JobID:  "1",
				Status: "completed",
			},
		},

		{
			name: "empty id",
			fields: fields{
				logger:  logger,
				service: service,
				encoder: encoder,
			},
			args: args{
				path: "/get_job_status",
				mocksSetup: func(s *mocks.MockGeneratorService, e *mocks.MockEncoder) {
					response, _ := json.Marshal(dto.ErrorResponse{Details: domain.ErrBadRequest.Msg})
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return(response, nil)
				},
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedBody: dto.ErrorResponse{
				Details: domain.ErrBadRequest.Msg,
			},
		},

		{
			name: "service error",
			fields: fields{
				logger:  logger,
				service: service,
				encoder: encoder,
			},
			args: args{
				path: "/get_job_status?id=1",
				mocksSetup: func(s *mocks.MockGeneratorService, e *mocks.MockEncoder) {
					s.
						EXPECT().
						CheckJobStatus(gomock.Any(), gomock.Any()).
						Return(domain.StatusFailed, errors.New("some error"))

					response, _ := json.Marshal(dto.ErrorResponse{Details: domain.ErrInternal.Msg})
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return(response, nil)
				},
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody: dto.ErrorResponse{
				Details: domain.ErrInternal.Msg,
			},
		},

		{
			name: "encoder error",
			fields: fields{
				logger:  logger,
				service: service,
				encoder: encoder,
			},
			args: args{
				path: "/get_job_status?id=1",
				mocksSetup: func(s *mocks.MockGeneratorService, e *mocks.MockEncoder) {
					s.
						EXPECT().
						CheckJobStatus(gomock.Any(), gomock.Any()).
						Return(domain.StatusCompleted, nil)

					dtoService := dto.GetJobStatusResponse{
						JobID:  "1",
						Status: "completed",
					}
					dtoError := dto.ErrorResponse{Details: domain.ErrInternal.Msg}
					errorResponse, _ := json.Marshal(dtoError)
					e.
						EXPECT().
						Marshal(dtoService).
						Return(nil, errors.New("some error"))
					e.
						EXPECT().
						Marshal(dtoError).
						Return(errorResponse, nil)
				},
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody: dto.ErrorResponse{
				Details: domain.ErrInternal.Msg,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.mocksSetup != nil {
				tt.args.mocksSetup(tt.fields.service, tt.fields.encoder)
			}

			h := API{
				Log:     tt.fields.logger,
				Service: tt.fields.service,
				Encoder: tt.fields.encoder,
			}

			req := httptest.NewRequest(http.MethodGet, tt.args.path, nil)
			rr := httptest.NewRecorder()

			h.handleGetJobStatus(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			switch tt.expectedBody.(type) {
			case dto.GetJobStatusResponse:
				var responseBody dto.GetJobStatusResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &responseBody); err != nil {
					t.Errorf("handleGetJobStatus(): can't unmarshal response body: %v", err)
					return
				}
				assert.Equal(t, tt.expectedBody, responseBody)
			case dto.ErrorResponse:
				var responseBody dto.ErrorResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &responseBody); err != nil {
					t.Errorf("handleGetJobStatus(): can't unmarshal response body: %v", err)
					return
				}
				assert.Equal(t, tt.expectedBody, responseBody)
			case nil:
				if rr.Body.String() != "" {
					t.Errorf("handleGetJobStatus(): not nil response body, expected nil")
					return
				}
			default:
				t.Errorf("handleGetJobStatus(): unhandled expected response scheme: %T, can handle %T, %T, %T",
					tt.expectedBody,
					dto.GetJobStatusResponse{},
					dto.ErrorResponse{},
					nil,
				)
				return
			}
		})
	}
}

func TestAPI_HandleGetArchive(t *testing.T) {
	ctrl := gomock.NewController(t)
	logger := logger_package.NewNoop()
	service := mocks.NewMockGeneratorService(ctrl)
	encoder := mocks.NewMockEncoder(ctrl)

	type fields struct {
		logger  logger_package.Logger
		service *mocks.MockGeneratorService
		encoder *mocks.MockEncoder
	}

	type args struct {
		path       string
		mocksSetup func(s *mocks.MockGeneratorService, e *mocks.MockEncoder)
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
				path: "/get_archive?id=1",
				mocksSetup: func(s *mocks.MockGeneratorService, e *mocks.MockEncoder) {
					s.
						EXPECT().
						GetArchive(gomock.Any(), gomock.Any()).
						Return([]byte("success"), nil)
				},
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       []byte("success"),
		},

		{
			name: "empty id",
			fields: fields{
				logger:  logger,
				service: service,
				encoder: encoder,
			},
			args: args{
				path: "/get_archive",
				mocksSetup: func(s *mocks.MockGeneratorService, e *mocks.MockEncoder) {
					response, _ := json.Marshal(dto.ErrorResponse{Details: domain.ErrBadRequest.Msg})
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return(response, nil)
				},
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       dto.ErrorResponse{Details: domain.ErrBadRequest.Msg},
		},

		{
			name: "service error",
			fields: fields{
				logger:  logger,
				service: service,
				encoder: encoder,
			},
			args: args{
				path: "/get_archive?id=1",
				mocksSetup: func(s *mocks.MockGeneratorService, e *mocks.MockEncoder) {
					s.
						EXPECT().
						GetArchive(gomock.Any(), gomock.Any()).
						Return(nil, errors.New("some error"))

					response, _ := json.Marshal(dto.ErrorResponse{Details: domain.ErrInternal.Msg})
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return(response, nil)
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

			h := API{
				Log:     tt.fields.logger,
				Service: tt.fields.service,
				Encoder: tt.fields.encoder,
			}

			req := httptest.NewRequest(http.MethodGet, tt.args.path, nil)
			rr := httptest.NewRecorder()

			h.handleGetArchive(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			switch tt.expectedBody.(type) {
			case []byte:
				assert.Equal(t, tt.expectedBody, rr.Body.Bytes())
			case dto.ErrorResponse:
				var responseBody dto.ErrorResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &responseBody); err != nil {
					t.Errorf("handleGetArchive(): can't unmarshal response body: %v", err)
					return
				}
				assert.Equal(t, tt.expectedBody, responseBody)
			case nil:
				if rr.Body.String() != "" {
					t.Errorf("handleGetArchive(): not nil response body, expected nil")
					return
				}
			default:
				t.Errorf("handleGetArchive(): unhandled expected response scheme: %T, can handle %T, %T, %T",
					tt.expectedBody,
					[]byte{},
					dto.ErrorResponse{},
					nil,
				)
				return
			}
		})
	}
}

func TestAPI_HandleGetArchiveInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	logger := logger_package.NewNoop()
	service := mocks.NewMockGeneratorService(ctrl)
	encoder := mocks.NewMockEncoder(ctrl)

	type fields struct {
		logger  logger_package.Logger
		service *mocks.MockGeneratorService
		encoder *mocks.MockEncoder
	}

	type args struct {
		path       string
		mocksSetup func(s *mocks.MockGeneratorService, e *mocks.MockEncoder)
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
				path: "/get_archive_info?id=1",
				mocksSetup: func(s *mocks.MockGeneratorService, e *mocks.MockEncoder) {
					s.
						EXPECT().
						GetArchiveInfo(gomock.Any(), gomock.Any()).
						Return([]domain.FilesErrors{}, 1, nil)

					response, _ := json.Marshal(dto.GetArchiveInfoResponse{GenCnt: 1})
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return(response, nil)
				},
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       dto.GetArchiveInfoResponse{GenCnt: 1},
		},

		{
			name: "empty id",
			fields: fields{
				logger:  logger,
				service: service,
				encoder: encoder,
			},
			args: args{
				path: "/get_archive_info",
				mocksSetup: func(s *mocks.MockGeneratorService, e *mocks.MockEncoder) {
					response, _ := json.Marshal(dto.ErrorResponse{Details: domain.ErrBadRequest.Msg})
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return(response, nil)
				},
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       dto.ErrorResponse{Details: domain.ErrBadRequest.Msg},
		},

		{
			name: "service error",
			fields: fields{
				logger:  logger,
				service: service,
				encoder: encoder,
			},
			args: args{
				path: "/get_archive_info?id=1",
				mocksSetup: func(s *mocks.MockGeneratorService, e *mocks.MockEncoder) {
					s.
						EXPECT().
						GetArchiveInfo(gomock.Any(), gomock.Any()).
						Return(nil, 0, errors.New("some error"))

					response, _ := json.Marshal(dto.ErrorResponse{Details: domain.ErrInternal.Msg})
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return(response, nil)
				},
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody:       dto.ErrorResponse{Details: domain.ErrInternal.Msg},
		},

		{
			name: "encoder error",
			fields: fields{
				logger:  logger,
				service: service,
				encoder: encoder,
			},
			args: args{
				path: "/get_archive_info?id=1",
				mocksSetup: func(s *mocks.MockGeneratorService, e *mocks.MockEncoder) {
					s.
						EXPECT().
						GetArchiveInfo(gomock.Any(), gomock.Any()).
						Return([]domain.FilesErrors{}, 1, nil)

					dtoService := dto.GetArchiveInfoResponse{
						GenErrs: []dto.FilesErrors{},
						GenCnt:  1,
					}
					dtoError := dto.ErrorResponse{Details: domain.ErrInternal.Msg}
					errorResponse, _ := json.Marshal(dtoError)

					e.
						EXPECT().
						Marshal(dtoService).
						Return(nil, errors.New("some error"))
					e.
						EXPECT().
						Marshal(dtoError).
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

			h := API{
				Log:     tt.fields.logger,
				Service: tt.fields.service,
				Encoder: tt.fields.encoder,
			}

			req := httptest.NewRequest(http.MethodGet, tt.args.path, nil)
			rr := httptest.NewRecorder()

			h.handleGetArchiveInfo(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			switch tt.expectedBody.(type) {
			case dto.GetArchiveInfoResponse:
				var responseBody dto.GetArchiveInfoResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &responseBody); err != nil {
					t.Errorf("handleGetArchiveInfo(): can't unmarshal response body: %v", err)
					return
				}
				assert.Equal(t, tt.expectedBody, responseBody)
			case dto.ErrorResponse:
				var responseBody dto.ErrorResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &responseBody); err != nil {
					t.Errorf("handleGetArchiveInfo(): can't unmarshal response body: %v", err)
					return
				}
				assert.Equal(t, tt.expectedBody, responseBody)
			case nil:
				if rr.Body.String() != "" {
					t.Errorf("handleGetArchiveInfo(): not nil response body, expected nil")
					return
				}
			default:
				t.Errorf("handleGetArchiveInfo(): unhandled expected response scheme: %T, can handle %T, %T, %T",
					tt.expectedBody,
					dto.GetArchiveInfoResponse{},
					dto.ErrorResponse{},
					nil,
				)
				return
			}
		})
	}
}

func Test_WriteError(t *testing.T) {
	ctrl := gomock.NewController(t)
	logger := logger_package.NewNoop()
	encoder := mocks.NewMockEncoder(ctrl)

	type fields struct {
		logger  logger_package.Logger
		encoder *mocks.MockEncoder
	}

	type args struct {
		error      error
		mocksSetup func(e *mocks.MockEncoder)
	}

	tests := []struct {
		name               string
		fields             fields
		args               args
		expectedStatusCode int
		expectedBody       any
	}{
		{
			name: "appErr",
			fields: fields{
				logger:  logger,
				encoder: encoder,
			},
			args: args{
				error: fmt.Errorf("some error: %w", domain.ErrBadRequest),
				mocksSetup: func(e *mocks.MockEncoder) {
					response, _ := json.Marshal(dto.ErrorResponse{Details: domain.ErrBadRequest.Msg})
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return(response, nil)
				},
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       dto.ErrorResponse{Details: domain.ErrBadRequest.Msg},
		},

		{
			name: "unexpected error",
			fields: fields{
				logger:  logger,
				encoder: encoder,
			},
			args: args{
				error: fmt.Errorf("some error"),
				mocksSetup: func(e *mocks.MockEncoder) {
					response, _ := json.Marshal(dto.ErrorResponse{Details: domain.ErrInternal.Msg})
					e.
						EXPECT().
						Marshal(gomock.Any()).
						Return(response, nil)
				},
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody:       dto.ErrorResponse{Details: domain.ErrInternal.Msg},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.mocksSetup != nil {
				tt.args.mocksSetup(tt.fields.encoder)
			}

			rr := httptest.NewRecorder()

			writeError(rr, tt.args.error, tt.fields.encoder, tt.fields.logger)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			switch tt.expectedBody.(type) {
			case dto.ErrorResponse:
				var responseBody dto.ErrorResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &responseBody); err != nil {
					t.Errorf("writeError(): can't unmarshal response body: %v", err)
					return
				}
				assert.Equal(t, tt.expectedBody, responseBody)
			case nil:
				if rr.Body.String() != "" {
					t.Errorf("writeError(): not nil response body, expected nil")
					return
				}
			default:
				t.Errorf("writeError(): unhandled expected response scheme: %T, can handle %T, %T",
					tt.expectedBody,
					dto.ErrorResponse{},
					nil,
				)
				return
			}
		})
	}
}
