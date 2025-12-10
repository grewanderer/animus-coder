package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// PatchEntry represents a single backup with lineage.
type PatchEntry struct {
	ID        string    `json:"id"`
	ParentID  string    `json:"parent_id,omitempty"`
	FileName  string    `json:"file_name"`
	CreatedAt time.Time `json:"created_at"`
}

type patchStack struct {
	Entries []PatchEntry `json:"entries"`
}

func (s *patchStack) latest() *PatchEntry {
	if len(s.Entries) == 0 {
		return nil
	}
	return &s.Entries[len(s.Entries)-1]
}

func loadPatchStack(path string) (*patchStack, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &patchStack{}, nil
		}
		return nil, err
	}
	var ps patchStack
	if err := json.Unmarshal(data, &ps); err != nil {
		return nil, err
	}
	return &ps, nil
}

func (s *patchStack) save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
