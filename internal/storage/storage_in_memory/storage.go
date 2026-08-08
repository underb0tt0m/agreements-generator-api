package storage_in_memory

import (
	"context"
	"sync"
	"time"

	"agreements-generator/internal/config"
	"agreements-generator/internal/domain"
)

type JobData struct {
	Status     domain.JobStatus
	Archive    []byte
	Errors     []domain.FilesErrors
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
	for id, job := range s.data {
		if time.Since(job.CreatedAt) > s.cfg.Storage.JobTTL {
			delete(s.data, id)
		}
	}
}

func (s *MemoryStorage) StoreJob(_ context.Context, job domain.Job) error {
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

func (s *MemoryStorage) UpdateJob(_ context.Context, job domain.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	jobData, exists := s.data[job.ID]
	if !exists {
		return domain.ErrNotFound.Wrap("job not found", nil)
	}
	jobData.Status = job.Status
	return nil
}

func (s *MemoryStorage) CheckJobStatus(_ context.Context, id string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, exists := s.data[id]
	if !exists {
		return "", domain.ErrNotFound.Wrap("job not found", nil)
	}
	return string(job.Status), nil
}

func (s *MemoryStorage) SaveResponse(_ context.Context, job domain.Job, response *domain.GenResponse, err error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	jobData, exists := s.data[job.ID]
	if !exists {
		return domain.ErrNotFound.Wrap("job not found", nil)
	}
	if err != nil {
		job.Status = domain.StatusFailed
	} else {
		job.Status = domain.StatusCompleted
	}
	jobData.FatalError = err
	jobData.Archive = response.Archive
	jobData.Errors = response.Errors
	jobData.GenCount = response.GenCount
	return nil
}

func (s *MemoryStorage) GetArchive(_ context.Context, jobID string) (string, []byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, exists := s.data[jobID]
	if !exists {
		return "", nil, domain.ErrNotFound.Wrap("job not found", nil)
	}

	return string(job.Status), job.Archive, nil
}

func (s *MemoryStorage) GetArchiveInfo(_ context.Context, jobID string) (string, []domain.FilesErrors, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, exists := s.data[jobID]
	if !exists {
		return "", nil, 0, domain.ErrNotFound.Wrap("job not found", nil)
	}

	return string(job.Status), job.Errors, job.GenCount, nil
}

func (s *MemoryStorage) Register(_ context.Context, user domain.User) error {
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

func (s *MemoryStorage) LogIn(_ context.Context, login string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userData, exists := s.users[login]
	if !exists {
		return nil, domain.ErrNotFound.Wrap("user not found", nil)
	}
	return userData.PasswordHash, nil
}
