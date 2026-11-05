package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/crush/internal/agent/tools"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/shell"
)

type backgroundJobTracker struct {
	mu   sync.Mutex
	jobs map[string]int
}

func (b *backgroundJobTracker) observe(result message.ToolResult) {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch result.Name {
	case tools.BashToolName:
		var meta tools.BashResponseMetadata
		if !result.IsError && json.Unmarshal([]byte(result.Metadata), &meta) == nil && meta.AutoBackground && meta.ShellID != "" {
			b.jobs[meta.ShellID] = 0
		}
	case tools.JobOutputToolName:
		var meta tools.JobOutputResponseMetadata
		if json.Unmarshal([]byte(result.Metadata), &meta) == nil && meta.ShellID != "" {
			if meta.Done {
				delete(b.jobs, meta.ShellID)
			} else if _, ok := b.jobs[meta.ShellID]; ok {
				b.jobs[meta.ShellID] = 0
			}
		}
	case tools.JobKillToolName:
		var meta tools.JobKillResponseMetadata
		if !result.IsError && json.Unmarshal([]byte(result.Metadata), &meta) == nil {
			delete(b.jobs, meta.ShellID)
		}
	}
}

func (b *backgroundJobTracker) reminder() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	var ids []string
	for id, reminders := range b.jobs {
		if reminders < 3 {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	if len(ids) == 0 {
		return ""
	}
	var statuses []string
	for _, id := range ids {
		job, ok := shell.GetBackgroundShellManager().Get(id)
		if !ok {
			delete(b.jobs, id)
			continue
		}
		state := "still running"
		if job.IsDone() {
			state = "finished; result not yet collected"
		}
		statuses = append(statuses, fmt.Sprintf("Job %s: %s", id, state))
	}
	if len(statuses) == 0 {
		return ""
	}
	return "<system_reminder>These commands automatically moved to the background during the current task:\n" +
		strings.Join(statuses, "\n") +
		"\nTell the user any important finding or waiting status, then use job_output to inspect these jobs. For running jobs use wait=true and timeout=30; for finished jobs collect the output and exit status now. Continue the original task after checking the results. Do not claim builds or tests passed before collecting their results. If you cannot inspect a job, clearly report that limitation instead of silently waiting or claiming success.</system_reminder>"
}

func (b *backgroundJobTracker) continuation(ctx context.Context, timeout time.Duration) (string, error) {
	if b.reminder() == "" {
		return "", nil
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	completed := make(chan struct{}, 1)
	b.mu.Lock()
	for id, reminders := range b.jobs {
		if reminders >= 3 {
			continue
		}
		job, ok := shell.GetBackgroundShellManager().Get(id)
		if !ok {
			continue
		}
		go func() {
			if job.WaitContext(waitCtx) {
				select {
				case completed <- struct{}{}:
				default:
				}
			}
		}()
	}
	b.mu.Unlock()
	select {
	case <-completed:
	case <-waitCtx.Done():
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	prompt := b.reminder()
	b.mu.Lock()
	for id := range b.jobs {
		b.jobs[id]++
	}
	b.mu.Unlock()
	return prompt, nil
}
