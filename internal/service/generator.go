package service

import (
	"context"
	"fmt"

	"agreements-generator/internal/config"
	"agreements-generator/internal/domain"
	"agreements-generator/internal/gen_client"
	"agreements-generator/internal/logger"
	"agreements-generator/internal/storage"

	"github.com/google/uuid"
)

type Generator struct {
	client  gen_client.GeneratorClient
	log     logger.Logger
	storage storage.GeneratorStorage
	cfg     *config.Config
}

func NewGen(cfg *config.Config, l logger.Logger, s storage.GeneratorStorage, client gen_client.GeneratorClient) (*Generator, error) {
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
		return "", fmt.Errorf("can't create job: %w", err)
	}

	job := domain.Job{
		ID:     id.String(),
		Status: domain.StatusProcessing,
	}

	if err = g.storage.StoreJob(ctx, job); err != nil {
		return "", fmt.Errorf("can't add job with ID %s in storage: %w", job.ID, err)
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

		failedJob := domain.Job{ID: job.ID, Status: domain.StatusFailed}
		completedJob := domain.Job{ID: job.ID, Status: domain.StatusCompleted}

		if err != nil {
			if storageErr := g.storage.SaveResponse(jobCtx, failedJob, response, err); storageErr != nil {
				g.log.Error("can't update job info", logger.FieldError, storageErr)
			}

			if storageErr := g.storage.UpdateJob(jobCtx, failedJob); storageErr != nil {
				g.log.Error("job failed; can't update job status", logger.FieldError, storageErr)
			}

			return
		}

		if storageErr := g.storage.SaveResponse(jobCtx, completedJob, response, nil); storageErr != nil {
			g.log.Error("can't update job info", logger.FieldError, storageErr)

			if storageErr = g.storage.UpdateJob(jobCtx, failedJob); storageErr != nil {
				g.log.Error("can't update job status", logger.FieldError, storageErr)
			}

			return
		}

		if storageErr := g.storage.UpdateJob(jobCtx, completedJob); storageErr != nil {
			g.log.Error("can't update job status", logger.FieldError, storageErr)
		}

	}()

	return job.ID, nil
}

func (g *Generator) CheckJobStatus(ctx context.Context, id string) (domain.JobStatus, error) {
	status, err := g.storage.CheckJobStatus(ctx, id)
	if err != nil {
		return "", fmt.Errorf("can't check job status: %w", err)
	}

	return domain.JobStatusFromString(status)
}

func (g *Generator) GetArchive(ctx context.Context, jobID string) ([]byte, error) {
	status, archive, err := g.storage.GetArchive(ctx, jobID)

	jobStatus, statusErr := domain.JobStatusFromString(status)
	if statusErr != nil {
		return nil, fmt.Errorf("can't convert job status: %w", statusErr)
	}

	if jobStatus != domain.StatusCompleted {
		return nil, domain.ErrBadRequest.Wrap("job not completed", nil)
	}

	if archive == nil {
		return nil, domain.ErrNotFound.Wrap("archive not found", nil)
	}

	if err != nil {
		return nil, fmt.Errorf("can't get archive from store: %w", err)
	}

	return archive, nil
}

func (g *Generator) GetArchiveInfo(ctx context.Context, jobID string) ([]domain.FilesErrors, int, error) {
	status, genErrs, genCnt, err := g.storage.GetArchiveInfo(ctx, jobID)

	jobStatus, statusErr := domain.JobStatusFromString(status)
	if statusErr != nil {
		return nil, 0, fmt.Errorf("can't convert job status: %w", statusErr)
	}

	if jobStatus != domain.StatusCompleted {
		return nil, 0, domain.ErrBadRequest.Wrap("job not completed", nil)
	}

	if err != nil {
		return nil, 0, fmt.Errorf("can't get archive info from store: %w", err)
	}

	return genErrs, genCnt, nil
}
