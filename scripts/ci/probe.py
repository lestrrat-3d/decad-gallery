#!/usr/bin/env python3
"""Exercise planner outputs and gate results without running geometry tests."""
import json
import os
import subprocess
from unittest import mock

import gate
import plan


def selections():
    head = plan.git("rev-parse", "HEAD").decode()
    empty = plan.plan("push", head, head)
    original = subprocess.check_output

    def changed(command, **kwargs):
        if command[:3] == ["git", "diff", "--no-renames"]:
            return b"models/spinningwheel/images/parts/probe.png\0"
        return original(command, **kwargs)

    with mock.patch.object(plan.subprocess, "check_output", side_effect=changed):
        selected = plan.plan("push", head, head)
    if empty["suites"] != [] or selected["suites"] != ["wheel-parts"]:
        raise ValueError("planner did not produce expected probe selections")
    return empty, selected


def needs_for(selection, worker):
    matrix = gate.matrix_for(selection["suites"])
    needs = {job: {"result": "skipped"} for job in gate.BROAD_JOBS}
    needs.update(plan={"result": "success", "outputs": {
        "plan": json.dumps(selection), "matrix": json.dumps(matrix),
        "count": str(len(matrix["include"]))}}, repository={"result": "success"})
    if worker is not None:
        needs["worker"] = {"result": worker}
    return needs


def main():
    if os.environ.get("PROBE_MODE") == "plan":
        empty, selected = selections()
        with open(os.environ["GITHUB_OUTPUT"], "a", encoding="utf-8") as output:
            for name, selection in (("empty", empty), ("selected", selected)):
                print(f"{name}=" + json.dumps(selection, separators=(",", ":")), file=output)
                matrix = gate.matrix_for(selection["suites"])
                print(f"{name}_matrix=" + json.dumps(matrix, separators=(",", ":")), file=output)
                print(f"{name}_count=" + str(len(matrix["include"])), file=output)
        return
    actual = json.loads(os.environ["NEEDS_JSON"])
    empty = json.loads(os.environ["EMPTY_PLAN"])
    selected = json.loads(os.environ["SELECTED_PLAN"])
    cases = ((empty, actual["empty-worker"]["result"], False),
             (selected, actual["selected-worker"]["result"], False),
             (selected, actual["failed-worker"]["result"], True),
             (selected, "cancelled", True), (selected, None, True))
    for selection, worker, should_fail in cases:
        errors = gate.check(needs_for(selection, worker), "push")
        if bool(errors) != should_fail or (should_fail and not any("worker:" in error for error in errors)):
            raise SystemExit(f"unexpected gate result for {worker}: {errors}")
    print("Probe gate cases passed")


if __name__ == "__main__":
    main()
