#!/usr/bin/env python3
"""Check the declarations in test files that opt into individual selection."""

from pathlib import Path
import json
import os
import subprocess
import tempfile


def main():
    roots = ("models", "internal", "features", "tools/render")
    files = [path for root in roots for path in Path(root).rglob("*_test.go")
             if "// CI: isolated-tests" in path.read_text(encoding="utf-8")]
    plan = json.loads(os.environ.get("PLAN_JSON", "{}"))
    if os.environ.get("GITHUB_EVENT_NAME") in {"push", "pull_request"} and plan.get("changed_paths"):
        diff = subprocess.check_output(["git", "diff", "--no-renames", "--diff-filter=A", "--name-only", "-z",
                                        plan["base"], plan["tested"]])
        files.extend(Path(os.fsdecode(path)) for path in diff.split(b"\0") if path and
                     any(os.fsdecode(path).startswith(root + "/") for root in roots)
                     and os.fsdecode(path).endswith("_test.go"))
    Path(".tmp").mkdir(exist_ok=True)
    with tempfile.TemporaryDirectory(dir=".tmp") as directory:
        binary = str(Path(directory) / "testdiff")
        subprocess.run(["go", "build", "-o", binary, "scripts/ci/testdiff.go"], check=True)
        for path in set(files):
            subprocess.run([binary, "--layout", str(path)], check=True, stdout=subprocess.DEVNULL)


if __name__ == "__main__":
    main()
