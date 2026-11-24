#!/usr/bin/env python3
"""Exercise planner outputs and gate results without running geometry tests."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
from unittest import mock

import gate
import plan
import reconcile_records
import run_tests


FIXTURE_MODULE = "example.com/ci-probe"
FIXTURE_TEST = "TestProbe/child"


def record_fixture(path):
    """Run a small real Go test and record the resulting test events."""
    Path(".tmp").mkdir(exist_ok=True)
    with tempfile.TemporaryDirectory(dir=".tmp") as directory:
        fixture = Path(directory)
        (fixture / "go.mod").write_text(f"module {FIXTURE_MODULE}\n\ngo 1.22\n", encoding="utf-8")
        (fixture / "probe_test.go").write_text(
            'package probe_test\n\nimport "testing"\n\n'
            'func TestProbe(t *testing.T) { t.Run("child", func(t *testing.T) {}) }\n', encoding="utf-8")
        proc = subprocess.Popen(["go", "test", "-json", "-count=1", "./..."], cwd=fixture,
                                stdout=subprocess.PIPE, text=True)
        with proc.stdout, mock.patch.dict(os.environ, {"CI_TEST_RECORDS": str(Path(path).resolve())}):
            run_tests.finished_tests(proc.stdout, qualified=True, module=".")
        if proc.wait() != 0:
            raise RuntimeError("fixture go test failed")


def check_records(reference, candidate):
    result = reconcile_records.reconcile([reference], [candidate],
                                         [(".", FIXTURE_MODULE, FIXTURE_TEST)])
    if not result["ok"]:
        raise ValueError(f"fixture records did not reconcile: {json.dumps(result, sort_keys=True)}")
    print("Probe fixture records reconciled")


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
        "count": str(len(matrix["include"]))}}, repository={"result": "success"},
        **{name: {"result": "skipped"} for name in ("records-validation", "package-plan", "package-workers")})
    if worker is not None:
        needs["worker"] = {"result": worker}
    return needs


def main():
    if os.environ.get("PROBE_MODE") == "record":
        record_fixture(os.environ["CI_TEST_RECORDS"])
        return
    if os.environ.get("PROBE_MODE") == "reconcile":
        check_records(os.environ["REFERENCE_RECORDS"], os.environ["CANDIDATE_RECORDS"])
        return
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
    if actual["reconcile"]["result"] != "success":
        raise SystemExit("fixture records did not reconcile across jobs")
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
