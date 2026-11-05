`TestCoderAgent` replays the recorded HTTP interactions in this directory.
Requests are matched strictly, including the system prompt, tool descriptions,
parameter schemas, user messages, tool calls, and tool results. Session titles
use a fixed test model; title generation is covered by `title_lifetime_test.go`.

The fork's reading and progress changes were applied offline to the request
bodies in `TestCoderAgent/deepseek-v4/`: the coder prompt, bash and view
descriptions, view limit documentation, and the end-of-file marker in view
results. Request content lengths were updated. Recorded responses were kept
unchanged. These cassettes exercise agent and tool execution with a fixed model
response; they do not measure how a live model responds to the new prompt.

When prompts, schemas, or tool output change again, review the request diff and
update the corresponding fixtures, or re-record with `task test:record` using
provider credentials. Do not loosen request matching or skip these tests to
accommodate a mismatch.
