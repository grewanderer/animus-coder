package tools

import (
	"encoding/json"
	"net/http"

	"github.com/animus-coder/animus-coder/internal/tools"
)

// SchemaHandler serves tool schemas as JSON.
type SchemaHandler struct {
	Registry *tools.Registry
}

// ServeHTTP renders schemas.
func (h SchemaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.Registry.Schemas())
}
