package agent

import (
	"fmt"
	"strings"

	"github.com/animus-coder/animus-coder/internal/config"
)

// buildSystemPrompt returns the base system prompt given agent configuration.
func buildSystemPrompt(cfg config.AgentConfig) string {
	return strings.TrimSpace(`
You are MyCodex, a coding agent. Follow user instructions precisely, prefer minimal changes, and ask before destructive actions. Be concise in answers.`)
}

// buildPlanSystemPrompt defines the planning instruction prompt.
func buildPlanSystemPrompt(cfg config.AgentConfig) string {
	return strings.TrimSpace(`
You are MyCodex planning assistant. Draft a concise numbered plan (3-7 steps) to solve the user's task. Plans should include inspections, edits, validations, and tests when relevant. Do not execute actions; only outline the plan.`)
}

// buildReflectSystemPrompt instructs the model to critique the last action.
func buildReflectSystemPrompt(cfg config.AgentConfig) string {
	return strings.TrimSpace(`
You are MyCodex reflection assistant. Briefly assess the last assistant response for issues, risks, or missing checks. Return a JSON object matching:
{"quality":"good|ok|poor","issues":["..."],"recommendations":["..."],"block_apply":true|false,"notes":"optional free-text"}
Be concise in text fields. Prefer block_apply=true only when you see critical risks.`)
}

// buildUserPrompt embeds user prompt with optional context files.
func buildUserPrompt(prompt string, ctx []ContextFile) string {
	if len(ctx) == 0 {
		return prompt
	}

	var b strings.Builder
	b.WriteString(prompt)
	b.WriteString("\n\nContext:\n")
	for _, f := range ctx {
		fmt.Fprintf(&b, "File: %s\n", f.Path)
		b.WriteString(f.Content)
		if !strings.HasSuffix(f.Content, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("---\n")
	}
	return b.String()
}

// buildPlanUserPrompt formats the user task for planning mode.
func buildPlanUserPrompt(prompt string) string {
	return fmt.Sprintf("Task:\n%s\n\nReturn only the numbered plan.", prompt)
}

// buildReflectUserPrompt formats the reflection request.
func buildReflectUserPrompt(prompt string, lastResponse string, plan string, ctx ReflectionContext) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Task:\n%s\n\nLast response:\n%s\n", prompt, lastResponse)
	if strings.TrimSpace(plan) != "" {
		fmt.Fprintf(&b, "\nPlanned steps:\n%s\n", plan)
	}
	if strings.TrimSpace(ctx.SelfDiff) != "" {
		fmt.Fprintf(&b, "\nSelf-diff (last vs current response):\n%s\n", truncateForPrompt(ctx.SelfDiff, 2000))
	}
	if len(ctx.Tools) > 0 {
		b.WriteString("\nTool results:\n")
		for _, t := range ctx.Tools {
			summary := t.Output
			if summary == "" {
				summary = t.Error
			}
			summary = truncateForPrompt(summary, 800)
			if summary == "" {
				summary = "(no output)"
			}
			fmt.Fprintf(&b, "- %s: %s\n", t.Name, summary)
		}
	}
	if ctx.Test != nil {
		testCmd := strings.TrimSpace(ctx.Test.Command)
		if testCmd == "" {
			testCmd = "tests"
		}
		output := truncateForPrompt(ctx.Test.Output, 1200)
		fmt.Fprintf(&b, "\nTest run (%s) exit=%d attempts=%d:\n", testCmd, ctx.Test.ExitCode, ctx.Test.Attempts)
		if len(ctx.Test.Failing) > 0 {
			fmt.Fprintf(&b, "Failing tests: %s\n", strings.Join(ctx.Test.Failing, ", "))
		}
		if strings.TrimSpace(ctx.Test.Summary) != "" {
			fmt.Fprintf(&b, "Summary: %s\n", ctx.Test.Summary)
		}
		if strings.TrimSpace(output) != "" {
			fmt.Fprintf(&b, "%s\n", output)
		}
		if strings.TrimSpace(ctx.Test.Error) != "" {
			fmt.Fprintf(&b, "Error: %s\n", truncateForPrompt(ctx.Test.Error, 400))
		}
	}
	b.WriteString("\nReturn only the JSON critique.")
	return b.String()
}

func truncateForPrompt(text string, limit int) string {
	if limit <= 0 || len(text) <= limit {
		return text
	}
	return text[:limit] + "... [truncated]"
}
