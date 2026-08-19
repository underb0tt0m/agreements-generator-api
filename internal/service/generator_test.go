package service

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"agreements-generator/internal/domain"
	"agreements-generator/internal/gen_client"
	logger_package "agreements-generator/internal/logger"
	"agreements-generator/internal/mocks"
	storage_package "agreements-generator/internal/storage"

	"go.uber.org/mock/gomock"
)

func TestGenerator_GetArchiveInfo(t *testing.T) {
	ctx := context.Background()
	logger := logger_package.NewNoop()
	ctrl := gomock.NewController(t)

	storage := mocks.NewMockGeneratorStorage(ctrl)
	storage.
		EXPECT().
		GetArchiveInfo(ctx, "success").
		Return("completed", nil, 1, "", nil)
	storage.
		EXPECT().
		GetArchiveInfo(ctx, "wrong status").
		Return("wrong status", nil, 0, "", nil)
	storage.
		EXPECT().
		GetArchiveInfo(ctx, "job not completed").
		Return("processing", nil, 0, "", nil)
	storage.
		EXPECT().
		GetArchiveInfo(ctx, "error from storage").
		Return("", nil, 0, "", domain.ErrStorageBadRequest)

	client := mocks.NewMockGeneratorClient(ctrl)

	type fields struct {
		logger  logger_package.Logger
		storage storage_package.GeneratorStorage
		client  gen_client.GeneratorClient
	}

	type args struct {
		ctx   context.Context
		jobID string
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
				storage: storage,
				client:  client,
			},
			args: args{
				ctx:   ctx,
				jobID: "success",
			},
			expectedErr: nil,
		},

		{
			name: "wrong status",
			fields: fields{
				logger:  logger,
				storage: storage,
				client:  client,
			},
			args: args{
				ctx:   ctx,
				jobID: "wrong status",
			},
			expectedErr: domain.ErrInternal,
		},

		{
			name: "job not completed",
			fields: fields{
				logger:  logger,
				storage: storage,
				client:  client,
			},
			args: args{
				ctx:   ctx,
				jobID: "job not completed",
			},
			expectedErr: domain.ErrJobNotFinished,
		},

		{
			name: "error from storage",
			fields: fields{
				logger:  logger,
				storage: storage,
				client:  client,
			},
			args: args{
				ctx:   ctx,
				jobID: "error from storage",
			},
			expectedErr: domain.ErrStorageBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := NewGen(tt.fields.logger, tt.fields.storage, tt.fields.client, "", "", 10)

			_, _, err := s.GetArchiveInfo(tt.args.ctx, tt.args.jobID)

			if (err != nil) != (tt.expectedErr != nil) {
				t.Errorf("GetArchiveInfo(): error presence mismatch: got %v, want %v", err, tt.expectedErr)
			}
			if err != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("GetArchiveInfo(): got %v, want %v", err, tt.expectedErr)
			}
		})
	}
}

func TestGenerator_GetArchive(t *testing.T) {
	ctx := context.Background()
	logger := logger_package.NewNoop()
	ctrl := gomock.NewController(t)

	storage := mocks.NewMockGeneratorStorage(ctrl)
	storage.
		EXPECT().
		GetArchive(ctx, "success").
		Return("completed", []byte{}, "", nil)
	storage.
		EXPECT().
		GetArchive(ctx, "wrong status").
		Return("wrong status", nil, "", nil)
	storage.
		EXPECT().
		GetArchive(ctx, "job not completed").
		Return("processing", nil, "", nil)
	storage.
		EXPECT().
		GetArchive(ctx, "empty archive").
		Return("completed", nil, "", nil)

	client := mocks.NewMockGeneratorClient(ctrl)

	type fields struct {
		logger  logger_package.Logger
		storage storage_package.GeneratorStorage
		client  gen_client.GeneratorClient
	}

	type args struct {
		ctx   context.Context
		jobID string
	}

	tests := []struct {
		name            string
		fields          fields
		args            args
		expectedArchive []byte
		expectedErr     error
	}{
		{
			name: "success",
			fields: fields{
				logger:  logger,
				storage: storage,
				client:  client,
			},
			args: args{
				ctx:   ctx,
				jobID: "success",
			},
			expectedArchive: []byte{},
			expectedErr:     nil,
		},

		{
			name: "wrong status",
			fields: fields{
				logger:  logger,
				storage: storage,
				client:  client,
			},
			args: args{
				ctx:   ctx,
				jobID: "wrong status",
			},
			expectedArchive: nil,
			expectedErr:     domain.ErrInternal,
		},

		{
			name: "job not completed",
			fields: fields{
				logger:  logger,
				storage: storage,
				client:  client,
			},
			args: args{
				ctx:   ctx,
				jobID: "job not completed",
			},
			expectedArchive: nil,
			expectedErr:     domain.ErrJobNotFinished,
		},

		{
			name: "empty archive",
			fields: fields{
				logger:  logger,
				storage: storage,
				client:  client,
			},
			args: args{
				ctx:   ctx,
				jobID: "empty archive",
			},
			expectedArchive: nil,
			expectedErr:     domain.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := NewGen(tt.fields.logger, tt.fields.storage, tt.fields.client, "", "", 10)

			archive, err := s.GetArchive(tt.args.ctx, tt.args.jobID)

			if (err != nil) != (tt.expectedErr != nil) {
				t.Errorf("GetArchive(): error presence mismatch: got %v, want %v", err, tt.expectedErr)
			}
			if err != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("GetArchive(): got %v, want %v", err, tt.expectedErr)
			}

			if !bytes.Equal(archive, tt.expectedArchive) {
				t.Errorf("GetArchive(): got %v, want %v", archive, tt.expectedArchive)
			}
		})
	}
}

