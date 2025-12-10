package tools

import (
	"errors"
	"fmt"
)

// ValidateCall performs minimal validation of tool call arguments.
func ValidateCall(reg *Registry, name string, args map[string]interface{}) error {
	if reg == nil {
		return errors.New("tool registry unavailable")
	}
	if schema, ok := reg.Schema(name); ok {
		if err := validateAgainstSchema(schema, args); err != nil {
			return err
		}
	}
	switch name {
	case "fs.read_file", "fs.write_file", "fs.search":
		if _, ok := args["path"].(string); !ok && name != "fs.search" {
			return fmt.Errorf("path is required and must be string")
		}
		if name == "fs.write_file" {
			if _, ok := args["content"].(string); !ok {
				return fmt.Errorf("content is required and must be string")
			}
			if !reg.FS.allowWrite {
				return fmt.Errorf("write operations are disabled by configuration")
			}
		}
		if name == "fs.search" {
			if _, ok := args["pattern"].(string); !ok {
				return fmt.Errorf("pattern is required and must be string")
			}
		}
	case "terminal.exec":
		if reg.Terminal == nil || !reg.Terminal.AllowExecution {
			return fmt.Errorf("exec disabled by configuration")
		}
		if _, ok := args["command"].(string); !ok {
			return fmt.Errorf("command is required and must be string")
		}
	case "git.apply_patch", "git.status":
		if reg.Git == nil || !reg.Git.AllowExec {
			return fmt.Errorf("git operations disabled")
		}
		if name == "git.apply_patch" {
			if _, ok := args["patch"].(string); !ok {
				return fmt.Errorf("patch is required")
			}
			if reg.Git.DryRunOnly {
				if dry, _ := args["dry_run"].(bool); !dry {
					return fmt.Errorf("apply_patch allowed only in dry-run mode")
				}
			}
		}
	case "git.restore_backup", "git.list_backups", "git.preview_backup":
		if reg.Git == nil || !reg.Git.AllowExec {
			return fmt.Errorf("git operations disabled")
		}
		if name == "git.restore_backup" {
			if val, ok := args["name"]; ok {
				if _, ok := val.(string); !ok {
					return fmt.Errorf("name must be string")
				}
			}
		}
	case "semantic.search":
		if reg.Semantic == nil {
			return fmt.Errorf("semantic engine unavailable")
		}
		if _, ok := args["query"].(string); !ok {
			return fmt.Errorf("query is required and must be string")
		}
		if limit, ok := args["limit"]; ok {
			switch limit.(type) {
			case float64, int, int64:
			default:
				return fmt.Errorf("limit must be number")
			}
		}
	default:
		return fmt.Errorf("unknown tool %q", name)
	}
	return nil
}

func validateAgainstSchema(schema Schema, args map[string]interface{}) error {
	for _, field := range schema.Parameters {
		val, exists := args[field.Name]
		if field.Required && !exists {
			return fmt.Errorf("%s is required", field.Name)
		}
		if !exists {
			continue
		}
		switch field.Type {
		case "string":
			if _, ok := val.(string); !ok {
				return fmt.Errorf("%s must be string", field.Name)
			}
		case "boolean":
			if _, ok := val.(bool); !ok {
				return fmt.Errorf("%s must be boolean", field.Name)
			}
		case "array":
			if _, ok := val.([]interface{}); !ok {
				return fmt.Errorf("%s must be array", field.Name)
			}
		case "integer":
			switch val.(type) {
			case float64, int, int64:
			default:
				return fmt.Errorf("%s must be integer", field.Name)
			}
		}
		if len(field.Enum) > 0 {
			s, _ := val.(string)
			valid := false
			for _, allowed := range field.Enum {
				if s == allowed {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("%s must be one of %v", field.Name, field.Enum)
			}
		}
	}
	return nil
}
