package watcher

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

func SaveFrame(i int, b []byte, path string) {
	//log.Printf("Saving frame to %s/frame%d\n", path, i)
	f, err := os.Create(fmt.Sprintf("%s/frame%d.yuv", path, i))
	if err != nil {
		panic(fmt.Sprintf("Cant create file: %v", err))
	}
	defer f.Close()
	f.Write(b)
}
func SaveMetadata(width, height uint32, path string) {
	log.Printf("Saving frame to %s/meta.txt\n", path)
	f, err := os.Create(fmt.Sprintf("%s/meta.txt", path))
	if err != nil {
		panic(fmt.Sprintf("Cant create file: %v", err))
	}
	defer f.Close()
	f.Write([]byte(fmt.Sprintf("%d %d", width, height)))
}
func IsMetadataExists(path string) bool {
	_, err := os.Stat(fmt.Sprintf("%s/meta.txt", path))
	return !errors.Is(err, os.ErrNotExist)
}
func ReadMetadata(path string) (uint32, uint32, error) {
	data, err := os.ReadFile(fmt.Sprintf("%s/meta.txt", path))
	if err != nil {
		return 0, 0, err
	}
	parts := strings.Split(string(data), " ")
	width, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return 0, 0, err
	}
	height, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return 0, 0, err
	}
	return uint32(width), uint32(height), nil
}
func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error accessing file %v", err)
			return err
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
			_, err := fmt.Sscanf(file.Name(), "frame%d.yuv", &num)
			if err == nil && num > maxNum {
				maxNum = num
			}
		}
	}

	return maxNum + 1, nil
}
func GetDirsSortedByCreatedDesc(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	type dirInfo struct {
		name  string
		ctime int64
	}
	var dirs []dirInfo
	for _, entry := range entries {
		if entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			// Use ModTime as a proxy for creation time (Go doesn't expose ctime cross-platform)
			dirs = append(dirs, dirInfo{name: entry.Name(), ctime: info.ModTime().Unix()})
		}
	}
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].ctime > dirs[j].ctime // DESC
	})
	names := make([]string, len(dirs))
	for i, d := range dirs {
		names[i] = d.name
	}
	return names, nil
}
func TouchLastDirIndex(basePath string) int {
	dirIndex := 1
	names, err := GetDirsSortedByCreatedDesc(basePath)
	if err != nil {
		path := filepath.Join(basePath, fmt.Sprintf("%d", dirIndex))
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(path, 0755); err != nil {
				panic(fmt.Sprintf("Cannot create directory: %v", err))
			}
		}
		return dirIndex
	}
	if len(names) > 0 {
		lastDir, _ := strconv.Atoi(names[0])
		dirIndex = lastDir
	}
	return dirIndex
}
func CreateNewDirIndex(basePath string) int {
	newIndex := TouchLastDirIndex(basePath) + 1
	path := filepath.Join(basePath, fmt.Sprintf("%d", newIndex))
	if err := os.MkdirAll(path, 0755); err != nil {
		panic(fmt.Sprintf("Cannot create directory: %v", err))
	}
	return newIndex
}
func TouchDirAndGetIndex(basePath string, sizeLimit int64) (int, string, error) {
	dirIndex := TouchLastDirIndex(basePath)
	path := filepath.Join(basePath, fmt.Sprintf("%d", dirIndex))
	for {
		size, err := DirSize(path)
		if err != nil {
			return -1, "", err
		}
		if size < sizeLimit {
			break
		}
		dirIndex += 1
		path = filepath.Join(basePath, fmt.Sprintf("%d", dirIndex))
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
	return index, path, nil
}
func GetDateDirNames(basePath string, skipDirs []string) ([]string, error) {
	dirs, err := os.ReadDir(basePath)
	if err != nil {
		return nil, err
	}
	dirNames := make([]string, 0)
	for _, dir := range dirs {
		if dir.IsDir() && len(dir.Name()) == 10 && !slices.Contains(skipDirs, dir.Name()) {
			dirNames = append(dirNames, dir.Name())
		}
	}
	return dirNames, nil
}
func GetOldestDateDirName(basePath string, skipDirs []string) (string, error) {
	dirs, err := GetDateDirNames(basePath, skipDirs)
	if err != nil {
		return "", err
	}
	if len(dirs) == 0 {
		return "", nil
	}
	return dirs[0], nil
}
func GetChunkNames(basePath string, skipDirs []string) ([]string, error) {
	dirs, err := os.ReadDir(basePath)
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
func GetOldestChunkDirName(basePath string, skipDirs []string) (string, error) {
	dirNames, _ := GetChunkNames(basePath, skipDirs)
	if len(dirNames) == 0 {
		return "", nil
	}
	fmt.Printf("Oldest chunk dir names: %v\n", dirNames[0])
	return dirNames[0], nil
}

func GetOldestChunkInDateDir(basePath string, skipDirs []string) string {
	lastDir, _ := GetOldestDateDirName(basePath, skipDirs)
	if lastDir == "" {
		return ""
	}
	chunkPath := fmt.Sprintf("%s/%s", basePath, lastDir)
	lastChunk, _ := GetOldestChunkDirName(chunkPath, skipDirs)
	return fmt.Sprintf("%s/%s", chunkPath, lastChunk)
}
func IsCloseToDirSize(path string, saveChunkSize int, saveDirMaxSize int) bool {
	currentSize, _ := DirSize(path)
	limit := int64(saveDirMaxSize - (saveChunkSize * 2))
	if limit < 0 {
		limit = int64(saveDirMaxSize)
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

func RemoveOldestDir(savePath string, skipDirs []string, saveChunkSize int, saveDirMaxSize int) bool {
	if !IsCloseToDirSize(savePath, saveChunkSize, saveDirMaxSize) {
		return false
	}
	fmt.Printf("Removing oldest chunk: %s\n", GetOldestChunkInDateDir(savePath, skipDirs))

	RemoveChunk(savePath, skipDirs)
	return true
}
func IsCloseToVideoSize(path string, extensions []string, convertedVideoSpace int, saveChunkSize int) bool {
	size, _ := FileSizeByExtension(path, extensions)
	limit := int64(convertedVideoSpace - (saveChunkSize * 2))
	if limit < 0 {
		limit = int64(convertedVideoSpace)
	}
	return size >= limit
}
func RemoveOldestVideo(path string, extensions []string, skipDates []string, convertedVideoSpace int, saveChunkSize int) bool {
	isClose := IsCloseToVideoSize(path, extensions, convertedVideoSpace, saveChunkSize)
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

func CountChunksInDateDir(basePath string, skipDirs []string) int {
	lastDir, _ := GetOldestDateDirName(basePath, skipDirs)
	if lastDir == "" {
		return 0
	}
	chunkPath := fmt.Sprintf("%s/%s", basePath, lastDir)
	chunks, _ := os.ReadDir(chunkPath)
	fmt.Printf("Chunks in date dir %v\n", chunks)
	return len(chunks)
}
