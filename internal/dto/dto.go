package dto

import "agreements-generator/gen/go/generator"

type BulkGenerateResponse struct {
	Archive    []byte        `json:"archive"`
	ErrorsList []FilesErrors `json:"errors_list"`
	GenCount   int           `json:"gen_count"`
}

type FilesErrors struct {
	FileName string   `json:"file_name"`
	Errors   []string `json:"errors"`
}

func NewBulkGenerateResponse(archive []byte, grpcErrs []*generator.FileErrors, genCnt int) *BulkGenerateResponse {
	httpErrs := make([]FilesErrors, 0, len(grpcErrs))
	for _, fileErrs := range grpcErrs {
		errs := make([]string, 0, len(fileErrs.Errors))
		for _, err := range fileErrs.Errors {
			errs = append(errs, err.Message)
		}
		httpErrs = append(httpErrs, FilesErrors{
			FileName: fileErrs.FileName,
			Errors:   errs,
		})
	}
	return &BulkGenerateResponse{
		Archive:    archive,
		ErrorsList: httpErrs,
		GenCount:   genCnt,
	}
}
