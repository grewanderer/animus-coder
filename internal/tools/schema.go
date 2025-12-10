package tools

// Schema describes a tool for JSON schema/tool-calling.
type Schema struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Parameters  []SchemaField `json:"parameters"`
}

// SchemaField describes a single parameter.
type SchemaField struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Required    bool     `json:"required"`
	Enum        []string `json:"enum,omitempty"`
}

// Schemas provides descriptors for available tools (stub).
func (r *Registry) Schemas() []Schema {
	s := []Schema{
		{
			Name:        "fs.read_file",
			Description: "Read a file relative to workspace",
			Parameters: []SchemaField{
				{Name: "path", Type: "string", Description: "Relative file path", Required: true},
			},
		},
		{
			Name:        "fs.write_file",
			Description: "Write content to a file",
			Parameters: []SchemaField{
				{Name: "path", Type: "string", Required: true},
				{Name: "content", Type: "string", Required: true},
				{Name: "overwrite", Type: "boolean", Required: false},
			},
		},
		{
			Name:        "terminal.exec",
			Description: "Execute a command",
			Parameters: []SchemaField{
				{Name: "command", Type: "string", Required: true},
				{Name: "args", Type: "array", Description: "Arguments", Required: false},
			},
		},
		{
			Name:        "git.apply_patch",
			Description: "Apply a git patch; use dry_run true to validate without applying",
			Parameters: []SchemaField{
				{Name: "patch", Type: "string", Required: true},
				{Name: "dry_run", Type: "boolean", Required: false},
			},
		},
		{
			Name:        "git.restore_backup",
			Description: "Restore the latest saved patch backup (or specific id/name if provided)",
			Parameters: []SchemaField{
				{Name: "name", Type: "string", Required: false},
			},
		},
		{
			Name:        "git.list_backups",
			Description: "List saved patch backups",
			Parameters:  []SchemaField{},
		},
		{
			Name:        "git.preview_backup",
			Description: "Preview a backup by name (or latest if not provided)",
			Parameters: []SchemaField{
				{Name: "name", Type: "string", Required: false},
			},
		},
	}
	if r.Semantic != nil {
		s = append(s, Schema{
			Name:        "semantic.search",
			Description: "Find relevant files by semantic overlap (lightweight tokenizer-based search)",
			Parameters: []SchemaField{
				{Name: "query", Type: "string", Required: true},
				{Name: "limit", Type: "integer", Required: false},
			},
		})
	}
	return s
}
