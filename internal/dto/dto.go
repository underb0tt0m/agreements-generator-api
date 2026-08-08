package dto

import (
	"agreements-generator/internal/domain"
)

type BulkGenerateResponse struct {
	JobID string `json:"job_id"`
}

type GetJobStatusResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

type GetArchiveInfoResponse struct {
	GenErrs []FilesErrors `json:"generation_errors"`
	GenCnt  int           `json:"generated_count"`
}

type FilesErrors struct {
	FileName string   `json:"file_name"`
	Errors   []string `json:"errors"`
}

func NewGetArchiveInfoResponse(grpcErrs []domain.FilesErrors, genCnt int) *GetArchiveInfoResponse {
	httpErrs := make([]FilesErrors, 0, len(grpcErrs))
	for _, fileErrs := range grpcErrs {
		errs := make([]string, 0, len(fileErrs.Errors))
		for _, err := range fileErrs.Errors {
			errs = append(errs, err.Msg)
		}
		httpErrs = append(httpErrs, FilesErrors{
			FileName: fileErrs.Name,
			Errors:   errs,
		})
	}
	return &GetArchiveInfoResponse{
		GenErrs: httpErrs,
		GenCnt:  genCnt,
	}
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Login    string `json:"login"`
	Password string `json:"password"`
}

type LogInRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
}
