package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/animus-coder/animus-coder/internal/agent"
	"github.com/animus-coder/animus-coder/internal/rpc"
	"github.com/animus-coder/animus-coder/internal/tools"
)

// AgentRunner bridges the agent core to RPC events.
type AgentRunner struct {
	Agent   *agent.Agent
	Metrics interface {
		RecordAgentRun(finishReason string, duration time.Duration, tokenCount int)
		RecordModelUsage(role, model string)
		RecordModelFailure(role, model string)
	}
	Tools    *tools.Registry
	Strategy *agent.StrategyEngine
	Logger   *zap.Logger
}

// Run executes the agent loop with step limits and emits word-based token events.
func (r *AgentRunner) Run(reqCtx *http.Request, req rpc.RunTaskRequest) (<-chan rpc.RunTaskEvent, error) {
	out := make(chan rpc.RunTaskEvent, 16)
	go func() {
		defer close(out)
		start := time.Now()
		expensiveUsed := 0
		corr := req.CorrelationID
		if corr == "" {
			corr = req.SessionID
		}

		if r.Agent == nil {
			out <- rpc.RunTaskEvent{Type: "error", SessionID: req.SessionID, CorrelationID: corr, Error: "agent unavailable"}
			return
		}

		maxBytes := 0
		maxBytes = r.Agent.MaxContextBytes()
		ctxFiles, err := buildContextFiles(r.Tools, req.Prompt, req.ContextPaths, maxBytes)
		if err != nil {
			out <- rpc.RunTaskEvent{Type: "error", SessionID: req.SessionID, CorrelationID: corr, Error: err.Error()}
			return
		}

		tokenCount := 0
		forcedFinish := ""
		initialTools := make([]agent.ToolObservation, 0, len(req.Tools))

		if len(req.Tools) > 0 && r.Tools != nil {
			for _, tc := range req.Tools {
				output, err := executeTool(reqCtx.Context(), r.Tools, tc)
				obs := agent.ToolObservation{Name: tc.Name, Output: output}
				if err != nil {
					obs.Error = err.Error()
					out <- rpc.RunTaskEvent{Type: "tool", SessionID: req.SessionID, CorrelationID: corr, ToolName: tc.Name, ToolOutput: err.Error(), Error: err.Error()}
					return
				}
				initialTools = append(initialTools, obs)
				out <- rpc.RunTaskEvent{Type: "tool", SessionID: req.SessionID, CorrelationID: corr, ToolName: tc.Name, ToolOutput: output}
			}
		}

		if r.Agent != nil && r.Agent.PlanningEnabled() {
			planModel := r.selectModel("planner", firstNonEmpty(req.PlannerModel, req.Model), &expensiveUsed)
			plan, err := r.Agent.Plan(reqCtx.Context(), agent.Request{
				SessionID: req.SessionID,
				Model:     planModel,
				Prompt:    req.Prompt,
				Context:   ctxFiles,
			})
			if err != nil {
				if r.Metrics != nil {
					r.Metrics.RecordModelFailure("planner", planModel)
				}
				r.logf("planner model %s failed: %v", planModel, err)
				if fb := r.pickFallbackModel("planner", planModel, &expensiveUsed); fb != "" {
					planModel = fb
					plan, err = r.Agent.Plan(reqCtx.Context(), agent.Request{
						SessionID: req.SessionID,
						Model:     planModel,
						Prompt:    req.Prompt,
						Context:   ctxFiles,
					})
				}
			}
			if err != nil {
				if r.Metrics != nil {
					r.Metrics.RecordModelFailure("planner", planModel)
				}
				out <- rpc.RunTaskEvent{Type: "error", SessionID: req.SessionID, CorrelationID: corr, Error: err.Error()}
				return
			}
			if strings.TrimSpace(plan) != "" {
				out <- rpc.RunTaskEvent{Type: "plan", SessionID: req.SessionID, CorrelationID: corr, Message: plan}
			}
		}

		maxSteps := r.Agent.MaxSteps()
		for step := 1; step <= maxSteps; step++ {
			if err := reqCtx.Context().Err(); err != nil {
				out <- rpc.RunTaskEvent{Type: "error", SessionID: req.SessionID, CorrelationID: corr, Error: "cancelled"}
				return
			}

			stepTools := append([]agent.ToolObservation{}, initialTools...)
			initialTools = nil

			modelOverride := r.selectModel("coder", firstNonEmpty(req.Model), &expensiveUsed)
			resp, err := r.Agent.Run(reqCtx.Context(), agent.Request{
				SessionID: req.SessionID,
				Model:     modelOverride,
				Prompt:    req.Prompt,
				Context:   ctxFiles,
			})
			if err != nil {
				if r.Metrics != nil {
					r.Metrics.RecordModelFailure("coder", modelOverride)
				}
				r.logf("coder model %s failed: %v", modelOverride, err)
				if fb := r.pickFallbackModel("coder", modelOverride, &expensiveUsed); fb != "" {
					modelOverride = fb
					resp, err = r.Agent.Run(reqCtx.Context(), agent.Request{
						SessionID: req.SessionID,
						Model:     modelOverride,
						Prompt:    req.Prompt,
						Context:   ctxFiles,
					})
				}
				if err != nil {
					if r.Metrics != nil {
						r.Metrics.RecordModelFailure("coder", modelOverride)
					}
					out <- rpc.RunTaskEvent{Type: "error", SessionID: req.SessionID, CorrelationID: corr, Error: err.Error()}
					return
				}
			}

			out <- rpc.RunTaskEvent{
				Type:          "message",
				SessionID:     req.SessionID,
				CorrelationID: corr,
				Message:       resp.Message.Content,
				Step:          step,
			}

			tokens := strings.Fields(resp.Message.Content)
			for idx, token := range tokens {
				select {
				case <-reqCtx.Context().Done():
					out <- rpc.RunTaskEvent{Type: "error", SessionID: req.SessionID, CorrelationID: corr, Error: "cancelled"}
					return
				case out <- rpc.RunTaskEvent{Type: "token", SessionID: req.SessionID, CorrelationID: corr, Token: token, Step: step*1000 + idx}:
				}
			}
			tokenCount += len(tokens)

			// Execute any tool calls emitted by the model before deciding to stop.
			if r.Tools != nil {
				tcalls := extractToolCalls(resp.Message.Content)
				for _, tc := range tcalls {
					output, err := executeTool(reqCtx.Context(), r.Tools, tc)
					obs := agent.ToolObservation{Name: tc.Name, Output: output}
					if err != nil {
						obs.Error = err.Error()
						out <- rpc.RunTaskEvent{Type: "tool", SessionID: req.SessionID, CorrelationID: corr, ToolName: tc.Name, ToolOutput: err.Error(), Error: err.Error()}
						return
					}
					stepTools = append(stepTools, obs)
					out <- rpc.RunTaskEvent{Type: "tool", SessionID: req.SessionID, CorrelationID: corr, ToolName: tc.Name, ToolOutput: output}
				}
			}

			done := isResponseDone(resp)

			var testObs *agent.TestObservation
			if done && r.Agent != nil && r.Agent.TestRunEnabled() {
				output, exitCode, testErr, attempts := r.runTests(reqCtx.Context(), r.Agent.TestCommand(), r.Agent.TestRetries(), r.Agent.TestTimeoutSeconds())
				testSummary, failing := parseTestOutput(output)
				testObs = &agent.TestObservation{
					Command:  r.Agent.TestCommand(),
					Output:   output,
					ExitCode: exitCode,
					Summary:  testSummary,
					Failing:  failing,
					Attempts: attempts,
				}
				if testErr != nil {
					testObs.Error = testErr.Error()
				}
				evt := rpc.RunTaskEvent{Type: "test", SessionID: req.SessionID, CorrelationID: corr, Message: output, Step: step, ExitCode: exitCode, TestSummary: testSummary, TestAttempts: attempts}
				if testErr != nil {
					evt.Error = testErr.Error()
				}
				if len(failing) > 0 {
					evt.FailingTests = failing
				}
				out <- evt
			}

			if r.Agent != nil && r.Agent.ReflectionEnabled() {
				selfDiff := ""
				if r.Agent.EnableSelfDiff() {
					selfDiff = computeSelfDiff(resp.PreviousAssistant, resp.Message.Content)
				}
				criticModel := r.selectModel("critic", firstNonEmpty(req.CriticModel, req.Model), &expensiveUsed)
				reflection, err := r.Agent.Reflect(reqCtx.Context(), agent.Request{
					SessionID: req.SessionID,
					Model:     criticModel,
					Prompt:    req.Prompt,
				}, resp, agent.ReflectionContext{
					Tools:    stepTools,
					Test:     testObs,
					SelfDiff: selfDiff,
				})
				if err != nil {
					if r.Metrics != nil {
						r.Metrics.RecordModelFailure("critic", criticModel)
					}
					r.logf("critic model %s failed: %v", criticModel, err)
					if fb := r.pickFallbackModel("critic", criticModel, &expensiveUsed); fb != "" {
						criticModel = fb
						reflection, err = r.Agent.Reflect(reqCtx.Context(), agent.Request{
							SessionID: req.SessionID,
							Model:     criticModel,
							Prompt:    req.Prompt,
						}, resp, agent.ReflectionContext{
							Tools:    stepTools,
							Test:     testObs,
							SelfDiff: selfDiff,
						})
					}
				}
				if err != nil {
					if r.Metrics != nil {
						r.Metrics.RecordModelFailure("critic", criticModel)
					}
					out <- rpc.RunTaskEvent{Type: "error", SessionID: req.SessionID, CorrelationID: corr, Error: err.Error()}
					return
				}
				if strings.TrimSpace(reflection) != "" {
					critique := parseCritique(reflection)
					out <- rpc.RunTaskEvent{
						Type:          "reflect",
						SessionID:     req.SessionID,
						CorrelationID: corr,
						Message:       reflection,
						Critique:      critique,
						Step:          step,
					}
					if critiqueBlocksApply(critique) && shouldBlockOnCritique(r.Agent.ReflectionPolicy()) {
						done = true
						forcedFinish = "blocked_by_reflect"
					}
				}
			}

			if done {
				finishReason := resp.FinishReason
				if forcedFinish != "" {
					finishReason = forcedFinish
					out <- rpc.RunTaskEvent{
						Type:          "message",
						SessionID:     req.SessionID,
						CorrelationID: corr,
						Message:       fmt.Sprintf("Run halted by reflection policy (%s)", finishReason),
						Step:          step,
					}
				}
				out <- rpc.RunTaskEvent{
					Type:          "done",
					SessionID:     req.SessionID,
					CorrelationID: corr,
					Done:          true,
					FinishReason:  finishReason,
					Step:          step,
				}
				if r.Metrics != nil {
					r.Metrics.RecordAgentRun(finishReason, time.Since(start), tokenCount)
				}
				return
			}
		}

		out <- rpc.RunTaskEvent{
			Type:          "done",
			SessionID:     req.SessionID,
			CorrelationID: corr,
			Done:          true,
			FinishReason:  "max_steps",
			Step:          maxSteps,
		}
		if r.Metrics != nil {
			r.Metrics.RecordAgentRun("max_steps", time.Since(start), tokenCount)
		}
	}()
	return out, nil
}

