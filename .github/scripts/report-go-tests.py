"""Show Go test progress and bounded failures while retaining the full log."""

import collections
import html
import json
import os
import sys


def main():
    tails = collections.defaultdict(lambda: collections.deque(maxlen=60))
    failures = []
    with open(sys.argv[1], "w", encoding="utf-8") as raw:
        for line in sys.stdin:
            raw.write(line)
            try:
                event = json.loads(line)
            except json.JSONDecodeError:
                print(line, end="", flush=True)
                continue

            package = event.get("Package", event.get("ImportPath", "go test"))
            test = event.get("Test", "")
            key = (package, test)
            action = event.get("Action")
            if action in ("output", "build-output"):
                tails[key].extend(event.get("Output", "").splitlines())
            elif action == "start":
                print(f"Testing {package}", flush=True)
            elif action in ("fail", "build-fail"):
                name = f"{package}/{test}" if test else package
                detail = "\n".join(tails.pop(key, []))
                failures.append((name, detail))
                print(f"\nFAIL: {name}\n{detail}\n", flush=True)
            elif action in ("pass", "skip"):
                tails.pop(key, None)
                if not test:
                    print(f"{action.upper()}: {package} ({event.get('Elapsed', 0)}s)", flush=True)

    summary = os.environ.get("GITHUB_STEP_SUMMARY")
    if summary and failures:
        with open(summary, "a", encoding="utf-8") as output:
            output.write("## Failed Go tests\n\n")
            for name, detail in failures:
                output.write(f"### {name}\n\n<pre>\n")
                output.write(html.escape(detail[-12000:]))
                output.write("\n</pre>\n\n")
            output.write("Full JSON log: the `go-test-log` artifact on this run.\n")


if __name__ == "__main__":
    main()
