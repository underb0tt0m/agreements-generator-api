package views

import "agreements-generator/internal/domain"

type ArchiveInfo struct {
	Status      string
	Errors      []domain.FilesErrors
	Count       int
	FatalGenErr string
}
