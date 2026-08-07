package storage

import (
	"context"

	"agreements-generator/internal/domain"
)

type GeneratorStorage interface {
	GetArchive(ctx context.Context, jobID string) ([]byte, error)
	GetArchiveInfo(ctx context.Context, jobID string) ([]domain.FilesErrors, int, error)
	SaveResponse(ctx context.Context, jobID string, response *domain.GenResponse, err error) error
	StoreJob(ctx context.Context, job domain.Job) error
	UpdateJob(ctx context.Context, id string, status domain.JobStatus) error
	CheckJobStatus(ctx context.Context, id string) (string, error)
}

type UserStorage interface {
	Register(ctx context.Context, user domain.User) error
	LogIn(ctx context.Context, login string) ([]byte, error)
}
