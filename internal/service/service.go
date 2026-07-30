package service

import (
	"context"
	"fmt"

	"agreements-generator/gen/go/generator"
	"agreements-generator/internal/config"
	"agreements-generator/internal/domain"
	"agreements-generator/internal/logging"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Generator struct {
	grpcClient generator.GeneratorClient
	conn       *grpc.ClientConn
	log        logging.Logger
}

func New(cfg *config.Config, l logging.Logger) (*Generator, error) {
	URI := fmt.Sprintf("%s:%s", cfg.GRPCClient.Host, cfg.GRPCClient.Port)
	conn, err := grpc.NewClient(URI, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("can't create GRPC Client: %w", err)
	}
	return &Generator{
		grpcClient: generator.NewGeneratorClient(conn),
		conn:       conn,
		log:        l,
	}, nil
}

func (g *Generator) BulkGenerate(ctx context.Context, archiveBytes []byte) ([]byte, []*generator.FileErrors, int, error) {
	g.log.Debug("sending gRPC request",
		"archive_size", len(archiveBytes),
	)
	response, err := g.grpcClient.Generate(ctx, &generator.GenerateRequest{Archive: archiveBytes})
	if err != nil {

		return nil, []*generator.FileErrors{}, 0, domain.ErrInternal.Wrap("can't execute grpc request", err)
	}
	g.log.Debug("gRPC response received",
		"zip_size", len(response.ZipArchive),
		"errors_count", len(response.Errors),
		"generated_count", response.GeneratedCount,
	)
	return response.ZipArchive, response.Errors, int(response.GeneratedCount), nil
}

func (g *Generator) Close() error {
	return g.conn.Close()
}
