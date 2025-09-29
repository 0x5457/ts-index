package parser

import "github.com/0x5457/ts-index/internal/models"

type Parser interface {
	ParseFile(path string) ([]models.Symbol, []models.CodeChunk, error)
	ParseFileWithRoot(root, path string) ([]models.Symbol, []models.CodeChunk, error)
	ParseProject(root string) ([]models.Symbol, []models.CodeChunk, error)
}
