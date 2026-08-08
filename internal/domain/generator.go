package domain

import (
	"fmt"
)

type Job struct {
	ID     string
	Status JobStatus
}

type JobStatus string

const (
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
)

func JobStatusFromString(status string) (JobStatus, error) {
	switch status {
	case string(StatusProcessing):
		return StatusProcessing, nil
	case string(StatusCompleted):
		return StatusCompleted, nil
	case string(StatusFailed):
		return StatusFailed, nil
	default:
		return "", ErrInternal.Wrap(fmt.Sprint("unknown job status: ", status), nil)
	}
}

type User struct {
	Name     string
	Login    string
	Password []byte
}

type ContextKey string

const LoginKey ContextKey = "login"

type GenResponse struct {
	Archive  []byte
	Errors   []FilesErrors
	GenCount int
}

type FilesErrors struct {
	Name   string
	Errors []FileError
}

type FileError struct {
	Code int
	Msg  string
}
