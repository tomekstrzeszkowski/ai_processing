package watcher

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
