package tools

import (
	"encoding/json"
	"testing"
	"time"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/shell"
	"github.com/stretchr/testify/require"
)

func TestJobOutputBoundedWaitLeavesJobRunning(t *testing.T) {
	t.Parallel()

	manager := shell.GetBackgroundShellManager()
	job, err := manager.Start(t.Context(), t.TempDir(), nil, "echo started; sleep 10", "Build")
	require.NoError(t, err)
	t.Cleanup(func() { _ = manager.Kill(job.ID) })
	input, err := json.Marshal(JobOutputParams{ShellID: job.ID, Wait: true, Timeout: 1})
	require.NoError(t, err)
	started := time.Now()
	resp, err := NewJobOutputTool(t.TempDir()).Run(t.Context(), fantasy.ToolCall{Name: JobOutputToolName, Input: string(input)})
	require.NoError(t, err)
	require.False(t, resp.IsError)
	require.Less(t, time.Since(started), 5*time.Second)
	require.Contains(t, resp.Content, "Status: running")
	require.False(t, job.IsDone())

	var meta JobOutputResponseMetadata
	require.NoError(t, json.Unmarshal([]byte(resp.Metadata), &meta))
	require.False(t, meta.Done)
	require.Nil(t, meta.ExitCode)
}

func TestJobOutputMarksCommandFailure(t *testing.T) {
	t.Parallel()

	manager := shell.GetBackgroundShellManager()
	job, err := manager.Start(t.Context(), t.TempDir(), nil, "echo failed; exit 42", "Test")
	require.NoError(t, err)
	t.Cleanup(func() { _ = manager.Kill(job.ID) })
	job.Wait()
	input, err := json.Marshal(JobOutputParams{ShellID: job.ID, Wait: true})
	require.NoError(t, err)
	resp, err := NewJobOutputTool(t.TempDir()).Run(t.Context(), fantasy.ToolCall{Name: JobOutputToolName, Input: string(input)})
	require.NoError(t, err)
	require.True(t, resp.IsError)
	require.Contains(t, resp.Content, "Exit code 42")

	var meta JobOutputResponseMetadata
	require.NoError(t, json.Unmarshal([]byte(resp.Metadata), &meta))
	require.True(t, meta.Done)
	require.NotNil(t, meta.ExitCode)
	require.Equal(t, 42, *meta.ExitCode)
}