// EchoRunner is a fallback runner that echoes prompt words.
type EchoRunner struct{}

func (EchoRunner) Run(reqCtx *http.Request, req rpc.RunTaskRequest) (<-chan rpc.RunTaskEvent, error) {
	return runTaskEcho(req), nil
}

func isResponseDone(resp agent.Response) bool {
	if resp.FinishReason != "" && resp.FinishReason != "length" {
		return true
	}
	content := strings.ToLower(resp.Message.Content)
	if strings.Contains(content, "[done]") || strings.Contains(content, "<done>") {
		return true
	}
	return false
}

func executeTool(ctx context.Context, reg *tools.Registry, tc rpc.ToolCall) (string, error) {
	if reg == nil {
		return "", fmt.Errorf("tool registry unavailable")
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return "", err
		}
	}
	if err := tools.ValidateCall(reg, tc.Name, tc.Args); err != nil {
		return "", err
	}
	switch tc.Name {
	case "fs.read_file":
		path, _ := tc.Args["path"].(string)
		return reg.FS.ReadFile(path)
	case "fs.write_file":
		path, _ := tc.Args["path"].(string)
		content, _ := tc.Args["content"].(string)
		if err := reg.FS.WriteFile(path, content); err != nil {
			return "", err
		}
		return "ok", nil
	case "fs.search":
		root, _ := tc.Args["root"].(string)
		pattern, _ := tc.Args["pattern"].(string)
		results, err := reg.FS.Search(root, pattern, 10)
		if err != nil {
			return "", err
		}
		var b strings.Builder
		for _, r := range results {
			b.WriteString(r.Path)
			b.WriteString(":")
			b.WriteString(fmt.Sprintf("%d", r.Line))
			b.WriteString(" ")
			b.WriteString(r.Snippet)
			b.WriteString("\n")
		}
		return b.String(), nil
	case "terminal.exec":
		command, _ := tc.Args["command"].(string)
		var args []string
		if raw, ok := tc.Args["args"].([]interface{}); ok {
			for _, a := range raw {
				if s, ok := a.(string); ok {
					args = append(args, s)
				}
			}
		}
		res, err := reg.Terminal.Exec(ctx, command, args...)
		if err != nil {
			return res.Stderr, err
		}
		return res.Stdout, nil
	case "git.apply_patch":
		if reg.Git == nil {
			return "", fmt.Errorf("git tool unavailable")
		}
		patch, _ := tc.Args["patch"].(string)
		dryRun, ok := tc.Args["dry_run"].(bool)
		if !ok && reg.Git.DryRunOnly {
			dryRun = true
		}
		return reg.Git.ApplyPatch(patch, dryRun)
	case "git.status":
		if reg.Git == nil {
			return "", fmt.Errorf("git tool unavailable")
		}
		return reg.Git.Status()
	case "git.restore_backup":
		if reg.Git == nil {
			return "", fmt.Errorf("git tool unavailable")
		}
		name, _ := tc.Args["name"].(string)
		return reg.Git.RestoreBackup(name)
	case "git.list_backups":
		if reg.Git == nil {
			return "", fmt.Errorf("git tool unavailable")
		}
		list, err := reg.Git.ListBackups()
		if err != nil {
			return "", err
		}
		return strings.Join(list, "\n"), nil
	case "git.preview_backup":
		if reg.Git == nil {
			return "", fmt.Errorf("git tool unavailable")
		}
		name, _ := tc.Args["name"].(string)
		return reg.Git.PreviewBackup(name)
	case "semantic.search":
		if reg.Semantic == nil {
			return "", fmt.Errorf("semantic tool unavailable")
		}
		query, _ := tc.Args["query"].(string)
		limit := 0
		if raw, ok := tc.Args["limit"]; ok {
			switch v := raw.(type) {
			case float64:
				limit = int(v)
			case int:
				limit = v
			case int64:
				limit = int(v)
			}
		}
		results, err := reg.Semantic.Search(query, limit)
		if err != nil {
			return "", err
		}
		var b strings.Builder
		for _, r := range results {
			fmt.Fprintf(&b, "%s (score: %.2f)\n%s\n\n", r.Path, r.Score, r.Snippet)
		}
		return strings.TrimSpace(b.String()), nil
	default:
		return "", fmt.Errorf("unknown tool %q", tc.Name)
	}
}

