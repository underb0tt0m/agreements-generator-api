package storage_in_memory

import (
	"context"
	"sync"
	"time"

	"agreements-generator/gen/go/generator"
	"agreements-generator/internal/config"
	"agreements-generator/internal/domain"
)

type JobData struct {
	Status     domain.JobStatus
	Archive    []byte
	Errors     []*generator.FileErrors
	GenCount   int
	CreatedAt  time.Time
	FatalError error
}

type UserData struct {
	Name         string
	Login        string
	PasswordHash []byte
	CreatedAt    time.Time
}

type MemoryStorage struct {
	mu    sync.RWMutex
	data  map[string]*JobData
	users map[string]*UserData
	cfg   *config.Config
}

func NewMemoryStorage(cfg *config.Config) *MemoryStorage {
	s := &MemoryStorage{
		data:  make(map[string]*JobData),
		users: make(map[string]*UserData),
		cfg:   cfg,
	}

	go s.cleanupLoop()
	return s
}

func (s *MemoryStorage) cleanupLoop() {
	ticker := time.NewTicker(s.cfg.Storage.JobTTL)
	for range ticker.C {
		s.cleanup()
	}
}

func (s *MemoryStorage) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for id, job := range s.data {
		if now.Sub(job.CreatedAt) > s.cfg.Storage.JobTTL {
			delete(s.data, id)
		}
	}
}

func (s *MemoryStorage) StoreJob(ctx context.Context, job domain.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.data[job.ID]; exists {
		return domain.ErrConflict.Wrap("job already exists", nil)
	}
	s.data[job.ID] = &JobData{
		Status:    job.Status,
		CreatedAt: time.Now(),
	}
	return nil
}

func (s *MemoryStorage) UpdateJob(ctx context.Context, id string, status domain.JobStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, exists := s.data[id]
	if !exists {
		return domain.ErrNotFound.Wrap("job not found", nil)
	}
	job.Status = status
	return nil
}

func (s *MemoryStorage) CheckJobStatus(ctx context.Context, id string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, exists := s.data[id]
	if !exists {
		return "", domain.ErrNotFound.Wrap("job not found", nil)
	}
	return string(job.Status), nil
}

func (s *MemoryStorage) SaveResponse(ctx context.Context, jobID string, archive []byte, errs []*generator.FileErrors, genCnt int, err error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, exists := s.data[jobID]
	if !exists {
		return domain.ErrNotFound.Wrap("job not found", nil)
	}
	if err != nil {
		job.Status = domain.StatusFailed
	} else {
		job.Status = domain.StatusCompleted
	}
	job.FatalError = err
	job.Archive = archive
	job.Errors = errs
	job.GenCount = genCnt
	return nil
}

func (s *MemoryStorage) GetArchive(ctx context.Context, jobID string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, exists := s.data[jobID]
	if !exists {
		return nil, domain.ErrNotFound.Wrap("job not found", nil)
	}
	if job.Status != domain.StatusCompleted {
		return nil, domain.ErrBadRequest.Wrap("job not completed", nil)
	}
	if job.Archive == nil {
		return nil, domain.ErrNotFound.Wrap("archive not found", nil)
	}
	return job.Archive, nil
}

func (s *MemoryStorage) GetArchiveInfo(ctx context.Context, jobID string) ([]*generator.FileErrors, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, exists := s.data[jobID]
	if !exists {
		return nil, 0, domain.ErrNotFound.Wrap("job not found", nil)
	}
	if job.Status != domain.StatusCompleted && job.Status != domain.StatusFailed {
		return nil, 0, domain.ErrBadRequest.Wrap("job not finished", nil)
	}
	return job.Errors, job.GenCount, nil
}

func (s *MemoryStorage) Register(ctx context.Context, user domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[user.Login]; exists {
		return domain.ErrConflict.Wrap("user already exists", nil)
	}

	s.users[user.Login] = &UserData{
		PasswordHash: user.Password,
		Name:         user.Name,
		CreatedAt:    time.Now(),
	}
	return nil
}

func (s *MemoryStorage) LogIn(ctx context.Context, login string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userData, exists := s.users[login]
	if !exists {
		return nil, domain.ErrNotFound.Wrap("user not found", nil)
	}
	return userData.PasswordHash, nil
}
