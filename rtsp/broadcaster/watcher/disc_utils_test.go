package watcher

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsCloseToVideoSize(t *testing.T) {
	t.Run("Size is bigger than limit", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.jpg")
		if err := os.WriteFile(testFile, make([]byte, 100), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		actualSize, _ := FileSizeByExtension(tempDir, []string{".jpg"})
		convertedVideoSpace := int(actualSize * 2)
		saveChunkSize := 100
		os.WriteFile(filepath.Join(tempDir, "test2.jpg"), make([]byte, 100), 0644)
		result := IsCloseToVideoSize(tempDir, []string{".jpg"}, convertedVideoSpace, saveChunkSize)
		if !result {
			size, _ := FileSizeByExtension(tempDir, []string{".jpg"})
			limit := int64(convertedVideoSpace - (saveChunkSize * 2))
			t.Errorf("Expected true when size (%d) >= limit (%d), got false", size, limit)
		}
	})

	t.Run("Size is smaller than limit", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.jpg")
		if err := os.WriteFile(testFile, make([]byte, 100), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		actualSize, _ := FileSizeByExtension(tempDir, []string{".jpg"})
		convertedVideoSpace := int(actualSize * 10) // 10x the size
		saveChunkSize := 100

		result := IsCloseToVideoSize(tempDir, []string{".jpg"}, convertedVideoSpace, saveChunkSize)

		if result {
			size, _ := FileSizeByExtension(tempDir, []string{".jpg"})
			limit := int64(convertedVideoSpace - (saveChunkSize * 2))
			t.Errorf("Expected false when size (%d) < limit (%d), got true", size, limit)
		}
	})
}
func TestCountContentInDir(t *testing.T) {
	t.Run("count chunks in date dir", func(t *testing.T) {
		baseDir := t.TempDir()
		tempDir := filepath.Join(baseDir, "2025-01-01")
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			t.Fatal(err)
		}
		testFile := filepath.Join(tempDir, "test.jpg")
		if err := os.WriteFile(testFile, make([]byte, 1), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		testFile2 := filepath.Join(tempDir, "test2.jpg")
		if err := os.WriteFile(testFile2, make([]byte, 1), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		result := CountChunksInDateDir(baseDir, []string{})
		if result != 2 {
			t.Errorf("Expected 2 chunks, got %d", result)
		}
	})
}
func TestDirIndex(t *testing.T) {
	t.Run("increase index", func(t *testing.T) {
		baseDir := t.TempDir()
		tempDir := filepath.Join(baseDir, "2025-01-01")
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			t.Fatal(err)
		}
		testFile := filepath.Join(tempDir, "test.jpg")
		if err := os.WriteFile(testFile, make([]byte, 5), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		CreateNewDirIndex(tempDir)
		index, path, _ := TouchDirAndGetIndex(tempDir, 4)
		if index != 0 {
			t.Errorf("Expected index 2, got %d", index)
		}
		expected := filepath.Join(tempDir, "2")
		if path != expected {
			t.Errorf("Expected path %s, got %s", expected, path)
		}
	})
}
