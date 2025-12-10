package tools

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/animus-coder/animus-coder/internal/tools"
)

func TestSchemaHandler(t *testing.T) {
	reg := tools.NewRegistry(nil, nil, nil, nil)
	h := SchemaHandler{Registry: reg}
	req := httptest.NewRequest(http.MethodGet, "/tools/schemas", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Body.Len() == 0 {
		t.Fatalf("expected body")
	}
}