func TestGenerator_CheckJobStatus(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	logger := logger_package.NewNoop()

	storage := mocks.NewMockGeneratorStorage(ctrl)
	storage.
		EXPECT().
		CheckJobStatus(ctx, "success").
		Return("completed", nil)
	storage.
		EXPECT().
		CheckJobStatus(ctx, "error from storage").
		Return("jfjf", domain.ErrStorageBadRequest)
	storage.
		EXPECT().
		CheckJobStatus(ctx, "invalid job status").
		Return("jfjf", nil)

	client := mocks.NewMockGeneratorClient(ctrl)

	type fields struct {
		logger  logger_package.Logger
		storage storage_package.GeneratorStorage
		client  gen_client.GeneratorClient
	}

	type args struct {
		ctx   context.Context
		jobId string
	}

	tests := []struct {
		name           string
		fields         fields
		args           args
		expectedStatus domain.JobStatus
		expectedErr    error
	}{
		{
			name: "success",
			fields: fields{
				logger:  logger,
				storage: storage,
				client:  client,
			},
			args: args{
				ctx:   ctx,
				jobId: "success",
			},
			expectedStatus: domain.StatusCompleted,
			expectedErr:    nil,
		},

		{
			name: "error from storage",
			fields: fields{
				logger:  logger,
				storage: storage,
				client:  client,
			},
			args: args{
				ctx:   ctx,
				jobId: "error from storage",
			},
			expectedStatus: domain.StatusFailed,
			expectedErr:    domain.ErrStorageBadRequest,
		},

		{
			name: "invalid job status",
			fields: fields{
				logger:  logger,
				storage: storage,
				client:  client,
			},
			args: args{
				ctx:   ctx,
				jobId: "invalid job status",
			},
			expectedStatus: domain.StatusFailed,
			expectedErr:    domain.ErrInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := NewGen(tt.fields.logger, tt.fields.storage, tt.fields.client, "", "", 10)

			status, err := s.CheckJobStatus(tt.args.ctx, tt.args.jobId)

			if (err != nil) != (tt.expectedErr != nil) {
				t.Errorf("CheckJobStatus(): error presence mismatch: got %v, want %v", err, tt.expectedErr)
			}

			if err != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("CheckJobStatus(): got %v, want %v", err, tt.expectedErr)
			}

			if status != tt.expectedStatus {
				t.Errorf("CheckJobStatus(): got %v, want %v", status, tt.expectedStatus)
			}
		})
	}
}

func TestGenerator_BulkGenerate(t *testing.T) {
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	logger := logger_package.NewNoop()

	storage := mocks.NewMockGeneratorStorage(ctrl)
	client := mocks.NewMockGeneratorClient(ctrl)

	type fields struct {
		logger  logger_package.Logger
		storage *mocks.MockGeneratorStorage
		client  *mocks.MockGeneratorClient
	}

	type args struct {
		ctx        context.Context
		archive    []byte
		setupMocks func(s *mocks.MockGeneratorStorage, c *mocks.MockGeneratorClient)
	}

	tests := []struct {
		name           string
		fields         fields
		args           args
		expectedStatus domain.JobStatus
		expectedErr    error
	}{
		{
			name: "success",
			fields: fields{
				logger:  logger,
				storage: storage,
				client:  client,
			},
			args: args{
				ctx:     ctx,
				archive: []byte("success"),
				setupMocks: func(s *mocks.MockGeneratorStorage, c *mocks.MockGeneratorClient) {
					s.
						EXPECT().
						StoreJob(ctx, gomock.Any(), gomock.Any()).
						Return(nil)
					c.
						EXPECT().
						BulkGenerate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						AnyTimes()
				},
			},
			expectedErr: nil,
		},

		{
			name: "error from storage",
			fields: fields{
				logger:  logger,
				storage: storage,
				client:  client,
			},
			args: args{
				ctx:     ctx,
				archive: []byte("error from storage"),
				setupMocks: func(s *mocks.MockGeneratorStorage, _ *mocks.MockGeneratorClient) {
					s.
						EXPECT().
						StoreJob(ctx, gomock.Any(), gomock.Any()).
						Return(domain.ErrStorageBadRequest)
				},
			},
			expectedErr: domain.ErrStorageBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.setupMocks != nil {
				tt.args.setupMocks(tt.fields.storage, tt.fields.client)
			}

			s, _ := NewGen(tt.fields.logger, tt.fields.storage, tt.fields.client, "", "", 10)

			jobID, err := s.BulkGenerate(tt.args.ctx, tt.args.archive)

			if (err != nil) != (tt.expectedErr != nil) {
				t.Errorf("BulkGenerate(): error presence mismatch: got %v, want %v", err, tt.expectedErr)
			}

			if err != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("BulkGenerate(): got %v, want %v", err, tt.expectedErr)
			}

			if err == nil && jobID == "" {
				t.Errorf("BulkGenerate(): got empty id string without error")
			}
		})
	}
}

