package storage

import (
	"context"

	"agreements-generator/internal/domain"
)

//go:generate mockgen -source=storage.go -destination=../mocks/storage.go -package=mocks
type GeneratorStorage interface {
	GetArchive(ctx context.Context, jobID string) (string, []byte, error)
	GetArchiveInfo(ctx context.Context, jobID string) (string, []domain.FilesErrors, int, error)
	SaveResponse(ctx context.Context, job domain.Job, response *domain.GenResponse, err error) error
	StoreJob(ctx context.Context, job domain.Job) error
	UpdateJob(ctx context.Context, job domain.Job) error
	CheckJobStatus(ctx context.Context, id string) (string, error)
}

type UserStorage interface {
	Register(ctx context.Context, user domain.User) error
	LogIn(ctx context.Context, login string) ([]byte, error)
}
