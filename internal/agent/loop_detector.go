package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/charmbracelet/crush/internal/agent/tools"
)

// defaultLoopMaxRepeats is the number of consecutive identical read-only
// tool calls that trips the circuit breaker.
const defaultLoopMaxRepeats = 3

// loopDetectorHistorySize caps how many recent tool calls are retained.
// It only needs to cover a few multiples of the repeat threshold.
const loopDetectorHistorySize = 10

// loopDetectorContextKey is the context key carrying the current turn's
// LoopDetector, installed by the coordinator.
type loopDetectorContextKey struct{}

// LoopDetector watches tool invocations within a single agent turn and
// blocks read-only exploration tools (view, grep, glob, ls) when the
// model issues the exact same call too many times in a row, a strong
// signal of an infinite loop. A blocked call never executes; instead a
// guidance message is returned as the tool result so the model can
// recover within the same turn. Any other tool (edit, write, bash, ...)
// counts as forward progress and resets the repetition history.
type LoopDetector struct {
	mu          sync.Mutex
	maxRepeats  int
	recentCalls []toolCallRecord
}

// toolCallRecord is a single recorded tool invocation.
type toolCallRecord struct {
	toolName string
	argsHash string
}

// NewLoopDetector returns a LoopDetector that blocks a guarded tool call
// once it has been issued maxRepeats times consecutively with identical
// arguments. A non-positive maxRepeats selects the default threshold.
func NewLoopDetector(maxRepeats int) *LoopDetector {
	if maxRepeats <= 0 {
		maxRepeats = defaultLoopMaxRepeats
	}
	return &LoopDetector{maxRepeats: maxRepeats}
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
	repeats := 1
	for i := len(d.recentCalls) - 1; i >= 0; i-- {
		if d.recentCalls[i].toolName == toolName && d.recentCalls[i].argsHash == hash {
			repeats++
		} else {
			break
		}
	}

	d.recentCalls = append(d.recentCalls, toolCallRecord{toolName: toolName, argsHash: hash})
	if len(d.recentCalls) > loopDetectorHistorySize {
		d.recentCalls = d.recentCalls[len(d.recentCalls)-loopDetectorHistorySize:]
	}

	if repeats >= d.maxRepeats {
		return true, loopWarning(toolName, repeats)
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
		"Loop detected: you have called '%s' with identical arguments %d times in a row. "+
			"This call was blocked to prevent an infinite loop. Do not call '%s' with the same arguments again. "+
			"If you are searching for code, use 'grep' with a specific pattern or the LSP tools "+
			"('lsp_definition', 'lsp_symbols', 'references') instead of re-reading the same region. "+
			"If you already have the information you need, move on to the next step.",
		toolName, repeats, toolName,
	)
}
