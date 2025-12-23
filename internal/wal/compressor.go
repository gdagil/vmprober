package wal

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

// Compressor интерфейс для компрессии данных
type Compressor interface {
	Compress(data []byte) ([]byte, error)
	Decompress(data []byte) ([]byte, error)
}

// GzipCompressor реализация компрессора на основе gzip
type GzipCompressor struct{}

// NewCompressor создает новый компрессор
func NewCompressor(compressionType string) Compressor {
	switch compressionType {
	case "gzip", "gz":
		return &GzipCompressor{}
	default:
		return nil
	}
}

// Compress сжимает данные
func (g *GzipCompressor) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)

	if _, err := writer.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write data to gzip writer: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// Decompress распаковывает данные
func (g *GzipCompressor) Decompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return nil, fmt.Errorf("failed to decompress data: %w", err)
	}

	return buf.Bytes(), nil
}


