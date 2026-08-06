package storage

import (
	"context"

	"agreements-generator/gen/go/generator"
	"agreements-generator/internal/domain"
)

type GeneratorStorage interface {
	GetArchive(ctx context.Context, jobID string) ([]byte, error)
	GetArchiveInfo(ctx context.Context, jobID string) ([]*generator.FileErrors, int, error)
	SaveResponse(ctx context.Context, jobID string, archive []byte, errs []*generator.FileErrors, genCnt int, err error) error
	StoreJob(ctx context.Context, job domain.Job) error
	UpdateJob(ctx context.Context, id string, status domain.JobStatus) error
	CheckJobStatus(ctx context.Context, id string) (string, error)
}

type UserStorage interface {
	Register(ctx context.Context, user domain.User) error
	LogIn(ctx context.Context, login string) ([]byte, error)
}
