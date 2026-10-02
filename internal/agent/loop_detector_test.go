package agent

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"charm.land/fantasy"
	"github.com/stretchr/testify/require"
)

const loopTestViewArgs = `{"file_path":"main.go","offset":0,"limit":100}`

func TestLoopDetector_BlocksOnConsecutiveIdenticalCalls(t *testing.T) {
	t.Parallel()

	d := NewLoopDetector(3)

	blocked, _ := d.CheckAndRecord("view", loopTestViewArgs)
	require.False(t, blocked, "first call should pass")

	blocked, _ = d.CheckAndRecord("view", loopTestViewArgs)
	require.False(t, blocked, "second call should pass")

	blocked, msg := d.CheckAndRecord("view", loopTestViewArgs)
	require.True(t, blocked, "third identical call should be blocked")
	require.Contains(t, msg, "Loop detected")
	require.Contains(t, msg, "grep")

	// Further identical calls stay blocked until the model changes approach.
	blocked, _ = d.CheckAndRecord("view", loopTestViewArgs)
	require.True(t, blocked, "repeated call after block should stay blocked")
}

func TestLoopDetector_DifferentArgumentsDoNotTrigger(t *testing.T) {
	t.Parallel()

	d := NewLoopDetector(3)
	for i := 0; i < 10; i++ {
		args := fmt.Sprintf(`{"file_path":"main.go","offset":%d,"limit":100}`, i*100)
		blocked, _ := d.CheckAndRecord("view", args)
		require.False(t, blocked, "paginating with different offsets should not trigger")
	}
}

func TestLoopDetector_InterleavedCallsBreakTheChain(t *testing.T) {
	t.Parallel()

	d := NewLoopDetector(3)

	blocked, _ := d.CheckAndRecord("view", loopTestViewArgs)
	require.False(t, blocked)
	blocked, _ = d.CheckAndRecord("view", loopTestViewArgs)
	require.False(t, blocked)
	// A different guarded call breaks the consecutive chain.
	blocked, _ = d.CheckAndRecord("grep", `{"pattern":"foo"}`)
	require.False(t, blocked)
	blocked, _ = d.CheckAndRecord("view", loopTestViewArgs)
	require.False(t, blocked)
	blocked, _ = d.CheckAndRecord("view", loopTestViewArgs)
	require.False(t, blocked, "chain was broken by grep, so this is only the 2nd consecutive view")
}

func TestLoopDetector_MutatingToolResetsHistory(t *testing.T) {
	t.Parallel()

	d := NewLoopDetector(3)

	blocked, _ := d.CheckAndRecord("view", loopTestViewArgs)
	require.False(t, blocked)
	blocked, _ = d.CheckAndRecord("view", loopTestViewArgs)
	require.False(t, blocked)

	// A mutating tool counts as forward progress and resets the history.
	blocked, _ = d.CheckAndRecord("edit", `{"file_path":"main.go"}`)
	require.False(t, blocked, "mutating tools are never blocked")

	blocked, _ = d.CheckAndRecord("view", loopTestViewArgs)
	require.False(t, blocked)
	blocked, _ = d.CheckAndRecord("view", loopTestViewArgs)
	require.False(t, blocked)
	blocked, _ = d.CheckAndRecord("view", loopTestViewArgs)
	require.True(t, blocked, "3rd consecutive view after the reset should be blocked")
}

func TestLoopDetector_UnguardedToolsAreNeverBlocked(t *testing.T) {
	t.Parallel()

	d := NewLoopDetector(3)
	for _, tool := range []string{"bash", "edit", "write", "todos", "agent", "question"} {
		for i := 0; i < 6; i++ {
			blocked, _ := d.CheckAndRecord(tool, `{"same":"args"}`)
			require.False(t, blocked, "tool %q should never be blocked", tool)
		}
	}
}

func TestLoopDetector_CustomThreshold(t *testing.T) {
	t.Parallel()

	d := NewLoopDetector(2)

	blocked, _ := d.CheckAndRecord("ls", `{"path":"."}`)
	require.False(t, blocked)
	blocked, _ = d.CheckAndRecord("ls", `{"path":"."}`)
	require.True(t, blocked, "custom threshold of 2 should block the 2nd identical call")
}

