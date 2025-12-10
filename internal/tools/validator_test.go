package tools

import (
	"testing"

	"github.com/animus-coder/animus-coder/internal/semantic"
)

func TestValidateCallSchema(t *testing.T) {
	reg := NewRegistry(nil, nil, nil, nil)
	err := ValidateCall(reg, "fs.read_file", map[string]interface{}{"path": "file.txt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ValidateCall(reg, "fs.read_file", map[string]interface{}{}); err == nil {
		t.Fatalf("expected missing path error")
	}

	if err := ValidateCall(reg, "terminal.exec", map[string]interface{}{"command": 123}); err == nil {
		t.Fatalf("expected type error")
	}
}

func TestValidateSemanticSearch(t *testing.T) {
	reg := NewRegistry(nil, nil, nil, nil)
	if err := ValidateCall(reg, "semantic.search", map[string]interface{}{"query": "hi"}); err == nil {
		t.Fatalf("expected error when semantic engine is nil")
	}
	reg = NewRegistry(nil, nil, nil, &semantic.Engine{})
	if err := ValidateCall(reg, "semantic.search", map[string]interface{}{"query": "hi", "limit": 3}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
