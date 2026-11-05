package chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/crush/internal/agent/tools"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/x/ansi"
)

func (t *baseToolMessageItem) renderSummary(width int) string {
	params := partialJSONFields(t.toolCall.Input)
	var description string
	for _, key := range []string{"description", "file_path", "pattern", "command", "query", "url", "path", "shell_id", "symbol", "name", "label", "prompt"} {
		if value, ok := params[key].(string); ok && value != "" {
			description = value
			break
		}
	}

	status := t.computeStatus()
	state := ""
	switch status {
	case ToolStatusRunning:
		state = "running"
	case ToolStatusAwaitingPermission:
		state = "awaiting permission"
	case ToolStatusCanceled:
		state = "canceled"
	}
	if t.result != nil {
		switch t.toolCall.Name {
		case tools.ViewToolName:
			var meta tools.ViewResponseMetadata
			if json.Unmarshal([]byte(t.result.Metadata), &meta) == nil && meta.Content != "" {
				state = fmt.Sprintf("%d lines", strings.Count(meta.Content, "\n")+1)
			}
		case tools.BashToolName:
			var meta tools.BashResponseMetadata
			if json.Unmarshal([]byte(t.result.Metadata), &meta) == nil && meta.Background {
				status = ToolStatusRunning
				state = "background job " + meta.ShellID
			}
		case tools.JobOutputToolName:
			var meta tools.JobOutputResponseMetadata
			if json.Unmarshal([]byte(t.result.Metadata), &meta) == nil {
				if meta.Description != "" {
					description = meta.Description
				}
				state = "completed"
				if !meta.Done {
					status = ToolStatusRunning
					state = fmt.Sprintf("still running · %ds", meta.ElapsedSeconds)
				}
			}
		case tools.TodosToolName:
			var params tools.TodosParams
			if json.Unmarshal([]byte(t.toolCall.Input), &params) == nil {
				completed := 0
				for _, todo := range params.Todos {
					if todo.Status == "completed" {
						completed++
					} else if todo.Status == "in_progress" {
						description = todo.Content
					}
				}
				state = fmt.Sprintf("%d/%d completed", completed, len(params.Todos))
			}
		}
		if t.result.IsError {
			status = ToolStatusError
			state = "failed: " + summaryText(t.result.Content)
			description = state + " · " + description
			state = ""
		}
	}
	description = summaryText(description)
	if state != "" {
		if description != "" {
			description += " · "
		}
		description += state
	}
	if t.isSpinning() {
		description = t.anim.Render() + " " + description
	}

	prefix := "▸ "
	if t.isCompact {
		prefix = ""
	}
	opts := &ToolRenderOpts{ToolCall: t.toolCall, Result: t.result, Compact: t.isCompact}
	headerWidth := max(0, width-ansi.StringWidth(prefix))
	name := prettifyToolName(t.toolCall.Name)
	var header string
	if t.result == nil || !t.result.IsError {
		switch renderer := t.toolRenderer.(type) {
		case *AgentToolRenderContext:
			if renderer.agent.dispatchLabel != "" {
				if t.isSpinning() {
					status = ToolStatusRunning
				}
				header = subagentHeader(t.sty, status, name, renderer.agent.dispatchLabel, dispatchRunModel(opts), headerWidth, opts)
			}
		case *AgentControlToolRenderContext:
			if label, _ := agentControlParts(t.toolCall); label != "" {
				header = subagentHeader(t.sty, status, name, label, sendRunModel(opts), headerWidth, opts)
			}
		}
	}
	if header == "" {
		header = toolHeader(t.sty, status, name, headerWidth, opts, description)
	} else if t.isSpinning() {
		header += " " + t.anim.Render()
	}
	return ansi.Truncate(prefix+header, max(0, width), "…")
}

func summaryText(text string) string {
	return strings.Join(strings.Fields(common.StripCursorControl(ansi.Strip(text))), " ")
}
