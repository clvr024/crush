package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/agent/tools"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/shell"
	"github.com/stretchr/testify/require"
)

func TestBackgroundJobsFollowUpAndCollectResult(t *testing.T) {
	t.Parallel()

	manager := shell.GetBackgroundShellManager()
	job, err := manager.Start(t.Context(), t.TempDir(), nil, "echo done", "Build", config.DefaultMaxBackgroundJobs)
	require.NoError(t, err)
	t.Cleanup(func() { _ = manager.Kill(job.ID) })
	job.Wait()

	tracker := &backgroundJobTracker{jobs: make(map[string]int)}
	meta, err := json.Marshal(tools.BashResponseMetadata{Background: true, AutoBackground: true, ShellID: job.ID})
	require.NoError(t, err)
	tracker.observe(message.ToolResult{Name: tools.BashToolName, Metadata: string(meta)})
	prompt, err := tracker.continuation(t.Context(), time.Second)
	require.NoError(t, err)
	require.Contains(t, prompt, job.ID)
	require.Contains(t, prompt, "finished; result not yet collected")

	meta, err = json.Marshal(tools.JobOutputResponseMetadata{ShellID: job.ID, Done: true})
	require.NoError(t, err)
	tracker.observe(message.ToolResult{Name: tools.JobOutputToolName, Metadata: string(meta), IsError: true})
	require.Empty(t, tracker.reminder())
}

func TestBackgroundJobsIgnoreExplicitBackgroundProcesses(t *testing.T) {
	t.Parallel()

	tracker := &backgroundJobTracker{jobs: make(map[string]int)}
	meta, err := json.Marshal(tools.BashResponseMetadata{Background: true, ShellID: "server"})
	require.NoError(t, err)
	tracker.observe(message.ToolResult{Name: tools.BashToolName, Metadata: string(meta)})
	require.Empty(t, tracker.jobs)
}

func TestBackgroundJobsConcurrentResults(t *testing.T) {
	t.Parallel()

	tracker := &backgroundJobTracker{jobs: make(map[string]int)}
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Go(func() {
			meta, err := json.Marshal(tools.BashResponseMetadata{AutoBackground: true, ShellID: fmt.Sprintf("job-%d", i)})
			if err != nil {
				t.Error(err)
				return
			}
			tracker.observe(message.ToolResult{Name: tools.BashToolName, Metadata: string(meta)})
		})
	}
	wg.Wait()
	require.Len(t, tracker.jobs, 20)
}

func TestBackgroundJobsCancelFollowUp(t *testing.T) {
	t.Parallel()

	manager := shell.GetBackgroundShellManager()
	job, err := manager.Start(t.Context(), t.TempDir(), nil, "sleep 10", "Test", config.DefaultMaxBackgroundJobs)
	require.NoError(t, err)
	t.Cleanup(func() { _ = manager.Kill(job.ID) })
	tracker := &backgroundJobTracker{jobs: map[string]int{job.ID: 0}}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	prompt, err := tracker.continuation(ctx, time.Second)
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, prompt)
}

func TestBackgroundJobsBoundIgnoredReminders(t *testing.T) {
	t.Parallel()

	manager := shell.GetBackgroundShellManager()
	job, err := manager.Start(t.Context(), t.TempDir(), nil, "echo done", "Build", config.DefaultMaxBackgroundJobs)
	require.NoError(t, err)
	t.Cleanup(func() { _ = manager.Kill(job.ID) })
	job.Wait()
	tracker := &backgroundJobTracker{jobs: map[string]int{job.ID: 0}}
	for range 3 {
		prompt, err := tracker.continuation(t.Context(), time.Second)
		require.NoError(t, err)
		require.NotEmpty(t, prompt)
	}
	prompt, err := tracker.continuation(t.Context(), time.Second)
	require.NoError(t, err)
	require.Empty(t, prompt)
}
