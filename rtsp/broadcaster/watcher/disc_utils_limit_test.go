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
