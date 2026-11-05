package chat

import (
	"strings"
	"testing"

	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

func TestToolsDefaultToSingleLine(t *testing.T) {
	t.Parallel()

	sty := styles.CharmtonePantera()
	for _, tc := range []struct {
		name  string
		input string
	}{
		{"view", `{"file_path":"main.go"}`},
		{"bash", `{"command":"echo one\necho two"}`},
		{"grep", `{"pattern":"handler","path":"src"}`},
		{"mcp_example_search", `{"query":"example"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			item := NewToolMessageItem(&sty, "msg", message.ToolCall{
				ID: "tool", Name: tc.name, Input: tc.input, Finished: true,
			}, &message.ToolResult{ToolCallID: "tool", Content: "first output\nlast output"}, false, "")
			collapsed := ansi.Strip(item.Render(80))
			require.NotContains(t, collapsed, "\n")
			require.NotContains(t, collapsed, "first output")
			require.LessOrEqual(t, ansi.StringWidth(collapsed), 80)

			expandable := item.(Expandable)
			require.True(t, expandable.ToggleExpanded())
			require.Contains(t, ansi.Strip(item.Render(80)), "last output")
			require.False(t, expandable.ToggleExpanded())
			require.Equal(t, collapsed, ansi.Strip(item.Render(80)))
		})
	}
}

func TestToolSummaryKeepsErrorsVisible(t *testing.T) {
	t.Parallel()

	sty := styles.CharmtonePantera()
	item := NewToolMessageItem(&sty, "msg", message.ToolCall{
		ID: "tool", Name: "bash", Input: `{"command":"build"}`, Finished: true,
	}, &message.ToolResult{ToolCallID: "tool", IsError: true, Content: "permission denied\nmore detail"}, false, "")
	text := ansi.Strip(item.Render(100))
	require.NotContains(t, text, "\n")
	require.Contains(t, text, "permission denied")
}

func TestToolSummaryTruncatesMultilineCommands(t *testing.T) {
	t.Parallel()

	sty := styles.CharmtonePantera()
	item := NewToolMessageItem(&sty, "msg", message.ToolCall{
		ID: "tool", Name: "bash", Input: `{"command":"echo first\necho ` + strings.Repeat("long", 100) + `"}`, Finished: true,
	}, &message.ToolResult{ToolCallID: "tool", Content: "done"}, false, "")
	text := ansi.Strip(item.Render(40))
	require.NotContains(t, text, "\n")
	require.Contains(t, text, "echo first")
	require.LessOrEqual(t, ansi.StringWidth(text), 40)
}
