package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"charm.land/fantasy"
	"github.com/stretchr/testify/require"
)

const loopTestViewArgs = `{"file_path":"main.go","offset":0,"limit":100}`

func TestLoopDetector_BlocksOnConsecutiveIdenticalCalls(t *testing.T) {
	t.Parallel()

	d := NewLoopDetector(3, 0)

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

	d := NewLoopDetector(3, 0)
	for i := 0; i < 10; i++ {
		args := fmt.Sprintf(`{"file_path":"main.go","offset":%d,"limit":100}`, i*100)
		blocked, _ := d.CheckAndRecord("view", args)
		require.False(t, blocked, "paginating with different offsets should not trigger")
	}
}

func TestLoopDetector_EquivalentJSONArguments(t *testing.T) {
	t.Parallel()

	d := NewLoopDetector(3, 0)
	for i, args := range []string{
		`{"pattern":"foo","path":"src"}`,
		`{ "path": "src", "pattern": "foo" }`,
		`{"path":"src","pattern":"\u0066oo"}`,
	} {
		blocked, _ := d.CheckAndRecord("grep", args)
		require.Equal(t, i == 2, blocked)
	}
}

func TestLoopDetector_EquivalentViewArguments(t *testing.T) {
	t.Parallel()

	d := NewLoopDetector(3, 0)
	d.workingDir = t.TempDir()
	absolute, err := json.Marshal(filepath.Join(d.workingDir, "main.go"))
	require.NoError(t, err)
	for i, args := range []string{
		`{"file_path":"./main.go"}`,
		`{"limit":1000,"offset":0,"file_path":"main.go"}`,
		fmt.Sprintf(`{"file_path":%s,"limit":0}`, absolute),
	} {
		blocked, _ := d.CheckAndRecord("view", args)
		require.Equal(t, i == 2, blocked)
	}
}

func TestLoopDetector_OverlappingViews(t *testing.T) {
	t.Parallel()

	d := NewLoopDetector(3, 0)
	for i, args := range []string{
		`{"file_path":"main.go","offset":0,"limit":200}`,
		`{"file_path":"main.go","offset":5,"limit":195}`,
		`{"file_path":"main.go","offset":10,"limit":190}`,
	} {
		blocked, msg := d.CheckAndRecord("view", args)
		require.Equal(t, i == 2, blocked)
		if blocked {
			require.Contains(t, msg, "overlapping region")
		}
	}
}

func TestLoopDetector_SkillDefaultLimit(t *testing.T) {
	t.Parallel()

	d := NewLoopDetector(3, 0)
	d.workingDir = t.TempDir()
	d.skillsPaths = []string{d.workingDir}
	require.NoError(t, os.WriteFile(filepath.Join(d.workingDir, "SKILL.md"), []byte("skill"), 0o644))
	for _, limit := range []int{1000, 1000, 0} {
		args := fmt.Sprintf(`{"file_path":"SKILL.md","limit":%d}`, limit)
		blocked, _ := d.CheckAndRecord("view", args)
		require.False(t, blocked, "the default skill limit must allow expanding beyond 1000 lines")
	}
}

func TestLoopDetector_AllowsViewsWithNewContext(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		offsets []int
		limits  []int
	}{
		{"expanding context", []int{0, 0, 0, 0}, []int{100, 200, 400, 800}},
		{"pagination with context overlap", []int{0, 90, 180, 270}, []int{100, 100, 100, 100}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d := NewLoopDetector(3, 0)
			for i, offset := range tc.offsets {
				args := fmt.Sprintf(`{"file_path":"main.go","offset":%d,"limit":%d}`, offset, tc.limits[i])
				blocked, _ := d.CheckAndRecord("view", args)
				require.False(t, blocked)
			}
		})
	}
}

func TestLoopDetector_FileChangesAllowRereading(t *testing.T) {
	t.Parallel()

	d := NewLoopDetector(3, 0)
	d.workingDir = t.TempDir()
	path := filepath.Join(d.workingDir, "main.go")
	require.NoError(t, os.WriteFile(path, []byte("original"), 0o644))
	for range 2 {
		blocked, _ := d.CheckAndRecord("view", loopTestViewArgs)
		require.False(t, blocked)
	}
	require.NoError(t, os.WriteFile(path, []byte("changed content"), 0o644))
	for range 2 {
		blocked, _ := d.CheckAndRecord("view", loopTestViewArgs)
		require.False(t, blocked)
	}
	blocked, _ := d.CheckAndRecord("view", loopTestViewArgs)
	require.True(t, blocked)
}

