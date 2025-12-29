package wal

import (
	"testing"
)

func TestGzipCompressor_CompressDecompress(t *testing.T) {
	compressor := NewCompressor("gzip")
	if compressor == nil {
		t.Fatal("Failed to create gzip compressor")
	}

	originalData := []byte("test data for compression")

	compressed, err := compressor.Compress(originalData)
	if err != nil {
		t.Fatalf("Failed to compress: %v", err)
	}

	if len(compressed) == 0 {
		t.Fatal("Compressed data is empty")
	}

	decompressed, err := compressor.Decompress(compressed)
	if err != nil {
		t.Fatalf("Failed to decompress: %v", err)
	}

	if string(decompressed) != string(originalData) {
		t.Errorf("Decompressed data doesn't match original. Expected: %s, Got: %s",
			string(originalData), string(decompressed))
	}
}

func TestGzipCompressor_LargeData(t *testing.T) {
	compressor := NewCompressor("gzip")
	if compressor == nil {
		t.Fatal("Failed to create gzip compressor")
	}

	// Create large data volume
	originalData := make([]byte, 10000)
	for i := range originalData {
		originalData[i] = byte(i % 256)
	}

	compressed, err := compressor.Compress(originalData)
	if err != nil {
		t.Fatalf("Failed to compress: %v", err)
	}

	decompressed, err := compressor.Decompress(compressed)
	if err != nil {
		t.Fatalf("Failed to decompress: %v", err)
	}

	if len(decompressed) != len(originalData) {
		t.Errorf("Decompressed size doesn't match. Expected: %d, Got: %d",
			len(originalData), len(decompressed))
	}

	for i := range originalData {
		if decompressed[i] != originalData[i] {
			t.Errorf("Data mismatch at index %d", i)
			break
		}
	}
}

func TestNewCompressor_InvalidType(t *testing.T) {
	compressor := NewCompressor("invalid")
	if compressor != nil {
		t.Error("Expected nil compressor for invalid type")
	}
}

func TestNewCompressor_EmptyType(t *testing.T) {
	compressor := NewCompressor("")
	if compressor != nil {
		t.Error("Expected nil compressor for empty type")
	}
}

func TestNewCompressor_GZ(t *testing.T) {
	compressor := NewCompressor("gz")
	if compressor == nil {
		t.Error("Expected compressor for 'gz' type")
	}
}

func TestGzipCompressor_Decompress_InvalidData(t *testing.T) {
	compressor := NewCompressor("gzip")
	if compressor == nil {
		t.Fatal("Failed to create gzip compressor")
	}

	invalidData := []byte("not valid gzip data")
	_, err := compressor.Decompress(invalidData)
	if err == nil {
		t.Error("Expected error for invalid gzip data")
	}
}

func TestGzipCompressor_Decompress_EmptyData(t *testing.T) {
	compressor := NewCompressor("gzip")
	if compressor == nil {
		t.Fatal("Failed to create gzip compressor")
	}

	emptyData := []byte{}
	_, err := compressor.Decompress(emptyData)
	if err == nil {
		t.Error("Expected error for empty gzip data")
	}
}
