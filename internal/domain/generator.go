package domain

import "fmt"

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

// в го функции вида Is... обычно возвращают true/false, тут лучше JobStatusFromString(s string) (JobStatus, error)

func IsJobStatus(status string) (JobStatus, error) {
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
