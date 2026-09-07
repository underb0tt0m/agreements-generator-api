package grpc_generator

import (
	"context"
	"fmt"
	"time"

	"agreements-generator/internal/cache"
	"agreements-generator/internal/domain"
	"agreements-generator/internal/gen_client"
	"agreements-generator/internal/logger"
	appmetrics "agreements-generator/internal/metrics"
	"agreements-generator/internal/storage"

	"github.com/google/uuid"
)

type generator struct {
	log            logger.Logger
	storage        storage.GeneratorStorage
	grpcClient     gen_client.GeneratorClient
	cacher         cache.Cacher
	jobMaxDuration time.Duration
}

func NewGRPCGen(
	l logger.Logger,
	s storage.GeneratorStorage,
	client gen_client.GeneratorClient,
	cacher cache.Cacher,
	jobMaxDuration time.Duration,
) (*generator, error) {
	return &generator{
		grpcClient:     client,
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
			WithLabelValues("grpc", result).
			Inc()

		appmetrics.JobSubmissionDuration.
			WithLabelValues("grpc").
			Observe(time.Since(start).Seconds())
	}()

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

	g.log.Debug(fmt.Sprintf(
		"set job context variables. login: %v, userID: %v",
		ctx.Value(domain.LoginKey),
		ctx.Value(domain.UserIDKey)),
	)
	jobCtx = context.WithValue(jobCtx, domain.LoginKey, ctx.Value(domain.LoginKey))
	jobCtx = context.WithValue(jobCtx, domain.UserIDKey, ctx.Value(domain.UserIDKey))

	responseChan := make(chan *domain.GenResponse)
	errChan := make(chan error)
	go g.ProcessJob(job, archiveBytes, jobCtx, cancel, errChan, responseChan)

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

func (g *generator) ProcessJob(
	job domain.Job,
	archiveBytes []byte,
	jobCtx context.Context,
	ctxCancel context.CancelFunc,
	errChan chan error,
	responseChan chan *domain.GenResponse) {

	start := time.Now()
	result := "success"

	appmetrics.JobsInProgress.WithLabelValues("grpc").Inc()

	defer func() {
		appmetrics.JobsInProgress.WithLabelValues("grpc").Dec()

		appmetrics.JobProcessingDuration.
			WithLabelValues("grpc", result).
			Observe(time.Since(start).Seconds())
	}()

	defer ctxCancel()
	defer close(errChan)
	defer close(responseChan)

	g.log.Debug("connecting to gRPC")

	go g.grpcClient.BulkGenerate(jobCtx, archiveBytes, responseChan, errChan)

	g.log.Debug("waiting client's response")

	err := <-errChan
	response := <-responseChan

	finalizeCtx, finalizeCancel := context.WithTimeout(
		context.WithoutCancel(jobCtx),
		5*time.Second,
	)
	defer finalizeCancel()

	g.log.Debug("response has been received")

	failedJob := domain.Job{ID: job.ID, Status: domain.StatusFailed}
	completedJob := domain.Job{ID: job.ID, Status: domain.StatusCompleted}

	if err != nil {
		result = "error"

		if storageErr := g.storage.SaveResponse(finalizeCtx, failedJob, response, err); storageErr != nil {
			g.log.Error("can't update job info", logger.FieldError, storageErr)
		}

		if storageErr := g.storage.UpdateJob(finalizeCtx, failedJob); storageErr != nil {
			g.log.Error("job failed; can't update job status", logger.FieldError, storageErr)
		}

		if cacherErr := g.cacher.SetJobStatus(finalizeCtx, failedJob.ID, string(failedJob.Status)); cacherErr != nil {
			g.log.Error("can't update job info in cache", logger.FieldError, cacherErr)
		}

		return
	}

	if storageErr := g.storage.SaveResponse(finalizeCtx, completedJob, response, nil); storageErr != nil {
		result = "error"

		g.log.Error("can't update job info", logger.FieldError, storageErr)

		if storageErr = g.storage.UpdateJob(finalizeCtx, failedJob); storageErr != nil {
			g.log.Error("can't update job status", logger.FieldError, storageErr)
		}

		if cacherErr := g.cacher.SetJobStatus(finalizeCtx, failedJob.ID, string(failedJob.Status)); cacherErr != nil {
			g.log.Error("can't update job info in cache", logger.FieldError, cacherErr)
		}

		return
	}

	if storageErr := g.storage.UpdateJob(finalizeCtx, completedJob); storageErr != nil {
		result = "error"

		g.log.Error("can't update job status", logger.FieldError, storageErr)

		if cacherErr := g.cacher.SetJobStatus(
			finalizeCtx,
			failedJob.ID,
			string(failedJob.Status),
		); cacherErr != nil {
			g.log.Error(
				"can't update job info in cache",
				logger.FieldError,
				cacherErr,
			)
		}

		return
	}

	if cacherErr := g.cacher.SetJobStatus(
		finalizeCtx,
		completedJob.ID,
		string(completedJob.Status),
	); cacherErr != nil {
		g.log.Error(
			"can't update job info in cache",
			logger.FieldError,
			cacherErr,
		)
	}

}
