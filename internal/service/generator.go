package service

import (
	"context"
	"time"

	"agreements-generator/internal/cache"
	"agreements-generator/internal/domain"
	"agreements-generator/internal/gen_client"
	"agreements-generator/internal/logger"
	"agreements-generator/internal/publisher"
	"agreements-generator/internal/service/grpc_generator"
	"agreements-generator/internal/service/queue_generator"
	"agreements-generator/internal/storage"
)

//go:generate mockgen -source=generator.go -destination=../mocks/generator.go -package=mocks -mock_names=Generator=MockGeneratorService
type Generator interface {
	BulkGenerate(ctx context.Context, archiveBytes []byte) (string, error)
	CheckJobStatus(ctx context.Context, id string) (domain.JobStatus, error)
	GetArchive(ctx context.Context, jobID string) ([]byte, error)
	GetArchiveInfo(ctx context.Context, jobID string) ([]domain.FilesErrors, int, error)
}

func NewGen(
	execMode string,
	l logger.Logger,
	s storage.GeneratorStorage,
	client gen_client.GeneratorClient,
	publer publisher.Publisher,
	cacher cache.Cacher,
	jobMaxDuration time.Duration,
) (Generator, error) {
	switch execMode {
	case "grpc":
		return grpc_generator.NewGRPCGen(
			l,
			s,
			client,
			cacher,
			jobMaxDuration,
		)
	case "queue":
		return queue_generator.NewQueueGen(
			l,
			s,
			publer,
			cacher,
			jobMaxDuration,
		)
	default:
		return grpc_generator.NewGRPCGen(
			l,
			s,
			client,
			cacher,
			jobMaxDuration,
		)
	}
}
