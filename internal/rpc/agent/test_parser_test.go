package agent

import "testing"

func TestParseTestOutputExtractsFailures(t *testing.T) {
	out := `--- FAIL: TestExample (0.00s)
    example_test.go:10: expected 1 got 2
FAIL    my/pkg   0.123s
`
	summary, failing := parseTestOutput(out)
	if len(failing) == 0 || failing[0] != "TestExample" {
		t.Fatalf("expected failing test extracted, got %v", failing)
	}
	if summary == "" {
		t.Fatalf("expected summary")
	}
}
