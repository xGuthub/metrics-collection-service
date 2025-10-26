package repository

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// StateStore defines persistence for metrics state.
// It is intentionally storage-agnostic (accepts plain maps).
type StateStore interface {
	Save(path string, gauges map[string]float64, counters map[string]int64) error
	Load(path string) (gauges map[string]float64, counters map[string]int64, err error)
}

// FileStateStore persists metrics state into a JSON file.
type FileStateStore struct{}

func NewFileStateStore() *FileStateStore { return &FileStateStore{} }

type stateDump struct {
	Gauges   map[string]float64 `json:"gauges"`
	Counters map[string]int64   `json:"counters"`
}

func (f *FileStateStore) Save(path string, gauges map[string]float64, counters map[string]int64) error {
	if path == "" {
		return nil
	}

	dump := stateDump{Gauges: gauges, Counters: counters}

	data, err := json.MarshalIndent(dump, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (f *FileStateStore) Load(path string) (map[string]float64, map[string]int64, error) {
	if path == "" {
		return map[string]float64{}, map[string]int64{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]float64{}, map[string]int64{}, nil
		}
		return nil, nil, err
	}
	var dump stateDump
	if err := json.Unmarshal(data, &dump); err != nil {
		return nil, nil, err
	}
	if dump.Gauges == nil {
		dump.Gauges = map[string]float64{}
	}
	if dump.Counters == nil {
		dump.Counters = map[string]int64{}
	}
	return dump.Gauges, dump.Counters, nil
}
