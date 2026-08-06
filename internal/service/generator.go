package service

import (
	"context"
	"fmt"

	"agreements-generator/gen/go/generator"
	"agreements-generator/internal/config"
	"agreements-generator/internal/domain"
	"agreements-generator/internal/logging"
	"agreements-generator/internal/storage"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Generator struct {
	grpcClient generator.GeneratorClient
	conn       *grpc.ClientConn
	log        logging.Logger
	storage    storage.GeneratorStorage
	cfg        *config.Config
}

func NewGen(cfg *config.Config, l logging.Logger, s storage.GeneratorStorage) (*Generator, error) {
	URI := fmt.Sprintf("%s:%s", cfg.GRPCClient.Host, cfg.GRPCClient.Port)
	conn, err := grpc.NewClient(URI, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("can't create GRPC Client: %w", err)
	}
	return &Generator{
		grpcClient: generator.NewGeneratorClient(conn),
		conn:       conn,
		log:        l,
		storage:    s,
		cfg:        cfg,
	}, nil
}

func (g *Generator) BulkGenerate(ctx context.Context, archiveBytes []byte) (string, error) {
	g.log.Debug("sending gRPC request",
		"archive_size", len(archiveBytes),
	)

	id, err := uuid.NewUUID()
	if err != nil {
		return "", domain.ErrInternal.Wrap("can't create job", err)
	}

	job := domain.Job{
		ID:     id.String(),
		Status: domain.StatusProcessing,
	}

	if err = g.storage.StoreJob(ctx, job); err != nil {
		return "", domain.ErrInternal.Wrap("can't create job", err)
	}

	g.log.Info("connecting to gRPC",
		"host", g.cfg.GRPCClient.Host,
		"port", g.cfg.GRPCClient.Port,
	)

	jobCtx, cancel := context.WithTimeout(context.Background(), g.cfg.GRPCClient.JobMaxDuration)
	go func() {
		defer cancel()
		g.grpcGenerate(jobCtx, &generator.GenerateRequest{Archive: archiveBytes}, job)
	}()

	return job.ID, nil
}

func (g *Generator) CheckJobStatus(ctx context.Context, id string) (domain.JobStatus, error) {
	status, err := g.storage.CheckJobStatus(ctx, id)
	if err != nil {
		return "", domain.CheckAppErr(err)
	}

	return domain.IsJobStatus(status)
}

func (g *Generator) GetArchive(ctx context.Context, jobID string) ([]byte, error) {
	archive, err := g.storage.GetArchive(ctx, jobID)
	if err != nil {
		return nil, domain.CheckAppErr(err)
	}

	return archive, nil
}

func (g *Generator) GetArchiveInfo(ctx context.Context, jobID string) ([]*generator.FileErrors, int, error) {
	genErrs, genCnt, err := g.storage.GetArchiveInfo(ctx, jobID)
	if err != nil {
		return nil, 0, domain.CheckAppErr(err)
	}

	return genErrs, genCnt, nil
}

func (g *Generator) Close() error {
	return g.conn.Close()
}

func (g *Generator) grpcGenerate(ctx context.Context, request *generator.GenerateRequest, job domain.Job) {

	response, err := g.grpcClient.Generate(ctx, request)
	if err != nil {
		g.log.Debug("error during grpc request", logging.FieldError, err)
		if err = g.storage.SaveResponse(
			ctx,
			job.ID,
			nil,
			[]*generator.FileErrors{},
			0,
			domain.ErrInternal.Wrap("can't execute grpc request", err),
		); err != nil {
			g.log.Error("failed to save response", logging.FieldError, err)
		}
		if err = g.storage.UpdateJob(ctx, job.ID, domain.StatusFailed); err != nil {
			g.log.Error("failed to update job status", logging.FieldError, err)
		}
		return
	}
	g.log.Debug("gRPC response received",
		"zip_size", len(response.ZipArchive),
		"errors_count", len(response.Errors),
		"generated_count", response.GeneratedCount,
	)

	if err = g.storage.SaveResponse(
		ctx,
		job.ID,
		response.ZipArchive,
		response.Errors,
		int(response.GeneratedCount),
		nil,
	); err != nil {
		g.log.Error("failed to save response", logging.FieldError, err)
	}
	if err = g.storage.UpdateJob(ctx, job.ID, domain.StatusCompleted); err != nil {
		g.log.Error("failed to update job status", logging.FieldError, err)
	}
}