func (r *AgentRunner) selectModel(role, override string, expensiveUsed *int) string {
	model := firstNonEmpty(override)
	if r.Strategy == nil {
		return model
	}

	_, route, _, isExp, err := r.Strategy.PickWithBudget(role, model, *expensiveUsed)
	if err != nil {
		if r.Metrics != nil {
			r.Metrics.RecordModelFailure(role, model)
		}
		r.logf("%s model selection failed (requested=%s): %v", role, model, err)
		return model
	}
	if route.Name == "" {
		return model
	}
	if isExp {
		*expensiveUsed++
	}
	if r.Metrics != nil {
		r.Metrics.RecordModelUsage(role, route.Name)
	}
	if route.Name != model {
		r.logf("%s model chosen: %s (requested=%s)", role, route.Name, model)
	}
	return route.Name
}

func (r *AgentRunner) pickFallbackModel(role, current string, expensiveUsed *int) string {
	if r.Strategy == nil {
		return ""
	}
	tried := map[string]struct{}{}
	if current != "" {
		tried[current] = struct{}{}
	}

	for fb := r.Strategy.NextFallback(current); fb != ""; fb = r.Strategy.NextFallback(fb) {
		if _, seen := tried[fb]; seen {
			continue
		}
		tried[fb] = struct{}{}

		_, route, _, isExp, err := r.Strategy.PickWithBudget(role, fb, *expensiveUsed)
		if err != nil {
			if r.Metrics != nil {
				r.Metrics.RecordModelFailure(role, fb)
			}
			r.logf("%s fallback selection %s failed: %v", role, fb, err)
			continue
		}
		if route.Name == "" || route.Name == current {
			continue
		}
		if isExp {
			*expensiveUsed++
		}
		if r.Metrics != nil {
			r.Metrics.RecordModelUsage(role, route.Name)
		}
		r.logf("%s model falling back from %s to %s", role, current, route.Name)
		return route.Name
	}
	return ""
}

