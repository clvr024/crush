package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/charmbracelet/crush/internal/shell"
	"github.com/stretchr/testify/require"
)

func TestBashToolCompletedCommandStatus(t *testing.T) {
	t.Parallel()

	for _, background := range []bool{false, true} {
		mode := "foreground"
		if background {
			mode = "background"
		}
		for _, tc := range []struct {
			name    string
			command string
			content string
			isError bool
		}{
			{"success", "echo done", "done", false},
			{"no_output", "true", BashNoOutput, false},
			{"stderr_success", "echo warning >&2", "warning", false},
			{"failure", "exit 7", "Exit code 7", true},
			{"failure_with_output", "echo before-failure; exit 7", "before-failure", true},
		} {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				t.Parallel()

				tool := newBashToolForTest(t.TempDir())
				ctx := context.WithValue(t.Context(), SessionIDContextKey, "test-session")
				resp := runBashTool(t, tool, ctx, BashParams{
					Command:         tc.command,
					RunInBackground: background,
				})
				require.Equal(t, tc.isError, resp.IsError)
				require.Contains(t, resp.Content, tc.content)
				if tc.isError {
					require.Contains(t, resp.Content, "Exit code 7")
				}

				var meta BashResponseMetadata
				require.NoError(t, json.Unmarshal([]byte(resp.Metadata), &meta))
				require.False(t, meta.Background, "a completed command is no longer a background job")
				require.False(t, meta.AutoBackground)
				require.Empty(t, meta.ShellID)
			})
		}
	}
}

func TestBashToolExplicitBackgroundKeepsRunningJob(t *testing.T) {
	t.Parallel()

	tool := newBashToolForTest(t.TempDir())
	ctx := context.WithValue(t.Context(), SessionIDContextKey, "test-session")
	resp := runBashTool(t, tool, ctx, BashParams{
		Command:         "sleep 10",
		RunInBackground: true,
	})
	var meta BashResponseMetadata
	require.NoError(t, json.Unmarshal([]byte(resp.Metadata), &meta))
	manager := shell.GetBackgroundShellManager()
	t.Cleanup(func() { _ = manager.Kill(meta.ShellID) })

	require.False(t, resp.IsError)
	require.True(t, meta.Background)
	require.False(t, meta.AutoBackground)
	require.NotEmpty(t, meta.ShellID)
	require.Contains(t, resp.Content, meta.ShellID)
	job, ok := manager.Get(meta.ShellID)
	require.True(t, ok)
	require.False(t, job.IsDone())
}
