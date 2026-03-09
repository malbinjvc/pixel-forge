package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type FileStorage struct {
	baseDir string
	mu      sync.RWMutex
}

func New(baseDir string) (*FileStorage, error) {
	if err := os.MkdirAll(baseDir, 0750); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}
	return &FileStorage{baseDir: baseDir}, nil
}

func (s *FileStorage) Save(name string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.baseDir, filepath.Base(name))
	return os.WriteFile(path, data, 0600)
}

func (s *FileStorage) Load(name string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.baseDir, filepath.Base(name))
	return os.ReadFile(path)
}

func (s *FileStorage) SaveFromReader(name string, r io.Reader) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.baseDir, filepath.Base(name))
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, r)
	return err
}

func (s *FileStorage) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.baseDir, filepath.Base(name))
	return os.Remove(path)
}

func (s *FileStorage) Exists(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.baseDir, filepath.Base(name))
	_, err := os.Stat(path)
	return err == nil
}

func (s *FileStorage) Stats() (int, int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	var totalSize int64

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return 0, 0
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		count++
		totalSize += info.Size()
	}

	return count, totalSize
}

func (s *FileStorage) Path(name string) string {
	return filepath.Join(s.baseDir, filepath.Base(name))
}
