package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"agreements-generator/internal/domain"
	"agreements-generator/internal/encoder"
	logger_package "agreements-generator/internal/logger"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type StorageGenerator struct {
	conn    *pgx.Conn
	logger  logger_package.Logger
	encoder encoder.Encoder
	jobTTL  time.Duration
}

func New(conn *pgx.Conn, logger logger_package.Logger, encoder encoder.Encoder, jobTTL time.Duration) *StorageGenerator {
	s := &StorageGenerator{
		conn:    conn,
		logger:  logger,
		encoder: encoder,
		jobTTL:  jobTTL,
	}

	go s.cleanUpLoop(context.Background())

	return s
}

func (s *StorageGenerator) GetArchive(ctx context.Context, jobID string) (string, []byte, string, error) {
	stmt := `
SELECT j.status, a.archive, a.fatal_gen_error
FROM archives a
JOIN jobs j ON a.job_id=j.id
WHERE a.job_id=$1;
`
	row := s.conn.QueryRow(ctx, stmt, jobID)

	var (
		status      string
		archive     []byte
		fatalGenErr string
	)
	if err := row.Scan(&status, &archive, &fatalGenErr); err != nil {
		return "", nil, "", newDomainErrFromPgx(err)
	}

	return status, archive, fatalGenErr, nil
}

func (s *StorageGenerator) GetArchiveInfo(ctx context.Context, jobID string) (string, []domain.FilesErrors, int, string, error) {
	stmt := `
SELECT j.status, a.gen_errors, a.gen_count, a.fatal_gen_error
FROM archives a
JOIN jobs j ON a.job_id=j.id
WHERE a.job_id=$1;
`
	row := s.conn.QueryRow(ctx, stmt, jobID)

	var (
		status       string
		genErrsBytes []byte
		genCnt       int
		fatalGenErr  string
	)
	if err := row.Scan(&status, &genErrsBytes, &genCnt, &fatalGenErr); err != nil {
		return "", nil, 0, "", newDomainErrFromPgx(err)
	}

	var genErrs []domain.FilesErrors
	if genErrsBytes != nil {
		if err := s.encoder.Unmarshal(genErrsBytes, &genErrs); err != nil {
			s.logger.Debug(fmt.Sprintf("can't marshal bytes into errors: %v", genErrsBytes))
			return "", nil, 0, "", fmt.Errorf("can't marshal bytes into errors: %v, %w", err, domain.ErrInternal)
		}
	}

	return status, genErrs, genCnt, fatalGenErr, nil
}

func (s *StorageGenerator) SaveResponse(
	ctx context.Context,
	job domain.Job,
	response *domain.GenResponse,
	fatalGenErr error) error {

	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return newDomainErrFromPgx(fmt.Errorf("can't begin transaction: %w", err))
	}
	defer tx.Rollback(ctx)

	stmt := `
INSERT INTO archives(job_id, archive, gen_count, gen_errors, fatal_gen_error) 
VALUES ($1, $2, $3, $4, $5);
`

	fatalGenErrString := ""
	if fatalGenErr != nil {
		fatalGenErrString = fatalGenErr.Error()
	}

	if _, err = tx.Exec(
		ctx,
		stmt,
		job.ID,
		response.Archive,
		response.GenCount,
		response.Errors,
		fatalGenErrString,
	); err != nil {
		return newDomainErrFromPgx(err)
	}

	tx.Commit(ctx)

	return nil
}

func (s *StorageGenerator) StoreJob(ctx context.Context, job domain.Job, userID int) error {
	stmt := `
INSERT INTO jobs (id, status, user_id)
VALUES ($1, $2, $3); 
`
	if _, err := s.conn.Exec(ctx, stmt, job.ID, job.Status, userID); err != nil {
		return newDomainErrFromPgx(err)
	}

	return nil
}

func (s *StorageGenerator) UpdateJob(ctx context.Context, job domain.Job) error {
	stmt := `
UPDATE jobs
SET status = $1, updated_at = $2
WHERE id = $3;
`
	if _, err := s.conn.Exec(ctx, stmt, job.Status, time.Now(), job.ID); err != nil {
		return newDomainErrFromPgx(err)
	}

	return nil
}

func (s *StorageGenerator) CheckJobStatus(ctx context.Context, id string) (string, error) {
	stmt := `
SELECT status
FROM jobs
WHERE id = $1; 
`
	var status string
	if err := s.conn.QueryRow(ctx, stmt, id).Scan(&status); err != nil {
		return "", newDomainErrFromPgx(err)
	}

	return status, nil
}

func (s *StorageGenerator) Register(ctx context.Context, user domain.User) (int, error) {
	stmt := `
INSERT INTO users (login, name, password)
VALUES ($1, $2, $3)
RETURNING id;
`

	row := s.conn.QueryRow(ctx, stmt, user.Login, user.Name, user.Password)

	var userID int
	if err := row.Scan(&userID); err != nil {
		return 0, newDomainErrFromPgx(err)
	}

	return userID, nil
}

func (s *StorageGenerator) LogIn(ctx context.Context, login string) (int, []byte, error) {
	stmt := `
SELECT id, password
FROM users
WHERE login = $1;
`
	row := s.conn.QueryRow(ctx, stmt, &login)

	var userID int
	var password []byte
	if err := row.Scan(&userID, &password); err != nil {
		return 0, nil, newDomainErrFromPgx(err)
	}

	return userID, password, nil
}

func newDomainErrFromPgx(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf(
			"resource not found in storage: %v, %w", err, domain.ErrNotFound,
		)
	}

	pgxErr, ok := errors.AsType[*pgconn.PgError](err)
	if !ok {
		return fmt.Errorf("unknown error from postgres: %v, %w", err, domain.ErrInternal)
	}
	switch pgxErr.Code {
	case "23505":
		return fmt.Errorf("duplicate key value violates unique constraint: %v, %w", err, domain.ErrConflict)
	case "23502":
		return fmt.Errorf("null value in column violates not-null constraint: %v, %w", err, domain.ErrBadRequest)
	default:
		return fmt.Errorf("unknown error from postgres: %v, %w", err, domain.ErrBadRequest)
	}
}

func (s *StorageGenerator) cleanUpLoop(ctx context.Context) {
	stmt := `
DELETE FROM jobs 
WHERE (NOW() - created_at) > $1;
`
	ticker := time.NewTicker(s.jobTTL)

	for range ticker.C {
		tx, err := s.conn.Begin(ctx)
		if err != nil {
			s.logger.Error("can't create cleanup transaction", logger_package.FieldError, err)
			continue
		}
		defer tx.Rollback(ctx)

		cmdTag, err := tx.Exec(ctx, stmt, s.jobTTL)
		if err != nil {
			s.logger.Error("can't delete old jobs", logger_package.FieldError, err)
			continue
		}

		s.logger.Debug(fmt.Sprintf("deleted jobs: %v", cmdTag.RowsAffected()))
		tx.Commit(ctx)
	}
}