func TestLoopDetector_DifferentFilesDoNotShareRanges(t *testing.T) {
	t.Parallel()

	d := NewLoopDetector(3, 0)
	for i := range 5 {
		args := fmt.Sprintf(`{"file_path":"file%d.go","offset":0,"limit":100}`, i)
		blocked, _ := d.CheckAndRecord("view", args)
		require.False(t, blocked)
	}
}

func TestHookedTool_FailedViewsDoNotCountAsReadContext(t *testing.T) {
	t.Parallel()

	inner := &fakeTool{name: "view", resp: fantasy.NewTextErrorResponse("Content section is too large")}
	tool := newHookedTool(inner, nil)
	ctx := WithLoopDetector(t.Context(), NewLoopDetector(3, 0))
	for _, limit := range []int{1000, 500, 100} {
		resp, err := tool.Run(ctx, fantasy.ToolCall{
			Name:  "view",
			Input: fmt.Sprintf(`{"file_path":"main.go","limit":%d}`, limit),
		})
		require.NoError(t, err)
		require.Contains(t, resp.Content, "Content section is too large")
	}
}

func TestHookedTool_BlockedOrRewrittenViewsDoNotCountAsReadContext(t *testing.T) {
	t.Parallel()

	for _, command := range []string{
		`echo '{"decision":"deny","reason":"blocked"}'`,
		`echo '{"updated_input":{"file_path":"other.go","limit":100}}'`,
	} {
		t.Run(command, func(t *testing.T) {
			t.Parallel()

			inner := &fakeTool{name: "view", resp: fantasy.NewTextResponse("ok")}
			tool := newHookedTool(inner, newRunner(t, command))
			detector := NewLoopDetector(3, 0)
			ctx := WithLoopDetector(t.Context(), detector)
			for _, limit := range []int{1000, 500, 100} {
				resp, err := tool.Run(ctx, fantasy.ToolCall{
					Name: "view", Input: fmt.Sprintf(`{"file_path":"main.go","limit":%d}`, limit),
				})
				require.NoError(t, err)
				require.NotContains(t, resp.Content, "Loop detected")
			}
			for _, record := range detector.recentCalls {
				require.Nil(t, record.view)
			}
		})
	}
}

func TestLoopDetector_InterleavedCallsStillDetectedAsOscillation(t *testing.T) {
	t.Parallel()

	d := NewLoopDetector(3, 0)

	blocked, _ := d.CheckAndRecord("view", loopTestViewArgs)
	require.False(t, blocked)
	blocked, _ = d.CheckAndRecord("view", loopTestViewArgs)
	require.False(t, blocked)
	// A different guarded call breaks the consecutive chain, but the
	// oscillation window still sees 3 identical views.
	blocked, _ = d.CheckAndRecord("grep", `{"pattern":"foo"}`)
	require.False(t, blocked)
	blocked, msg := d.CheckAndRecord("view", loopTestViewArgs)
	require.True(t, blocked, "3rd identical view within the window should be blocked even though interleaved")
	require.Contains(t, msg, "Loop detected")
}

func TestLoopDetector_OscillationAcrossTwoCalls(t *testing.T) {
	t.Parallel()

	d := NewLoopDetector(3, 0)
	grepArgs := `{"pattern":"foo"}`

	// A, B, A, B, A: the 3rd A trips the oscillation detector.
	sequence := []struct {
		tool string
		args string
		want bool
	}{
		{"view", loopTestViewArgs, false},
		{"grep", grepArgs, false},
		{"view", loopTestViewArgs, false},
		{"grep", grepArgs, false},
		{"view", loopTestViewArgs, true},
	}
	for i, s := range sequence {
		blocked, _ := d.CheckAndRecord(s.tool, s.args)
		require.Equal(t, s.want, blocked, "call %d (%s)", i+1, s.tool)
	}
}

