package watcher

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

func SaveFrame(i int, b []byte, path string) {
	f, err := os.Create(fmt.Sprintf("%s/frame%d.jpg", path, i))
	if err != nil {
		panic(fmt.Sprintf("Cant create file: %v", err))
	}
	defer f.Close()
	f.Write(b)
}
func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			panic(fmt.Sprintf("Error accessing file %s: %v", info.Name(), err))
		}
		if !info.IsDir() {
			stat, ok := info.Sys().(*syscall.Stat_t)
			if ok {
				size += int64(stat.Blocks) * 512 // 512 is the block size used by du
			} else {
				size += info.Size() // fallback
			}
		}
		return err
	})
	return size, err
}
func FileSizeByExtension(path string, exensions []string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			panic(fmt.Sprintf("Error accessing file %s: %v", info.Name(), err))
		}
		if !info.IsDir() && slices.Contains(exensions, filepath.Ext(info.Name())) {
			stat, ok := info.Sys().(*syscall.Stat_t)
			if ok {
				size += int64(stat.Blocks) * 512 // 512 is the block size used by du
			} else {
				size += info.Size() // fallback
			}
		}
		return err
	})
	return size, err
}
func GetNewFileIndex(path string) (int, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return 0, err
	}
	maxNum := -1
	for _, file := range files {
		if file.Type().IsRegular() {
			var num int
			_, err := fmt.Sscanf(file.Name(), "frame%d.jpg", &num)
			if err == nil && num > maxNum {
				maxNum = num
			}
		}
	}

	return maxNum + 1, nil
}
func TouchDirAndGetIterator(base_path string, size_limit int64) (int, string) {
	dir_i := 1
	path := filepath.Join(base_path, fmt.Sprintf("%d", dir_i))
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(path, 0755); err != nil {
			panic(fmt.Sprintf("Cannot create directory: %v", err))
		}
		return 0, path
	}
	for {
		size, _ := DirSize(path)
		if size < size_limit {
			break
		}
		dir_i += 1
		path = filepath.Join(base_path, fmt.Sprintf("%d", dir_i))
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(path, 0755); err != nil {
				panic(fmt.Sprintf("Cannot create directory: %v", err))
			}
		}
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(path, 0755); err != nil {
			panic(fmt.Sprintf("Cannot create directory: %v", err))
		}
	}
	index, err := GetNewFileIndex(path)
	if err != nil || index < 0 {
		panic(fmt.Sprintf("Cannot get new file index: %v, %d", err, index))
	}
	fmt.Printf("Using path: %s, index: %d\n", path, index)
	return index, path
}

func GetOldestDateDirName(base_path string, skipDirs []string) (string, error) {
	dirs, err := os.ReadDir(base_path)
	if err != nil {
		return "", err
	}
	for _, dir := range dirs {
		if dir.IsDir() && len(dir.Name()) == 10 && !slices.Contains(skipDirs, dir.Name()) {
			return dir.Name(), nil
		}
	}
	return "", nil
}
func GetChunkNames(base_path string, skipDirs []string) ([]string, error) {
	dirs, err := os.ReadDir(base_path)
	if err != nil {
		return nil, err
	}
	var dirNames []string
	for _, dir := range dirs {
		if dir.IsDir() {
			dirNames = append(dirNames, dir.Name())
		}
	}
	if len(dirNames) == 0 {
		return dirNames, nil
	}
	sort.Slice(dirNames, func(i, j int) bool {
		numI, _ := strconv.Atoi(dirNames[i])
		numJ, _ := strconv.Atoi(dirNames[j])
		return numI < numJ
	})
	return dirNames, nil
}
func GetOldestChunkDirName(base_path string, skipDirs []string) (string, error) {
	dirNames, _ := GetChunkNames(base_path, skipDirs)
	if len(dirNames) == 0 {
		return "", nil
	}
	return dirNames[0], nil
}

func GetOldestChunkInDateDir(base_path string, skipDirs []string) string {
	lastDir, _ := GetOldestDateDirName(base_path, skipDirs)
	if lastDir == "" {
		return ""
	}
	chunkPath := fmt.Sprintf("%s/%s", base_path, lastDir)
	lastChunk, _ := GetOldestChunkDirName(chunkPath, skipDirs)
	return fmt.Sprintf("%s/%s", chunkPath, lastChunk)
}
func IsCloseToDirSize(path string) bool {
	currentSize, _ := DirSize(path)
	limit := int64(saveDirMaxSize - (saveChunkSize * 2))
	if limit < 0 {
		limit = saveDirMaxSize
	}
	return currentSize >= limit
}
func RemoveChunk(path string, skipDirs []string) {
	pathToRemove := GetOldestChunkInDateDir(path, skipDirs)
	if pathToRemove == "" {
		lastDateDir, _ := GetOldestDateDirName(path, skipDirs)
		os.ReadDir(fmt.Sprintf("%s/%s", path, lastDateDir))
		pathToRemove = GetOldestChunkInDateDir(path, skipDirs)
		if pathToRemove == "" {
			return
		}
	}
	os.RemoveAll(pathToRemove)
	lastDateDir, _ := GetOldestDateDirName(path, skipDirs)
	datePath := fmt.Sprintf("%s/%s", path, lastDateDir)
	chunkDirName, _ := GetOldestChunkDirName(datePath, skipDirs)
	if chunkDirName == "" {
		os.RemoveAll(datePath)
	}
}

func RemoveOldestDir(savePath string, skipDirs []string) bool {
	if !IsCloseToDirSize(savePath) {
		return false
	}
	fmt.Printf("Removing oldest chunk: %s\n", GetOldestChunkInDateDir(savePath, skipDirs))

	RemoveChunk(savePath, skipDirs)
	return true
}
func IsCloseToVideoSize(path string, extensions []string) bool {
	size, _ := FileSizeByExtension(path, extensions)
	limit := int64(convertedVideoSpace - (saveChunkSize * 2))
	if limit < 0 {
		limit = convertedVideoSpace
	}
	return size >= limit
}
func RemoveOldestVideo(path string, extensions []string, skipDates []string) bool {
	isClose := IsCloseToVideoSize(path, extensions)
	if !isClose {
		return false
	}

	var videos []string
	files, _ := os.ReadDir(path)
	for _, file := range files {
		parts := strings.Split(file.Name(), "-")
		fileDate := fmt.Sprintf("%s-%s-%s", parts[0], parts[1], parts[2])
		ext := filepath.Ext(file.Name())
		if slices.Contains(extensions, ext) && !slices.Contains(skipDates, fileDate) {
			videos = append(videos, file.Name())
		}
	}
	if len(videos) == 0 {
		fmt.Println("No video files found")
		return false
	}
	sort.Strings(videos) // Natural sort works for this format
	oldest := videos[0]

	fmt.Printf("Deleting oldest: %s/%s\n", path, oldest)
	os.Remove(fmt.Sprintf("%s/%s", path, oldest))
	return true
}

func CountChunksInDateDir(base_path string, skipDirs []string) int {
	lastDir, _ := GetOldestDateDirName(base_path, skipDirs)
	if lastDir == "" {
		return 0
	}
	chunkPath := fmt.Sprintf("%s/%s", base_path, lastDir)
	chunks, _ := os.ReadDir(chunkPath)
	return len(chunks)
}
