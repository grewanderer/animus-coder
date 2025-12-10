package tools

import "github.com/animus-coder/animus-coder/internal/semantic"

// Registry exposes shared tool instances.
type Registry struct {
	FS       *Filesystem
	Terminal *Terminal
	Git      *GitTool
	Semantic *semantic.Engine
}

// NewRegistry builds a registry from instantiated tools.
func NewRegistry(fs *Filesystem, term *Terminal, git *GitTool, sem *semantic.Engine) *Registry {
	return &Registry{FS: fs, Terminal: term, Git: git, Semantic: sem}
}

// Schema returns schema for a given tool name if present.
func (r *Registry) Schema(name string) (Schema, bool) {
	for _, s := range r.Schemas() {
		if s.Name == name {
			return s, true
		}
	}
	return Schema{}, false
}