func TestLoopDetector_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	d := NewLoopDetector(3)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				d.CheckAndRecord("view", loopTestViewArgs)
				d.CheckAndRecord("edit", `{"file_path":"main.go"}`)
			}
		}()
	}
	wg.Wait()
}

func TestLoopDetector_ContextRoundTrip(t *testing.T) {
	t.Parallel()

	_, ok := loopDetectorFromContext(t.Context())
	require.False(t, ok, "empty context should carry no detector")

	d := NewLoopDetector(3)
	ctx := WithLoopDetector(t.Context(), d)
	got, ok := loopDetectorFromContext(ctx)
	require.True(t, ok)
	require.Same(t, d, got, "context should return the installed detector")
}

// countingTool is a fantasy.AgentTool that counts invocations.
type countingTool struct {
	name  string
	calls int
}

func (c *countingTool) Info() fantasy.ToolInfo { return fantasy.ToolInfo{Name: c.name} }
func (c *countingTool) Run(_ context.Context, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
	c.calls++
	return fantasy.NewTextResponse("ok"), nil
}
func (c *countingTool) ProviderOptions() fantasy.ProviderOptions     { return nil }
func (c *countingTool) SetProviderOptions(_ fantasy.ProviderOptions) {}

func TestHookedTool_LoopDetectorBlocksRepeatedCalls(t *testing.T) {
	t.Parallel()

	inner := &countingTool{name: "view"}
	runner := newRunner(t, `exit 0`) // no hooks configured effectively
	tool := newHookedTool(inner, runner)

	ctx := WithLoopDetector(t.Context(), NewLoopDetector(3))
	call := fantasy.ToolCall{ID: "call-1", Name: "view", Input: loopTestViewArgs}

	resp, err := tool.Run(ctx, call)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	resp, err = tool.Run(ctx, call)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	require.Equal(t, 2, inner.calls, "first two calls should reach the inner tool")

	resp, err = tool.Run(ctx, call)
	require.NoError(t, err)
	require.True(t, resp.IsError, "third identical call should be blocked")
	require.Contains(t, resp.Content, "Loop detected")
	require.Equal(t, 2, inner.calls, "blocked call must not reach the inner tool")
	require.Contains(t, resp.Metadata, `"blocked":true`)
}

func TestHookedTool_LoopDetectorResetsOnMutatingCall(t *testing.T) {
	t.Parallel()

	inner := &countingTool{name: "view"}
	edit := &countingTool{name: "edit"}
	runner := newRunner(t, `exit 0`)
	viewTool := newHookedTool(inner, runner)
	editTool := newHookedTool(edit, runner)

	ctx := WithLoopDetector(t.Context(), NewLoopDetector(3))
	viewCall := fantasy.ToolCall{ID: "v", Name: "view", Input: loopTestViewArgs}
	editCall := fantasy.ToolCall{ID: "e", Name: "edit", Input: `{"file_path":"main.go"}`}

	_, err := viewTool.Run(ctx, viewCall)
	require.NoError(t, err)
	_, err = viewTool.Run(ctx, viewCall)
	require.NoError(t, err)
	// The edit shares the turn's detector and resets the view chain.
	_, err = editTool.Run(ctx, editCall)
	require.NoError(t, err)

	resp, err := viewTool.Run(ctx, viewCall)
	require.NoError(t, err)
	require.False(t, resp.IsError, "view chain was reset by the edit")
	require.Equal(t, 3, inner.calls)
}

func TestHookedTool_NoDetectorInContextSkipsDetection(t *testing.T) {
	t.Parallel()

	inner := &countingTool{name: "view"}
	runner := newRunner(t, `exit 0`)
	tool := newHookedTool(inner, runner)

	// No detector installed: behavior is exactly as before.
	for i := 0; i < 5; i++ {
		resp, err := tool.Run(t.Context(), fantasy.ToolCall{ID: "c", Name: "view", Input: loopTestViewArgs})
		require.NoError(t, err)
		require.False(t, resp.IsError)
	}
	require.Equal(t, 5, inner.calls)
}