func (r *AgentRunner) logf(format string, args ...interface{}) {
	if r == nil || r.Logger == nil {
		return
	}
	r.Logger.Sugar().Infof(format, args...)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func (r *AgentRunner) runTests(ctx context.Context, command string, retries int, timeoutSeconds int) (string, int, error, int) {
	if r.Tools == nil || r.Tools.Terminal == nil {
		return "", -1, fmt.Errorf("terminal tool unavailable for tests"), 0
	}
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", -1, fmt.Errorf("test command is empty"), 0
	}
	if retries < 0 {
		retries = 0
	}

	var (
		output   string
		exitCode int
		err      error
	)

	attempts := 0
	for attempts = 1; attempts <= retries+1; attempts++ {
		attemptCtx := ctx
		var cancel context.CancelFunc = func() {}
		if timeoutSeconds > 0 {
			attemptCtx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
		}
		res, execErr := r.Tools.Terminal.Exec(attemptCtx, parts[0], parts[1:]...)
		cancel()

		output = res.Stdout
		if res.Stderr != "" {
			if output != "" {
				output += "\n"
			}
			output += res.Stderr
		}
		exitCode = res.ExitCode
		err = execErr

		if execErr == nil && res.ExitCode == 0 {
			break
		}
	}

	return output, exitCode, err, attempts
}

