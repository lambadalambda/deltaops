package binding

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"deltaops/internal/config"
)

type FileStore struct {
	path string
}

func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

func (s *FileStore) Load() (Binding, bool, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return Binding{}, false, nil
	}
	if err != nil {
		return Binding{}, false, fmt.Errorf("read binding: %w", err)
	}

	var binding Binding
	if err := json.Unmarshal(data, &binding); err != nil {
		return Binding{}, false, fmt.Errorf("parse binding: %w", err)
	}
	if err := validateBinding(binding); err != nil {
		return Binding{}, false, fmt.Errorf("validate binding: %w", err)
	}
	return binding, true, nil
}

func (s *FileStore) Save(binding Binding) error {
	if err := validateBinding(binding); err != nil {
		return fmt.Errorf("validate binding: %w", err)
	}
	data, err := json.MarshalIndent(binding, "", "  ")
	if err != nil {
		return fmt.Errorf("encode binding: %w", err)
	}
	data = append(data, '\n')
	if err := config.WriteSensitiveFile(s.path, data); err != nil {
		return fmt.Errorf("save binding: %w", err)
	}
	return nil
}

func (s *FileStore) Delete() error {
	if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete binding: %w", err)
	}
	return nil
}
