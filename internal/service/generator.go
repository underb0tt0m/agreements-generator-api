package service

import (
	"context"

	"agreements-generator/internal/config"
	"agreements-generator/internal/domain"
	"agreements-generator/internal/gen_client"
	"agreements-generator/internal/logging"
	"agreements-generator/internal/storage"

	"github.com/google/uuid"
)

type Generator struct {
	client  gen_client.GeneratorClient
	log     logging.Logger
	storage storage.GeneratorStorage
	cfg     *config.Config
}

func NewGen(cfg *config.Config, l logging.Logger, s storage.GeneratorStorage, client gen_client.GeneratorClient) (*Generator, error) {
	return &Generator{
		client:  client,
		log:     l,
		storage: s,
		cfg:     cfg,
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

	jobCtx, cancel := context.WithTimeout(context.Background(), g.cfg.GRPCClient.JobMaxDuration)
	responseChan := make(chan *domain.GenResponse)
	errChan := make(chan error)
	go func() {
		defer cancel()
		defer close(errChan)
		defer close(responseChan)

		g.log.Debug("connecting to gRPC",
			"host", g.cfg.GRPCClient.Host,
			"port", g.cfg.GRPCClient.Port,
		)

		go g.client.BulkGenerate(jobCtx, archiveBytes, responseChan, errChan)

		g.log.Debug("waiting client's response")

		err = <-errChan
		response := <-responseChan

		g.log.Debug("response has been received")

		if err != nil {
			if storageErr := g.storage.SaveResponse(jobCtx, job.ID, response, err); storageErr != nil {
				g.log.Error("can't update job info", logging.FieldError, storageErr)
			}

			if storageErr := g.storage.UpdateJob(jobCtx, job.ID, domain.StatusFailed); storageErr != nil {
				g.log.Error("job failed; can't update job status", logging.FieldError, storageErr)
			}

			return
		}

		if storageErr := g.storage.SaveResponse(jobCtx, job.ID, response, err); storageErr != nil {
			g.log.Error("can't update job info", logging.FieldError, storageErr)

			if storageErr = g.storage.UpdateJob(jobCtx, job.ID, domain.StatusFailed); storageErr != nil {
				g.log.Error("can't update job status", logging.FieldError, storageErr)
			}

			return
		}

		if storageErr := g.storage.UpdateJob(jobCtx, job.ID, domain.StatusCompleted); storageErr != nil {
			g.log.Error("can't update job status", logging.FieldError, storageErr)
		}

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

func (g *Generator) GetArchiveInfo(ctx context.Context, jobID string) ([]domain.FilesErrors, int, error) {
	genErrs, genCnt, err := g.storage.GetArchiveInfo(ctx, jobID)
	if err != nil {
		return nil, 0, domain.CheckAppErr(err)
	}

	return genErrs, genCnt, nil
}
