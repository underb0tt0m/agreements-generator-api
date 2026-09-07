package queue_generator

import (
	"context"
	"fmt"
	"time"

	"agreements-generator/internal/cache"
	"agreements-generator/internal/domain"
	"agreements-generator/internal/logger"
	appmetrics "agreements-generator/internal/metrics"
	"agreements-generator/internal/publisher"
	"agreements-generator/internal/storage"

	"github.com/google/uuid"
)

type generator struct {
	log            logger.Logger
	storage        storage.GeneratorStorage
	publisher      publisher.Publisher
	cacher         cache.Cacher
	jobMaxDuration time.Duration
}

func NewQueueGen(
	l logger.Logger,
	s storage.GeneratorStorage,
	publer publisher.Publisher,
	cacher cache.Cacher,
	jobMaxDuration time.Duration,
) (*generator, error) { // TODO подумать над тем, как исключить иначе циклические импорты
	return &generator{
		publisher:      publer,
		cacher:         cacher,
		log:            l,
		storage:        s,
		jobMaxDuration: jobMaxDuration,
	}, nil
}

func (g *generator) BulkGenerate(ctx context.Context, archiveBytes []byte) (jobID string, err error) {
	start := time.Now()

	defer func() {
		result := "success"
		if err != nil {
			result = "error"
		}

		appmetrics.JobsSubmittedTotal.
			WithLabelValues("queue", result).
			Inc()

		appmetrics.JobSubmissionDuration.
			WithLabelValues("queue").
			Observe(time.Since(start).Seconds())
	}()

	id, err := uuid.NewUUID()
	if err != nil {
		return "", fmt.Errorf("can't create job: %w", err)
	}

	job := domain.Job{
		ID:     id.String(),
		Status: domain.StatusProcessing,
	}

	userID, ok := ctx.Value(domain.UserIDKey).(int)
	if !ok {
		g.log.Error(fmt.Sprintf("userID isn't integer: %v", ctx.Value(domain.UserIDKey)))
	}

	if err = g.storage.StoreJob(ctx, job, userID); err != nil {
		return "", fmt.Errorf("can't add job with ID %s in storage: %w", job.ID, err)
	}

	if err = g.cacher.SetJobStatus(ctx, job.ID, string(job.Status)); err != nil {
		g.log.Error("failed to set status in cache", logger.FieldError, err)
	}

	jobCtx, cancel := context.WithTimeout(context.Background(), g.jobMaxDuration)
	defer cancel()

	g.log.Debug(fmt.Sprintf(
		"set job context variables. login: %v, userID: %v",
		ctx.Value(domain.LoginKey),
		ctx.Value(domain.UserIDKey)),
	)
	jobCtx = context.WithValue(jobCtx, domain.LoginKey, ctx.Value(domain.LoginKey))
	jobCtx = context.WithValue(jobCtx, domain.UserIDKey, ctx.Value(domain.UserIDKey))

	if storer, ok := g.storage.(storage.InputArchiveStorer); ok {
		if err = storer.SaveRawArchive(ctx, job.ID, archiveBytes); err != nil {
			return "", fmt.Errorf("can't store raw archive in storage: %v: %w", err, domain.ErrStorageBadRequest)
		}
	} else {
		return "", fmt.Errorf(
			"can't store raw archive in storage; storage doesn't implement InputArchiveStorer: %w",
			domain.ErrStorageBadRequest,
		)
	}

	if err = g.publishJob(jobCtx, job); err != nil {
		return "", fmt.Errorf("can't add job with ID %s to queue: %w", job.ID, err)
	}

	return job.ID, nil
}

func (g *generator) CheckJobStatus(ctx context.Context, id string) (domain.JobStatus, error) {
	var (
		dbErr     error
		cacherErr error
		status    string
	)

	status, cacherErr = g.cacher.GetJobStatus(ctx, id)

	if cacherErr != nil {
		g.log.Warn("can't get job status from cache, trying to get from db", logger.FieldError, cacherErr)

		status, dbErr = g.storage.CheckJobStatus(ctx, id)
		if dbErr != nil {
			return domain.StatusFailed, fmt.Errorf("can't check job status: %w", dbErr)
		}
	}

	jobStatus, statusErr := domain.JobStatusFromString(status)
	if statusErr != nil {
		return domain.StatusFailed, fmt.Errorf("can't convert job status: %w", statusErr)
	}

	if cacherErr != nil {
		if cacherErr = g.cacher.SetJobStatus(ctx, id, string(jobStatus)); cacherErr != nil {
			g.log.Error("can't update job status in cache", logger.FieldError, cacherErr)
		}
	}

	return jobStatus, nil
}

func (g *generator) GetArchive(ctx context.Context, jobID string) ([]byte, error) {
	status, archive, fatalGenErr, err := g.storage.GetArchive(ctx, jobID)

	if err != nil {
		return nil, fmt.Errorf("can't get archive from store: %w", err)
	}

	jobStatus, statusErr := domain.JobStatusFromString(status)
	if statusErr != nil {
		return nil, fmt.Errorf("can't convert job status: %w", statusErr)
	}

	if jobStatus != domain.StatusCompleted {
		if fatalGenErr != "" {
			return nil, fmt.Errorf("can't get archive: fatal generation error: %s: %w", fatalGenErr, domain.ErrInternal)
		}
		return nil, domain.ErrJobNotFinished
	}

	if archive == nil {
		return nil, domain.ErrNotFound
	}

	return archive, nil
}

func (g *generator) GetArchiveInfo(ctx context.Context, jobID string) ([]domain.FilesErrors, int, error) {
	status, genErrs, genCnt, fatalGenErr, err := g.storage.GetArchiveInfo(ctx, jobID)

	if err != nil {
		return nil, 0, fmt.Errorf("can't get archive info from store: %w", err)
	}

	jobStatus, statusErr := domain.JobStatusFromString(status)
	if statusErr != nil {
		return nil, 0, fmt.Errorf("can't convert job status: %w", statusErr)
	}

	if jobStatus != domain.StatusCompleted {
		if fatalGenErr != "" {
			return nil, 0, fmt.Errorf("can't get archive info: fatal generation error: %s: %w", fatalGenErr, domain.ErrInternal)
		}
		return nil, 0, fmt.Errorf("can't get archive info: %w", domain.ErrJobNotFinished)
	}

	return genErrs, genCnt, nil
}

func (g *generator) publishJob(ctx context.Context, job domain.Job) error {
	return g.publisher.Send(ctx, job.ID)
}
