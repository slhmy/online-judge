package storage

import (
	"io"
	"os"
	"path/filepath"
)

func (s *StorageService) uploadLocal(path string, reader io.Reader, size int64) error {
	fullPath := filepath.Join(s.localPath, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	_, err = io.Copy(file, reader)
	return err
}

func (s *StorageService) downloadLocal(path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.localPath, path)
	return os.Open(fullPath)
}

func (s *StorageService) deleteLocal(path string) error {
	fullPath := filepath.Join(s.localPath, path)
	return os.Remove(fullPath)
}

func (s *StorageService) existsLocal(path string) (bool, error) {
	fullPath := filepath.Join(s.localPath, path)
	_, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}