func TestGeneratorProcessJob(t *testing.T) {
	ctrl := gomock.NewController(t)
	logger := logger_package.NewNoop()

	storage := mocks.NewMockGeneratorStorage(ctrl)
	client := mocks.NewMockGeneratorClient(ctrl)

	type fields struct {
		logger  logger_package.Logger
		storage *mocks.MockGeneratorStorage
		client  *mocks.MockGeneratorClient
	}

	type args struct {
		job        domain.Job
		archive    []byte
		setupMocks func(s *mocks.MockGeneratorStorage, c *mocks.MockGeneratorClient)
	}

	tests := []struct {
		name           string
		fields         fields
		args           args
		expectedStatus domain.JobStatus
		expectedErr    error
	}{
		{
			name: "success",
			fields: fields{
				logger:  logger,
				storage: storage,
				client:  client,
			},
			args: args{
				job:     domain.Job{ID: "success", Status: domain.StatusProcessing},
				archive: []byte("success"),
				setupMocks: func(s *mocks.MockGeneratorStorage, c *mocks.MockGeneratorClient) {
					s.
						EXPECT().
						SaveResponse(
							gomock.Any(),
							gomock.Cond(func(x domain.Job) bool {
								return x.Status == domain.StatusCompleted
							}),
							gomock.Any(),
							gomock.Any(),
						).
						Return(nil)
					s.
						EXPECT().
						UpdateJob(gomock.Any(), gomock.Cond(func(x domain.Job) bool {
							return x.Status == domain.StatusCompleted
						})).
						Return(nil)
					c.
						EXPECT().
						BulkGenerate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Do(func(
							_ context.Context,
							_ []byte,
							responseChan chan *domain.GenResponse,
							errChan chan error) {

							errChan <- nil
							responseChan <- &domain.GenResponse{}
						},
						)
				},
			},
		},

		{
			name: "error during generation",
			fields: fields{
				logger:  logger,
				storage: storage,
				client:  client,
			},
			args: args{
				job:     domain.Job{ID: "error during generation", Status: domain.StatusProcessing},
				archive: []byte("error during generation"),
				setupMocks: func(s *mocks.MockGeneratorStorage, c *mocks.MockGeneratorClient) {
					s.
						EXPECT().
						SaveResponse(
							gomock.Any(),
							gomock.Cond(func(x domain.Job) bool {
								return x.Status == domain.StatusFailed
							}),
							gomock.Any(),
							gomock.Cond(func(err error) bool {
								return errors.Is(err, domain.ErrInternal)
							}),
						).
						Return(nil)
					s.
						EXPECT().
						UpdateJob(gomock.Any(), gomock.Any()).
						Return(nil)
					c.
						EXPECT().
						BulkGenerate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Do(func(
							_ context.Context,
							_ []byte,
							responseChan chan *domain.GenResponse,
							errChan chan error) {

							errChan <- domain.ErrInternal
							responseChan <- &domain.GenResponse{}
						},
						)
				},
			},
		},

		{
			name: "error during saving",
			fields: fields{
				logger:  logger,
				storage: storage,
				client:  client,
			},
			args: args{
				job:     domain.Job{ID: "error during saving", Status: domain.StatusProcessing},
				archive: []byte("error during saving"),
				setupMocks: func(s *mocks.MockGeneratorStorage, c *mocks.MockGeneratorClient) {
					s.
						EXPECT().
						SaveResponse(
							gomock.Any(),
							gomock.Cond(func(x domain.Job) bool {
								return x.Status == domain.StatusCompleted
							}),
							gomock.Any(),
							gomock.Any(),
						).
						Return(domain.ErrStorageBadRequest)
					s.
						EXPECT().
						UpdateJob(gomock.Any(), gomock.Cond(func(x domain.Job) bool {
							return x.Status == domain.StatusFailed
						})).
						Return(nil)
					c.
						EXPECT().
						BulkGenerate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Do(func(
							_ context.Context,
							_ []byte,
							responseChan chan *domain.GenResponse,
							errChan chan error) {

							errChan <- nil
							responseChan <- &domain.GenResponse{}
						},
						)
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.setupMocks != nil {
				tt.args.setupMocks(tt.fields.storage, tt.fields.client)
			}

			s, _ := NewGen(tt.fields.logger, tt.fields.storage, tt.fields.client, "", "", 10)

			errChan := make(chan error)
			responseChan := make(chan *domain.GenResponse)

			ctx, cancel := context.WithCancel(context.Background())

			s.ProcessJob(tt.args.job, tt.args.archive, ctx, cancel, errChan, responseChan)
		})
	}
}
