#!/usr/bin/env python3
"""Expand the pilot plan and reject missing or unsuccessful CI jobs."""
import json
import os
from pathlib import Path
import re
import subprocess
import sys

from run_tests import finished_tests, run_pattern


RUNNER = ["python3", "scripts/ci/run_tests.py"]
RENDER_TESTS = {
    "clamparm-render": ("40m", ("TestClampArmPresets", "TestClampArmExploded")),
    "swiss-render": ("40m", ("TestSwissPresets", "TestSwissExploded", "TestSwissMovers")),
    "siren-render": ("40m", ("TestHandCrankSirenView",)),
    "fan-view": ("40m", ("TestHandCrankFanView",)),
    "planetary-render": ("40m", ("TestPlanetaryMotionUsesBuiltGearbox", "TestPlanetaryKitParts")),
    "wheel-block": ("40m", ("TestSpinningWheelBlockOfEveryPart",)),
    "bevel-render": ("40m", ("TestBevelPairsRealBuild",)),
    "bearing-render": ("40m", ("TestDirectRenderMatchesPNGForClippedBearing",)),
    "planetary-motion": ("10m", ("TestPlanetaryGearboxMotion",)),
    "wheel-motion": ("30m", ("TestSpinningWheelMotion",)),
    "wheel-parts": ("40m", ("TestSpinningWheelPartsTable",)),
    "fan-motion": ("20m", ("TestHandCrankFanMotion",)),
    "escapement-exploded": ("15m", ("TestEscapementExploded",)),
}
WORKERS = {
    "cycloidal-pr": [RUNNER + ["cycloidal", "pr"]],
    "renderer-compile": [["go", "-C", "tools/render", "test", "-run", "^$", "./..."]],
    "renderer-all": [["go", "-C", "tools/render", "test", "-timeout", "170m", "./..."]],
    "feature-dovetail": [["go", "test", "./features/dovetail", "-timeout", "5m"]],
    **{name: [["go", "-C", "tools/render", "test", "-json", "-run", run_pattern(tests),
               "-timeout", timeout, "./..."]] for name, (timeout, tests) in RENDER_TESTS.items()},
    "wheel": [RUNNER + ["wheel", case] for case in (
        "default-verify", "default-linkage", "native", "small-seven-verify", "small-seven-linkage",
        "large-nine-verify", "large-nine-linkage", "twelve", "small-seventeen", "rest")],
    "siren": [RUNNER + ["siren", case] for case in ("default", "spike", "slots12", "slots24", "rest")],
    "bevel-pr": [RUNNER + ["bevel", "pr", case] for case in
                 ("abgh", "cdef", "rest-0", "rest-1", "rest-2", "rest-3")],
    **{name: [RUNNER + [name, "all"]] for name in ("clamparm", "escapement", "onewayclutch", "turbinefan")},
}
BROAD_JOBS = ("planetary-motion", "wheel-motion", "fan-motion", "escapement", "onewayclutch",
              "clamparm", "turbinefan", "escapement-exploded", "wheel-parts", "renderer-full", "wheel", "siren",
              "bevel-pr", "bevel-full")


def commands(suite):
    if suite in WORKERS:
        return WORKERS[suite]
    for prefix, directory in (("new-model-", "models"), ("new-package-", "internal"),
                              ("new-feature-", "features"),
                              ("model-", "models"), ("package-", "internal")):
        if suite.startswith(prefix):
            name = suite[len(prefix):]
            if re.fullmatch(r"[a-z][a-z0-9]*", name) and (Path(directory) / name).is_dir():
                if prefix == "model-":
                    return [RUNNER + ["model", "pr", name]]
                timeout = "60m" if prefix == "new-model-" else "5m"
                return [["go", "test", f"./{directory}/{name}", "-timeout", timeout]]
    raise ValueError(f"unregistered suite: {suite}")


def matrix_for(suites):
    return {"include": [{"suite": suite, "case": index}
                        for suite in suites if suite != "broad"
                        for index in range(len(commands(suite)))]}


def run_renderer(suite, command):
    proc = subprocess.Popen(command, stdout=subprocess.PIPE, text=True)
    finished = finished_tests(proc.stdout, qualified=True)
    if proc.wait() != 0:
        raise SystemExit(proc.returncode)
    package = "github.com/lestrrat-3d/decad-gallery/tools/render"
    expected = RENDER_TESTS[suite][1]
    missing = {test for test in expected if finished.get((package, test)) != "pass"}
    missing.update(test for (pkg, test), status in finished.items()
                   if pkg == package and status != "pass" and
                   any(test.startswith(parent + "/") for parent in expected))
    if missing:
        raise SystemExit("renderer tests did not pass: " + ", ".join(sorted(missing)[:20]))


def suites_from_plan(raw):
    plan = json.loads(raw)
    suites = plan["suites"]
    if (plan.get("version") != 1 or not isinstance(suites, list)
            or any(not isinstance(suite, str) for suite in suites)
            or len(set(suites)) != len(suites)
            or ("broad" in suites and any(suite not in ("broad", "renderer-all") and not suite.startswith(
                ("new-model-", "new-package-", "new-feature-")) for suite in suites))):
        raise ValueError("invalid suite selection")
    matrix_for(suites)
    return suites


def check(needs, event):
    errors = []
    for job in ("plan", "repository"):
        result = needs.get(job, {}).get("result", "missing")
        if result != "success":
            errors.append(f"{job}: expected success, got {result}")
    try:
        outputs = needs["plan"]["outputs"]
        suites = suites_from_plan(outputs["plan"])
        expected_matrix = matrix_for(suites)
        if json.loads(outputs["matrix"]) != expected_matrix or outputs["count"] != str(len(expected_matrix["include"])):
            raise ValueError("matrix does not match plan")
        if event in ("schedule", "workflow_dispatch") and "broad" not in suites:
            raise ValueError("full event did not select broad coverage")
    except (KeyError, TypeError, ValueError, json.JSONDecodeError) as exc:
        return errors + [f"planner did not report a valid selection: {exc}"]

    expected = {"worker": "success" if expected_matrix["include"] else "skipped"}
    for job in BROAD_JOBS:
        enabled = "broad" in suites
        if job == "bevel-pr":
            enabled &= event in ("pull_request", "push")
        if job in ("bevel-full", "renderer-full"):
            enabled &= event in ("schedule", "workflow_dispatch")
        expected[job] = "success" if enabled else "skipped"
    for job, status in expected.items():
        result = needs.get(job, {}).get("result", "missing")
        if result != status:
            errors.append(f"{job}: expected {status}, got {result}")
    return errors


def main():
    if len(sys.argv) > 1 and sys.argv[1] == "matrix":
        suites = suites_from_plan(os.environ["PLAN_JSON"])
        matrix = matrix_for(suites)
        with open(os.environ["GITHUB_OUTPUT"], "a", encoding="utf-8") as output:
            print("matrix=" + json.dumps(matrix, separators=(",", ":")), file=output)
            print("count=" + str(len(matrix["include"])), file=output)
    elif len(sys.argv) > 1 and sys.argv[1] == "run":
        suite = os.environ["SUITE"]
        command = commands(suite)[int(os.environ["CASE"])]
        if suite in RENDER_TESTS:
            run_renderer(suite, command)
        else:
            subprocess.run(command, check=True)
    else:
        errors = check(json.loads(os.environ["NEEDS_JSON"]), os.environ["EVENT_NAME"])
        if errors:
            sys.exit("\n".join(errors))
        print("All planned jobs passed")


if __name__ == "__main__":
    main()
