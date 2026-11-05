package tools

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/shell"
)

const (
	JobOutputToolName     = "job_output"
	DefaultJobWaitTimeout = 30
	MaxJobWaitTimeout     = 60
)

//go:embed job_output.md
var jobOutputDescription string

type JobOutputParams struct {
	ShellID string `json:"shell_id" description:"The ID of the background shell to retrieve output from"`
	Wait    bool   `json:"wait" description:"If true, wait up to timeout seconds for completion, then return current output and status"`
	Timeout int    `json:"timeout,omitempty" description:"Maximum seconds to wait when wait=true (default: 30, maximum: 60)"`
}

type JobOutputResponseMetadata struct {
	ShellID          string `json:"shell_id"`
	Command          string `json:"command"`
	Description      string `json:"description"`
	Done             bool   `json:"done"`
	WorkingDirectory string `json:"working_directory"`
	ElapsedSeconds   int64  `json:"elapsed_seconds"`
	OutputBytes      int    `json:"output_bytes"`
	ExitCode         *int   `json:"exit_code,omitempty"`
}

func NewJobOutputTool(spillDir string) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		JobOutputToolName,
		jobOutputDescription,
		func(ctx context.Context, params JobOutputParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.ShellID == "" {
				return fantasy.NewTextErrorResponse("missing shell_id"), nil
			}

			bgManager := shell.GetBackgroundShellManager()
			bgShell, ok := bgManager.Get(params.ShellID)
			if !ok {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("background shell not found: %s", params.ShellID)), nil
			}

			if params.Wait {
				if params.Timeout < 0 || params.Timeout > MaxJobWaitTimeout {
					return fantasy.NewTextErrorResponse("timeout must be between 0 and 60 seconds; 0 selects the default of 30 seconds"), nil
				}
				timeout := params.Timeout
				if timeout == 0 {
					timeout = DefaultJobWaitTimeout
				}
				waitCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
				bgShell.WaitContext(waitCtx)
				cancel()
			}
			if err := ctx.Err(); err != nil {
				return fantasy.ToolResponse{}, err
			}

			stdout, stderr, done, err := bgShell.GetOutput()

			var outputParts []string
			if stdout != "" {
				outputParts = append(outputParts, stdout)
			}
			if stderr != "" {
				outputParts = append(outputParts, stderr)
			}

			status := "running"
			if done {
				status = "completed"
				if err != nil {
					exitCode := shell.ExitCode(err)
					if exitCode != 0 {
						outputParts = append(outputParts, fmt.Sprintf("Exit code %d", exitCode))
					}
				}
			}

			output := strings.Join(outputParts, "\n")
			outputBytes := len(output)
			output = TruncateOutput(output, spillDir)

			metadata := JobOutputResponseMetadata{
				ShellID:          params.ShellID,
				Command:          bgShell.Command,
				Description:      bgShell.Description,
				Done:             done,
				WorkingDirectory: bgShell.WorkingDir,
				ElapsedSeconds:   int64(time.Since(bgShell.StartedAt).Seconds()),
				OutputBytes:      outputBytes,
			}
			if done {
				exitCode := shell.ExitCode(err)
				metadata.ExitCode = &exitCode
			}

			if output == "" {
				output = BashNoOutput
			}

			result := fmt.Sprintf("Status: %s\nElapsed: %ds\n\n%s", status, metadata.ElapsedSeconds, output)
			if !done {
				result += "\n\nThe command is still running. Report meaningful progress to the user, then use job_output with wait=true and timeout=30 to collect its result. Do not treat a wait timeout as command failure or claim completion before checking the exit status."
			}
			resp := fantasy.NewTextResponse(result)
			if done && err != nil {
				resp = fantasy.NewTextErrorResponse(result)
			}
			return fantasy.WithResponseMetadata(resp, metadata), nil
		},
	)
}
