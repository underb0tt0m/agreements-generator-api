package gen_client

import (
	"context"
	"errors"
	"testing"

	"agreements-generator/gen/go/generator"
	"agreements-generator/internal/domain"
	loggerModule "agreements-generator/internal/logger"
	"agreements-generator/internal/mocks"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
)

//go:generate mockgen -source=client_test.go -destination=../mocks/grpcClient.go -package=mocks
type GeneratorClientNoOp interface {
	Generate(ctx context.Context, in *generator.GenerateRequest, opts ...grpc.CallOption) (*generator.GenerateResponse, error)
}

func TestClient_BulkGenerate(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	logger := loggerModule.NewNoop()
	grpcClient := mocks.NewMockGeneratorClientNoOp(ctrl)
	grpcConn := &grpc.ClientConn{}

	type fields struct {
		grpcClient *mocks.MockGeneratorClientNoOp
		grpcConn   *grpc.ClientConn
		logger     loggerModule.Logger
	}

	type args struct {
		ctx       context.Context
		archive   []byte
		mockSetup func(c *mocks.MockGeneratorClientNoOp)
	}

	tests := []struct {
		name             string
		fields           fields
		args             args
		expectedErr      error
		expectedResponse *domain.GenResponse
	}{
		{
			name: "success",
			fields: fields{
				grpcClient: grpcClient,
				grpcConn:   grpcConn,
				logger:     logger,
			},
			args: args{
				ctx:     ctx,
				archive: []byte("success"),
				mockSetup: func(c *mocks.MockGeneratorClientNoOp) {
					c.
						EXPECT().
						Generate(gomock.Any(), gomock.Any()).
						Return(&generator.GenerateResponse{
							ZipArchive:     []byte("success"),
							GeneratedCount: 2,
							Errors: []*generator.FileErrors{
								{
									FileName: "success",
									Errors: []*generator.FileError{
										{Message: "success"},
									},
								},
							},
						}, nil)
				},
			},
			expectedErr: nil,
			expectedResponse: &domain.GenResponse{
				Archive:  []byte("success"),
				GenCount: 2,
				Errors: []domain.FilesErrors{
					{
						Name: "success",
						Errors: []domain.FileError{
							{Msg: "success"},
						},
					},
				},
			},
		},

		{
			name: "error from client",
			fields: fields{
				grpcClient: grpcClient,
				grpcConn:   grpcConn,
				logger:     logger,
			},
			args: args{
				ctx:     ctx,
				archive: []byte("success"),
				mockSetup: func(c *mocks.MockGeneratorClientNoOp) {
					c.
						EXPECT().
						Generate(gomock.Any(), gomock.Any()).
						Return(nil, errors.New("some error"))
				},
			},
			expectedErr:      domain.ErrGenClient,
			expectedResponse: &domain.GenResponse{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.mockSetup != nil {
				tt.args.mockSetup(tt.fields.grpcClient)
			}

			respChan := make(chan *domain.GenResponse)
			errChan := make(chan error)

			cl := New(tt.fields.grpcClient, tt.fields.grpcConn, tt.fields.logger)

			go cl.BulkGenerate(tt.args.ctx, tt.args.archive, respChan, errChan)

			err := <-errChan
			response := <-respChan

			if (err != nil) != (tt.expectedErr != nil) {
				t.Errorf("BulkGenerate(): error presence mismatch: got %v, want %v", err, tt.expectedErr)
			}
			if err != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("BulkGenerate(): got %v, want %v", err, tt.expectedErr)
			}

			if !assert.Equal(t, tt.expectedResponse, response) {
				t.Errorf("BulkGenerate(): got %v, want %v", response, tt.expectedResponse)
			}

		})
	}
}