func TestLoopDetector_ThreeCycleDetected(t *testing.T) {
	t.Parallel()

	d := NewLoopDetector(3, 0)
	grepArgs := `{"pattern":"foo"}`
	lsArgs := `{"path":"."}`

	// A, B, C, A, B, C, ...: the 3rd occurrence of each signature is
	// blocked, even though no two identical calls are adjacent.
	sequence := []struct {
		tool string
		args string
		want bool
	}{
		{"view", loopTestViewArgs, false}, // 1
		{"grep", grepArgs, false},         // 2
		{"ls", lsArgs, false},             // 3
		{"view", loopTestViewArgs, false}, // 4
		{"grep", grepArgs, false},         // 5
		{"ls", lsArgs, false},             // 6
		{"view", loopTestViewArgs, true},  // 7: 3rd identical view
		{"grep", grepArgs, true},          // 8: 3rd identical grep
		{"ls", lsArgs, true},              // 9: 3rd identical ls
	}
	for i, s := range sequence {
		blocked, _ := d.CheckAndRecord(s.tool, s.args)
		require.Equal(t, s.want, blocked, "call %d (%s)", i+1, s.tool)
	}
}

func TestLoopDetector_OldCallsAgeOutOfHistory(t *testing.T) {
	t.Parallel()

	d := NewLoopDetector(3, 0)

	blocked, _ := d.CheckAndRecord("view", loopTestViewArgs)
	require.False(t, blocked)
	// Push the first view out of the retained history (cap 10).
	for i := 0; i < 10; i++ {
		blocked, _ = d.CheckAndRecord("grep", fmt.Sprintf(`{"pattern":"p%d"}`, i))
		require.False(t, blocked)
	}
	blocked, _ = d.CheckAndRecord("view", loopTestViewArgs)
	require.False(t, blocked, "the first view aged out of history")
	blocked, _ = d.CheckAndRecord("view", loopTestViewArgs)
	require.False(t, blocked, "only the 2nd sighting in retained history")
	blocked, _ = d.CheckAndRecord("view", loopTestViewArgs)
	require.True(t, blocked, "3rd sighting in retained history")
}

func TestLoopDetector_OscillationResetsOnMutatingCall(t *testing.T) {
	t.Parallel()

	d := NewLoopDetector(3, 0)

	blocked, _ := d.CheckAndRecord("view", loopTestViewArgs)
	require.False(t, blocked)
	blocked, _ = d.CheckAndRecord("grep", `{"pattern":"foo"}`)
	require.False(t, blocked)
	blocked, _ = d.CheckAndRecord("view", loopTestViewArgs)
	require.False(t, blocked)
	// A mutating call resets the whole history.
	blocked, _ = d.CheckAndRecord("edit", `{"file_path":"main.go"}`)
	require.False(t, blocked)
	blocked, _ = d.CheckAndRecord("view", loopTestViewArgs)
	require.False(t, blocked)
	blocked, _ = d.CheckAndRecord("grep", `{"pattern":"foo"}`)
	require.False(t, blocked)
	blocked, _ = d.CheckAndRecord("view", loopTestViewArgs)
	require.False(t, blocked, "only the 2nd view since the reset")
}

func TestLoopDetector_MutatingToolResetsHistory(t *testing.T) {
	t.Parallel()

	d := NewLoopDetector(3, 0)

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

	d := NewLoopDetector(3, 0)
	for _, tool := range []string{"bash", "edit", "write", "todos", "agent", "question"} {
		for i := 0; i < 6; i++ {
			blocked, _ := d.CheckAndRecord(tool, `{"same":"args"}`)
			require.False(t, blocked, "tool %q should never be blocked", tool)
		}
	}
}

func TestLoopDetector_CustomThreshold(t *testing.T) {
	t.Parallel()

	d := NewLoopDetector(2, 0)

	blocked, _ := d.CheckAndRecord("ls", `{"path":"."}`)
	require.False(t, blocked)
	blocked, _ = d.CheckAndRecord("ls", `{"path":"."}`)
	require.True(t, blocked, "custom threshold of 2 should block the 2nd identical call")
}

func TestLoopDetector_DefaultThresholdIsFour(t *testing.T) {
	t.Parallel()

	d := NewLoopDetector(0, 0)

	for i := 0; i < 3; i++ {
		blocked, _ := d.CheckAndRecord("view", loopTestViewArgs)
		require.False(t, blocked, "call %d should pass with default threshold 4", i+1)
	}
	blocked, msg := d.CheckAndRecord("view", loopTestViewArgs)
	require.True(t, blocked, "4th identical call should be blocked with default threshold 4")
	require.Contains(t, msg, "4 times")
}

func TestLoopDetector_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	d := NewLoopDetector(3, 0)
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

	d := NewLoopDetector(3, 0)
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

	ctx := WithLoopDetector(t.Context(), NewLoopDetector(3, 0))
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

	ctx := WithLoopDetector(t.Context(), NewLoopDetector(3, 0))
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
