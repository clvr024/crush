package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/charmbracelet/crush/internal/agent/tools"
	"github.com/charmbracelet/crush/internal/config"
)

// loopDetectorContextKey is the context key carrying the current turn's
// LoopDetector, installed by the coordinator.
type loopDetectorContextKey struct{}

// LoopDetector watches tool invocations within a single agent turn and
// blocks read-only exploration tools (view, grep, glob, ls) when the
// model issues the exact same call maxRepeats times within the retained
// history, whether back-to-back or interleaved with other calls (e.g.
// view A, grep B, view A, ... or A, B, C, A, B, C, ...). That is a
// strong signal of an infinite loop: re-issuing an identical read-only
// call can only return bytes the model already has. A blocked call
// never executes; instead a guidance message is returned as the tool
// result so the model can recover within the same turn. Any other tool
// (edit, write, bash, ...) counts as forward progress and resets the
// repetition history.
type LoopDetector struct {
	mu          sync.Mutex
	maxRepeats  int
	maxHistory  int
	recentCalls []toolCallRecord
}

// toolCallRecord is a single recorded tool invocation.
type toolCallRecord struct {
	toolName string
	argsHash string
}

// NewLoopDetector returns a LoopDetector that blocks a guarded tool call
// once the identical (tool, arguments) signature has been issued
// maxRepeats times within the retained history. Non-positive maxRepeats
// or historySize select the configured defaults
// (config.DefaultLoopMaxRepeats, config.DefaultLoopHistorySize).
func NewLoopDetector(maxRepeats, historySize int) *LoopDetector {
	if maxRepeats <= 0 {
		maxRepeats = config.DefaultLoopMaxRepeats
	}
	if historySize <= 0 {
		historySize = config.DefaultLoopHistorySize
	}
	if historySize < maxRepeats {
		historySize = maxRepeats
	}
	return &LoopDetector{maxRepeats: maxRepeats, maxHistory: historySize}
}

// WithLoopDetector installs d into ctx for the current turn.
func WithLoopDetector(ctx context.Context, d *LoopDetector) context.Context {
	return context.WithValue(ctx, loopDetectorContextKey{}, d)
}

// loopDetectorFromContext returns the current turn's LoopDetector, if
// one is installed.
func loopDetectorFromContext(ctx context.Context) (*LoopDetector, bool) {
	d, ok := ctx.Value(loopDetectorContextKey{}).(*LoopDetector)
	return d, ok && d != nil
}

// CheckAndRecord evaluates a tool call against the repetition history
// and records it. It returns blocked=true with a guidance message for
// the model when the call reaches the repeat threshold; the caller
// should return the message as the tool result instead of executing
// the tool. Calls to tools outside the guarded set are never blocked
// and reset the history, since they represent forward progress.
func (d *LoopDetector) CheckAndRecord(toolName, rawArgs string) (blocked bool, message string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !isLoopGuardedTool(toolName) {
		d.recentCalls = nil
		return false, ""
	}

	hash := hashToolArgs(rawArgs)
	d.recentCalls = append(d.recentCalls, toolCallRecord{toolName: toolName, argsHash: hash})
	if len(d.recentCalls) > d.maxHistory {
		d.recentCalls = d.recentCalls[len(d.recentCalls)-d.maxHistory:]
	}

	same := func(r toolCallRecord) bool {
		return r.toolName == toolName && r.argsHash == hash
	}

	// Consecutive identical calls.
	consecutive := 0
	for i := len(d.recentCalls) - 1; i >= 0; i-- {
		if !same(d.recentCalls[i]) {
			break
		}
		consecutive++
	}
	if consecutive >= d.maxRepeats {
		return true, loopWarning(toolName, consecutive)
	}

	// Recurrence anywhere in the retained history: the same call issued
	// maxRepeats times, even with other calls interleaved. This catches
	// oscillating patterns like A, B, A, B, ... or A, B, C, A, B, C, ...
	// that the consecutive check above misses.
	total := 0
	for _, r := range d.recentCalls {
		if same(r) {
			total++
		}
	}
	if total >= d.maxRepeats {
		return true, loopWarning(toolName, total)
	}
	return false, ""
}

// isLoopGuardedTool reports whether a tool is a read-only exploration
// tool whose repetition is worth guarding.
func isLoopGuardedTool(name string) bool {
	switch name {
	case tools.ViewToolName, tools.GrepToolName, tools.GlobToolName, tools.LSToolName:
		return true
	default:
		return false
	}
}

// hashToolArgs returns a short stable hash of the raw tool arguments.
func hashToolArgs(args string) string {
	sum := sha256.Sum256([]byte(args))
	return hex.EncodeToString(sum[:8])
}

// loopWarning builds the guidance message returned to the model when a
// call is blocked. It names the problem and points at concrete
// alternatives instead of just refusing.
func loopWarning(toolName string, repeats int) string {
	return fmt.Sprintf(
		"Loop detected: you have called '%s' with identical arguments %d times recently. "+
			"This call was blocked to prevent an infinite loop. Do not call '%s' with the same arguments again. "+
			"If you are searching for code, use 'grep' with a specific pattern or the LSP tools "+
			"('lsp_definition', 'lsp_symbols', 'references') instead of re-reading the same region. "+
			"If you already have the information you need, move on to the next step.",
		toolName, repeats, toolName,
	)
}