func buildContextFiles(reg *tools.Registry, prompt string, paths []string, maxBytes int) ([]agent.ContextFile, error) {
	if reg == nil || reg.FS == nil {
		return nil, nil
	}

	candidates := append([]string{}, paths...)
	if len(candidates) == 0 {
		var semPaths []string
		if reg.Semantic != nil {
			if hits, err := reg.Semantic.Search(prompt, 5); err == nil {
				for _, h := range hits {
					semPaths = append(semPaths, h.Path)
				}
			}
		}
		candidates = append(semPaths, discoverContextPaths(reg.FS, prompt)...)
	}

	var (
		total      int
		out        []agent.ContextFile
		seen       = make(map[string]struct{})
		perFileCap = 32 * 1024
		maxContext = maxBytes
	)
	if maxContext > 0 && maxContext < perFileCap {
		perFileCap = maxContext
	}

	for _, p := range candidates {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}

		if maxContext > 0 && total >= maxContext {
			break
		}

		info, err := reg.FS.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("read context %s: %w", p, err)
		}

		if info.IsDir() {
			content, err := reg.FS.DescribeStructure(p, 3, 200)
			if err != nil {
				return nil, fmt.Errorf("describe context %s: %w", p, err)
			}
			exhausted := appendWithBudget(&out, agent.ContextFile{
				Path:    fmt.Sprintf("%s (structure)", strings.TrimSuffix(p, "/")),
				Content: content,
			}, &total, maxContext, perFileCap)
			if exhausted {
				break
			}
			continue
		}

		content, err := reg.FS.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("read context %s: %w", p, err)
		}
		exhausted := appendWithBudget(&out, agent.ContextFile{Path: p, Content: content}, &total, maxContext, perFileCap)
		if exhausted {
			break
		}
	}

	return out, nil
}

