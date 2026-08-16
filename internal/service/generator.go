package service

import (
	"context"
	"fmt"
	"time"

	"agreements-generator/internal/domain"
	"agreements-generator/internal/gen_client"
	"agreements-generator/internal/logger"
	"agreements-generator/internal/storage"

	"github.com/google/uuid"
)

//go:generate mockgen -source=generator.go -destination=../mocks/generator.go -package=mocks -mock_names=Generator=MockGeneratorService
type Generator interface {
	BulkGenerate(ctx context.Context, archiveBytes []byte) (string, error)
	CheckJobStatus(ctx context.Context, id string) (domain.JobStatus, error)
	GetArchive(ctx context.Context, jobID string) ([]byte, error)
	GetArchiveInfo(ctx context.Context, jobID string) ([]domain.FilesErrors, int, error)
}
type generator struct {
	log            logger.Logger
	storage        storage.GeneratorStorage
	client         gen_client.GeneratorClient
	clientHost     string
	clientPort     string
	jobMaxDuration time.Duration
}

func NewGen(
	l logger.Logger,
	s storage.GeneratorStorage,
	client gen_client.GeneratorClient,
	clientHost string,
	clientPort string,
	jobMaxDuration time.Duration,
) (Generator, error) {
	return &generator{
		client:         client,
		log:            l,
		storage:        s,
		clientHost:     clientHost,
		clientPort:     clientPort,
		jobMaxDuration: jobMaxDuration,
	}, nil
}

func (g *generator) BulkGenerate(ctx context.Context, archiveBytes []byte) (string, error) {
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

	jobCtx, cancel := context.WithTimeout(context.Background(), g.jobMaxDuration)
	responseChan := make(chan *domain.GenResponse)
	errChan := make(chan error)
	go g.processJob(job, archiveBytes, jobCtx, cancel, errChan, responseChan)

	return job.ID, nil
}

func (g *generator) CheckJobStatus(ctx context.Context, id string) (domain.JobStatus, error) {
	status, err := g.storage.CheckJobStatus(ctx, id)

	if err != nil {
		return domain.StatusFailed, fmt.Errorf("can't check job status: %w", err)
	}

	jobStatus, statusErr := domain.JobStatusFromString(status)
	if statusErr != nil {
		return domain.StatusFailed, fmt.Errorf("can't convert job status: %w", statusErr)
	}

	return jobStatus, nil
}

func (g *generator) GetArchive(ctx context.Context, jobID string) ([]byte, error) {
	status, archive, err := g.storage.GetArchive(ctx, jobID)

	if err != nil {
		return nil, fmt.Errorf("can't get archive from store: %w", err)
	}

	jobStatus, statusErr := domain.JobStatusFromString(status)
	if statusErr != nil {
		return nil, fmt.Errorf("can't convert job status: %w", statusErr)
	}

	if jobStatus != domain.StatusCompleted {
		return nil, domain.ErrJobNotFinished
	}

	if archive == nil {
		return nil, domain.ErrNotFound
	}

	return archive, nil
}

func (g *generator) GetArchiveInfo(ctx context.Context, jobID string) ([]domain.FilesErrors, int, error) {
	status, genErrs, genCnt, err := g.storage.GetArchiveInfo(ctx, jobID)

	if err != nil {
		return nil, 0, fmt.Errorf("can't get archive info from store: %w", err)
	}

	jobStatus, statusErr := domain.JobStatusFromString(status)
	if statusErr != nil {
		return nil, 0, fmt.Errorf("can't convert job status: %w", statusErr)
	}

	if jobStatus != domain.StatusCompleted {
		return nil, 0, fmt.Errorf("can't get archive info: %w", domain.ErrJobNotFinished)
	}

	return genErrs, genCnt, nil
}

func (g *generator) processJob(
	job domain.Job,
	archiveBytes []byte,
	jobCtx context.Context,
	ctxCancel context.CancelFunc,
	errChan chan error,
	responseChan chan *domain.GenResponse) {

	defer ctxCancel()
	defer close(errChan)
	defer close(responseChan)

	g.log.Debug("connecting to gRPC",
		"host", g.clientHost,
		"port", g.clientPort,
	)

	go g.client.BulkGenerate(jobCtx, archiveBytes, responseChan, errChan)

	g.log.Debug("waiting client's response")

	err := <-errChan
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

}
