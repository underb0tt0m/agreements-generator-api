package storage_in_memory

import (
	"context"
	"fmt"
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
	ID           int
	Name         string
	Login        string
	PasswordHash []byte
	CreatedAt    time.Time
}

type MemoryStorage struct {
	mu              sync.RWMutex
	data            map[string]*JobData
	users           map[string]*UserData
	userIDIncrement int
	cfg             *config.Config
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

func (s *MemoryStorage) StoreJob(_ context.Context, job domain.Job, _ int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.data[job.ID]; exists {
		return fmt.Errorf("job already exists: %w", domain.ErrConflict)
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
		return fmt.Errorf("job not found: %w", domain.ErrNotFound)
	}
	jobData.Status = job.Status
	return nil
}

func (s *MemoryStorage) CheckJobStatus(_ context.Context, id string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, exists := s.data[id]
	if !exists {
		return "", fmt.Errorf("job not found: %w", domain.ErrNotFound)
	}
	return string(job.Status), nil
}

func (s *MemoryStorage) SaveResponse(_ context.Context, job domain.Job, response *domain.GenResponse, err error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	jobData, exists := s.data[job.ID]
	if !exists {
		return fmt.Errorf("job not found: %w", domain.ErrNotFound)
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

func (s *MemoryStorage) GetArchive(_ context.Context, jobID string) (string, []byte, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, exists := s.data[jobID]
	if !exists {
		return "", nil, "", fmt.Errorf("job not found: %w", domain.ErrNotFound)
	}

	var fatalErrString string
	if job.FatalError != nil {
		fatalErrString = job.FatalError.Error()
	}

	return string(job.Status), job.Archive, fatalErrString, nil
}

func (s *MemoryStorage) GetArchiveInfo(_ context.Context, jobID string) (string, []domain.FilesErrors, int, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, exists := s.data[jobID]
	if !exists {
		return "", nil, 0, "", fmt.Errorf("job not found: %w", domain.ErrNotFound)
	}

	var fatalErrString string
	if job.FatalError != nil {
		fatalErrString = job.FatalError.Error()
	}

	return string(job.Status), job.Errors, job.GenCount, fatalErrString, nil
}

func (s *MemoryStorage) Register(_ context.Context, user domain.User) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[user.Login]; exists {
		return 0, fmt.Errorf("user already exists: %w", domain.ErrConflict)
	}

	id := s.userIDIncrement
	s.userIDIncrement++

	s.users[user.Login] = &UserData{
		ID:           id,
		PasswordHash: user.Password,
		Name:         user.Name,
		CreatedAt:    time.Now(),
	}

	return id, nil
}

func (s *MemoryStorage) LogIn(_ context.Context, login string) (int, []byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userData, exists := s.users[login]
	if !exists {
		return 0, nil, fmt.Errorf("user not found: %w", domain.ErrNotFound)
	}
	return userData.ID, userData.PasswordHash, nil
}
