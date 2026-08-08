package gen_client

import (
	"context"

	"agreements-generator/gen/go/generator"
	"agreements-generator/internal/domain"
	"agreements-generator/internal/logger"

	"google.golang.org/grpc"
)

type GeneratorClient interface {
	BulkGenerate(ctx context.Context, archive []byte, responseChan chan *domain.GenResponse, errChan chan error)
	Close() error
}

func New(grpcClient generator.GeneratorClient, conn *grpc.ClientConn, logger logger.Logger) GeneratorClient {
	return &client{grpcClient: grpcClient, conn: conn, logger: logger}
}

type client struct {
	grpcClient generator.GeneratorClient
	conn       *grpc.ClientConn
	logger     logger.Logger
}

func (c *client) BulkGenerate(
	ctx context.Context,
	archive []byte,
	responseChan chan *domain.GenResponse,
	errChan chan error,
) {
	response, err := c.grpcClient.Generate(ctx, &generator.GenerateRequest{Archive: archive})
	if err != nil {
		c.logger.Debug("can't get grpcs response", logger.FieldError, err)
		responseChan <- &domain.GenResponse{}
		errChan <- domain.ErrGenClient
		return
	}

	c.logger.Debug("grpcs response has been received")

	data := &domain.GenResponse{
		Archive:  response.ZipArchive,
		GenCount: int(response.GeneratedCount),
	}

	for _, file := range response.Errors {
		fileErrors := domain.FilesErrors{Name: file.FileName}
		for _, fileErr := range file.Errors {
			e := domain.FileError{Msg: fileErr.Message, Code: int(fileErr.Code)}
			fileErrors.Errors = append(fileErrors.Errors, e)
		}
		data.Errors = append(data.Errors, fileErrors)
	}

	c.logger.Debug("writing response into responseChan...")

	errChan <- nil
	responseChan <- data
	return
}

func (c *client) Close() error {
	return c.conn.Close()
}
