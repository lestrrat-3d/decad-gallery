#!/usr/bin/env python3
"""Replay real historical diffs through the planner and a simulated gate."""
import json
import subprocess
from unittest import mock

import gate
import plan


# Expectations describe the changes independently of the planner's output.
CASES = {
    "086f60de7": ({"AGENTS.md", "README.md"}, set(), {"broad", "wheel", "siren"}),
    "44e73e3e5": ({"models/cycloidal/build_test.go"}, {"cycloidal-pr"}, {"broad", "wheel", "siren"}),
    "535722af5": ({"internal/m5fastener/m5fastener.go"}, {"package-m5fastener"}, {"broad", "clamparm"}),
    "b8974b264": ({"models/clamparm/fits_test.go"}, {"clamparm"}, {"broad", "wheel", "siren"}),
    "9f71de90c": ({"go.mod", "go.sum"}, {"broad", "full"}, set()),
    "3e38f7503": ({".github/workflows/repository.yml"}, set(), {"broad"}),
    "b2200f3eb": ({"models/plaquestand/plaquestand.go"}, {"broad", "new-model-plaquestand"}, set()),
}


def validate(result, required_paths, required, excluded):
    paths, suites = set(result["changed_paths"]), set(result["suites"])
    errors = []
    for label, values in (("missing paths", required_paths - paths), ("missing suites", required - suites),
                          ("unexpected suites", excluded & suites)):
        if values:
            errors.append(f"{label}: {', '.join(sorted(values))}")
    if not required and suites:
        errors.append(f"expected empty selection, got {sorted(suites)}")
    return errors


def replay(revision):
    tested = subprocess.check_output(["git", "rev-parse", revision + "^{commit}"], text=True).strip()
    base = subprocess.check_output(["git", "rev-parse", tested + "^"], text=True).strip()
    real_git = plan.git

    def historical_head(*args):
        if args == ("rev-parse", "HEAD"):
            return tested.encode()
        return real_git(*args)

    # Only the checkout identity is simulated; diffs and both metadata snapshots are real.
    with mock.patch.object(plan, "git", side_effect=historical_head):
        result = plan.plan("push", base, tested)
    errors = validate(result, *CASES[revision])
    matrix = gate.matrix_for(result["suites"])
    needs = {job: {"result": "skipped"} for job in gate.BROAD_JOBS}
    if "broad" in result["suites"]:
        full = "full" in result["suites"]
        for job in gate.BROAD_JOBS:
            if (job not in ("renderer-full", "bevel-full", "bevel-pr", *gate.PR_RENDER_JOBS) or
                    (full and job in ("renderer-full", "bevel-full")) or
                    (not full and job == "bevel-pr") or
                    (not full and "renderer-all" not in result["suites"] and job in gate.PR_RENDER_JOBS)):
                needs[job]["result"] = "success"
    needs.update(plan={"result": "success", "outputs": {
        "plan": json.dumps(result), "matrix": json.dumps(matrix), "count": str(len(matrix["include"]))}},
        repository={"result": "success"}, worker={"result": "success" if matrix["include"] else "skipped"},
        **{"records-validation": {"result": "success" if "full" in result["suites"] else "skipped"}})
    errors.extend(gate.check(needs, "push"))
    return result, errors


def main():
    failed = False
    for revision in CASES:
        result, errors = replay(revision)
        print(json.dumps({"revision": revision, "suites": result["suites"], "reason": result["reason"],
                          "errors": errors}, sort_keys=True), flush=True)
        failed |= bool(errors)
    return int(failed)


if __name__ == "__main__":
    raise SystemExit(main())
