package agent

import (
	"regexp"
	"strings"
)

// parseTestOutput attempts to extract a short summary and failing test names.
func parseTestOutput(output string) (string, []string) {
	lines := strings.Split(output, "\n")
	failRe := regexp.MustCompile(`(?i)(FAIL|Error|ERROR):?\s+([A-Za-z0-9_./-]+)`)
	names := make([]string, 0, 8)
	for _, line := range lines {
		m := failRe.FindStringSubmatch(line)
		if len(m) >= 3 {
			names = append(names, strings.TrimSpace(m[2]))
		}
	}
	summary := ""
	if len(names) > 0 {
		summary = "Failing tests: " + strings.Join(unique(names), ", ")
	}
	return summary, unique(names)
}

func unique(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