func appendWithBudget(out *[]agent.ContextFile, cf agent.ContextFile, total *int, maxBytes int, perFileCap int) bool {
	limit := perFileCap
	if maxBytes > 0 {
		remaining := maxBytes - *total
		if remaining <= 0 {
			return true
		}
		if remaining < limit {
			limit = remaining
		}
	}

	if limit > 0 && len(cf.Content) > limit {
		cf.Content = cf.Content[:limit] + "\n[truncated]"
	}

	*out = append(*out, cf)
	*total += len([]byte(cf.Content))

	return maxBytes > 0 && *total >= maxBytes
}

func discoverContextPaths(fsTool *tools.Filesystem, prompt string) []string {
	defaults := []string{
		".",
		"README.md",
		"CONTRIBUTING.md",
		"go.mod",
		"package.json",
		"Makefile",
		"configs/config.yaml",
		"cmd",
		"internal",
		"src",
	}

	raw := append(extractMentionedPaths(prompt), defaults...)
	seen := make(map[string]struct{})
	out := make([]string, 0, 12)

	for _, cand := range raw {
		cand = strings.TrimSpace(cand)
		if cand == "" {
			continue
		}
		if _, ok := seen[cand]; ok {
			continue
		}
		if len(out) >= 12 {
			break
		}
		if _, err := fsTool.Stat(cand); err != nil {
			continue
		}
		seen[cand] = struct{}{}
		out = append(out, cand)
	}

	return out
}

func extractMentionedPaths(prompt string) []string {
	re := regexp.MustCompile(`[A-Za-z0-9][A-Za-z0-9._/-]+`)
	raw := re.FindAllString(prompt, -1)
	seen := make(map[string]struct{})
	out := make([]string, 0, len(raw))
	for _, token := range raw {
		token = strings.Trim(token, ".,;:!\"'")
		if token == "" {
			continue
		}
		if !strings.ContainsAny(token, "./") {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		out = append(out, token)
	}
	return out
}

// extractToolCalls parses basic JSON tool call structures from model content.
// Supports either a single object {"name":"tool","args":{...}} or an array of such objects.
func extractToolCalls(content string) []rpc.ToolCall {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}
	// Look for fenced code blocks with json for robustness.
	if strings.Contains(content, "```") {
		start := strings.Index(content, "```json")
		if start == -1 {
			start = strings.Index(content, "```")
		}
		if start != -1 {
			end := strings.Index(content[start+3:], "```")
			if end != -1 {
				content = content[start+3 : start+3+end]
				content = strings.TrimSpace(content)
			}
		}
	}
	var calls []rpc.ToolCall
	// try array form
	if strings.HasPrefix(content, "[") {
		if err := json.Unmarshal([]byte(content), &calls); err == nil {
			return calls
		}
	}
	// try single object
	var single rpc.ToolCall
	if err := json.Unmarshal([]byte(content), &single); err == nil && single.Name != "" {
		return []rpc.ToolCall{single}
	}
	return nil
}

func parseCritique(raw string) map[string]interface{} {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func critiqueBlocksApply(crit map[string]interface{}) bool {
	if crit == nil {
		return false
	}
	if v, ok := crit["block_apply"]; ok {
		switch val := v.(type) {
		case bool:
			return val
		case string:
			return strings.ToLower(val) == "true"
		}
	}
	return false
}

func shouldBlockOnCritique(policy string) bool {
	switch strings.ToLower(strings.TrimSpace(policy)) {
	case "never_block":
		return false
	case "warn_only":
		return false
	default:
		return true
	}
}

func computeSelfDiff(prev, current string) string {
	if strings.TrimSpace(prev) == "" || strings.TrimSpace(current) == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("Previous:\n")
	b.WriteString(prev)
	if !strings.HasSuffix(prev, "\n") {
		b.WriteString("\n")
	}
	b.WriteString("Current:\n")
	b.WriteString(current)
	return b.String()
}
