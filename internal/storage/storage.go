package storage

import (
	"context"
	"fmt"

	"agreements-generator/internal/config"
	"agreements-generator/internal/domain"
	"agreements-generator/internal/encoder"
	loggerModule "agreements-generator/internal/logger"
	postgres_package "agreements-generator/internal/storage/postgres"
	"agreements-generator/internal/storage/storage_in_memory"

	"github.com/jackc/pgx/v5"
)

const (
	inMemory = "in_memory"
	postgres = "postgres"
)

//go:generate mockgen -source=storage.go -destination=../mocks/storage.go -package=mocks
type GeneratorStorage interface {
	GetArchive(ctx context.Context, jobID string) (string, []byte, string, error)
	GetArchiveInfo(ctx context.Context, jobID string) (string, []domain.FilesErrors, int, string, error)
	SaveResponse(ctx context.Context, job domain.Job, response *domain.GenResponse, err error) error
	StoreJob(ctx context.Context, job domain.Job, userID int) error
	UpdateJob(ctx context.Context, job domain.Job) error
	CheckJobStatus(ctx context.Context, id string) (string, error)
}

type UserStorage interface {
	Register(ctx context.Context, user domain.User) (int, error)
	LogIn(ctx context.Context, login string) (int, []byte, error)
}

func New(ctx context.Context, cfg *config.Config, logger loggerModule.Logger, encoder encoder.Encoder) (GeneratorStorage, UserStorage, error) {
	var userStorage UserStorage
	var generatorStorage GeneratorStorage

	switch cfg.Storage.Type {
	case inMemory:
		s := storage_in_memory.NewMemoryStorage(cfg)
		generatorStorage = s
		userStorage = s
	case postgres:
		conn, err := pgx.Connect(
			ctx,
			fmt.Sprintf(
				"%v://%v:%v@%v:%v/%v",
				cfg.Storage.Driver,
				cfg.Security.DBUser,
				cfg.Security.DBPassword,
				cfg.Storage.Host,
				cfg.Storage.Port,
				cfg.Storage.Database,
			),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("can't create storage: %w", err)
		}
		s := postgres_package.New(conn, logger, encoder, cfg.Storage.JobTTL)
		generatorStorage = s
		userStorage = s
	default:
		return nil, nil, fmt.Errorf("unvalid storage type in config")
	}

	return generatorStorage, userStorage, nil
}
