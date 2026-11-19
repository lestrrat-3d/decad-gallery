#!/usr/bin/env python3
"""Expand the pilot plan and reject missing or unsuccessful CI jobs."""
import json
import os
from pathlib import Path
import re
import subprocess
import sys

from run_tests import SEVENSEGMENT_SHARDS, FAN_PR_SHARDS, PLANETARY_PR_SHARDS, finished_tests, go_test, renderer_packages, run_pattern


RUNNER = ["python3", "scripts/ci/run_tests.py"]
MODULE = "github.com/lestrrat-3d/decad-gallery"
RENDER_PR_SHARDS = 16
RENDER_TOOL_SHARDS = 16
# Tests that build a full assembly. Each runs as its own job in the item's render suite; the rest share one job.
RENDER_HEAVY = {
    "bevelgear": ("TestBevelPairsRealBuild",),
    "clamparm": ("TestClampArmPresets", "TestClampArmExploded"),
    "escapement": ("TestEscapementExploded",),
    "handcrankfan": ("TestHandCrankFanMotion", "TestHandCrankFanView"),
    "handcranksiren": ("TestHandCrankSirenView",),
    "planetarygearbox": ("TestPlanetaryGearboxMotion", "TestPlanetaryKitParts",
                         "TestPlanetaryMotionUsesBuiltGearbox"),
    "spinningwheel": ("TestSpinningWheelMotion", "TestSpinningWheelPartsTable", "TestSpinningWheelBlockOfEveryPart"),
    "swisslever": ("TestSwissExploded",),
}
RENDER_TESTS = {
    "clamparm-render": ("40m", "tools/render", ("TestClampArmPresets", "TestClampArmExploded")),
    "swiss-render": ("40m", "tools/render", ("TestSwissPresets", "TestSwissExploded", "TestSwissMovers")),
    "siren-render": ("40m", "tools/render", ("TestHandCrankSirenView",)),
    "fan-view": ("40m", "tools/render", ("TestHandCrankFanView",)),
    "planetary-render": ("40m", "tools/render", ("TestPlanetaryMotionUsesBuiltGearbox", "TestPlanetaryKitParts")),
    "wheel-block": ("40m", "tools/render", ("TestSpinningWheelBlockOfEveryPart",)),
    "bevel-render": ("40m", "tools/render", ("TestBevelPairsRealBuild",)),
    "bearing-render": ("40m", "tools/render", ("TestDirectRenderMatchesPNGForClippedBearing",)),
    "planetary-motion": ("10m", "tools/render", ("TestPlanetaryGearboxMotion",)),
    "wheel-motion": ("30m", "tools/render", ("TestSpinningWheelMotion",)),
    "wheel-parts": ("40m", "tools/render", ("TestSpinningWheelPartsTable",)),
    "fan-motion": ("20m", "tools/render", ("TestHandCrankFanMotion",)),
    "escapement-exploded": ("15m", "tools/render", ("TestEscapementExploded",)),
    "sevensegment-render": ("30m", "tools/render", ("TestSevenSegmentPresets", "TestSevenSegmentColors",
                                    "TestSevenSegmentMotionColors")),
}
WORKERS = {
    "cycloidal-pr": [RUNNER + ["cycloidal", "pr"]],
    "model-handcrankfan": [RUNNER + ["fan", "pr", str(i)] for i in range(FAN_PR_SHARDS)],
    "model-planetarygearbox": [RUNNER + ["planetary", "pr", str(i)] for i in range(PLANETARY_PR_SHARDS)],
    "renderer-compile": [["renderer-compile"]],
    "renderer-all": [["renderer-pr", str(i)] for i in range(RENDER_PR_SHARDS)],
    "feature-dovetail": [["go", "test", "./features/dovetail", "-timeout", "5m"]],
    **{name: [["go", "test", "-json", "-run", run_pattern(tests),
               "-timeout", timeout, "./" + directory]] for name, (timeout, directory, tests) in RENDER_TESTS.items()},
    "wheel": [RUNNER + ["wheel", case] for case in (
        "default-verify", "default-linkage", "native", "small-seven-verify", "small-seven-linkage",
        "large-nine-verify", "large-nine-linkage", "twelve", "small-seventeen", "rest")],
    "siren": [RUNNER + ["siren", case] for case in ("default", "spike", "slots12", "slots24", "rest")],
    "bevel-pr": [RUNNER + ["bevel", "pr", case] for case in
                 ("abgh", "cdef", "rest-0", "rest-1", "rest-2", "rest-3")],
    **{name: [RUNNER + [name, "all"]] for name in ("clamparm", "escapement", "onewayclutch", "turbinefan")},
    "sevensegment": [RUNNER + ["sevensegment", shard] for shard in SEVENSEGMENT_SHARDS],
    # A new model package would run in one job; the model's own shards split it, as for the targeted suite.
    "new-model-sevensegment": [RUNNER + ["sevensegment", shard] for shard in SEVENSEGMENT_SHARDS],
}
BROAD_JOBS = ("planetary-motion", "wheel-motion", "fan-motion", "escapement", "onewayclutch",
              "clamparm", "sevensegment", "sevensegment-render", "turbinefan",
              "escapement-exploded", "wheel-parts", "renderer-full", "wheel", "siren",
              "bevel-pr", "bevel-full", "planetary-full")
PR_RENDER_JOBS = {"planetary-motion", "wheel-motion", "wheel-parts", "fan-motion", "escapement-exploded"}


def isolated_suite(suite):
    match = re.fullmatch(r"isolated-(?:(tag-coplanar_test)-)?(?:(internal|feature|modelrender|featurerender)-)?"
                         r"([a-z][a-z0-9]*)-(Test[A-Za-z0-9_]*)", suite)
    if not match:
        return None
    tag, kind, name, test = match.groups()
    directory = {None: "models", "internal": "internal", "feature": "features",
                 "modelrender": "models", "featurerender": "features"}[kind]
    if kind in ("modelrender", "featurerender"):
        name += "/render"
    if kind is None and name == "render":
        directory, name = "tools", "render"
    if not (Path(directory) / name).is_dir():
        return None
    if tag and (directory, name) != ("models", "spinningwheel"):
        return None
    return directory, name, test, "coplanar_test" if tag else None


def render_directory(name):
    """Return the render package directory of an item, preferring models over features."""
    for kind in ("models", "features"):
        directory = f"{kind}/{name}/render"
        if Path(directory).is_dir():
            return directory
    raise ValueError(f"unregistered suite: render-{name}")


def commands(suite):
    if suite == "render-tool":
        return [["render-tool", str(index)] for index in range(RENDER_TOOL_SHARDS)]
    match = re.fullmatch(r"render-([a-z][a-z0-9]*)", suite)
    if match and suite not in WORKERS:
        name = match.group(1)
        directory = render_directory(name)
        return ([["render-package", directory, test] for test in RENDER_HEAVY.get(name, ())] +
                [["render-package", directory, "rest"]])
    isolated = isolated_suite(suite)
    if isolated:
        directory, name, test, tag = isolated
        package = "./" + directory + "/" + name
        timeout = "40m" if directory == "tools" or directory == "models" and (
            (name == "planetarygearbox" and test == "TestKitGearboxes") or
            (name == "handcrankfan" and test in ("TestHandCrankFanDefault", "TestHandCrankFanVariant"))
        ) else "5m"
        return [["go", "test", package, "-run", "^" + test + "$", "-timeout", timeout, "-count=1",
                 *(["-tags", tag] if tag else [])]]
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
                        for suite in suites if suite not in ("broad", "full")
                        for index in range(len(commands(suite)))]}


def run_renderer(suite, command):
    proc = subprocess.Popen(command, stdout=subprocess.PIPE, text=True)
    finished = finished_tests(proc.stdout, qualified=True, module="tools/render")
    if proc.wait() != 0:
        raise SystemExit(proc.returncode)
    package = MODULE + "/" + RENDER_TESTS[suite][1]
    expected = RENDER_TESTS[suite][2]
    missing = {test for test in expected if finished.get((package, test)) != "pass"}
    missing.update(test for (pkg, test), status in finished.items()
                   if pkg == package and status != "pass" and
                   any(test.startswith(parent + "/") for parent in expected))
    if missing:
        raise SystemExit("renderer tests did not pass: " + ", ".join(sorted(missing)[:20]))


def run_renderer_full():
    """Discover every renderer package and require each listed test to pass."""
    packages = renderer_packages()
    if not packages or len(packages) != len(set(packages)):
        raise SystemExit("renderer package inventory is empty or duplicated")
    expected = set()
    for package in packages:
        output = subprocess.run(["go", "test", "-list", "^Test", package],
                                check=True, capture_output=True, text=True).stdout
        expected.update((package, line) for line in output.splitlines() if line.startswith("Test"))
    if not expected:
        raise SystemExit("renderer test inventory is empty")
    proc = subprocess.Popen(["go", "test", "-json", "-timeout", "170m", *packages],
                            stdout=subprocess.PIPE, text=True)
    finished = finished_tests(proc.stdout, qualified=True, module="tools/render")
    if proc.wait() != 0:
        raise SystemExit(proc.returncode)
    missing = {identity for identity in expected if finished.get(identity) != "pass"}
    missing.update((package, test) for (package, test), status in finished.items()
                   if status != "pass" and (package, test.split("/", 1)[0]) in expected)
    if missing:
        names = (f"tools/render:{package}:{test}" for package, test in sorted(missing)[:20])
        raise SystemExit("renderer tests did not pass: " + ", ".join(names))


def run_render_case(directory, which):
    """Run one heavy test of an item's render package, or all its other tests when which is "rest"."""
    package = MODULE + "/" + directory
    name = directory.split("/")[1]
    heavy = RENDER_HEAVY.get(name, ())
    if which != "rest":
        if which not in heavy:
            raise SystemExit(f"{which} is not a heavy test of {directory}")
        go_test([f"./{directory}", "-run", f"^{which}$", "-timeout", "40m", "-count=1"], [which],
                package=package, module="tools/render")
        return
    listing = subprocess.run(["go", "test", "-list", "^Test", f"./{directory}"],
                             check=True, capture_output=True, text=True).stdout
    tests = [line for line in listing.splitlines() if line.startswith("Test") and line not in heavy]
    if not tests:
        print(f"{directory} has no tests outside the heavy cases")
        return
    go_test([f"./{directory}", "-run", run_pattern(tests), "-timeout", "40m", "-count=1"], tests,
            package=package, module="tools/render")


def run_renderer_pr(shard, packages=None, shards=RENDER_PR_SHARDS):
    """Run a disjoint share of the discovered tests of packages (default: every renderer package)."""
    if not shard.isdigit() or not 0 <= int(shard) < shards:
        raise SystemExit("invalid renderer PR shard")
    packages = renderer_packages() if packages is None else packages
    if not packages or len(packages) != len(set(packages)):
        raise SystemExit("renderer package inventory is empty or duplicated")
    inventory = []
    for package in packages:
        output = subprocess.run(["go", "test", "-list", "^Test", package],
                                check=True, capture_output=True, text=True).stdout
        inventory.extend((package, line) for line in output.splitlines() if line.startswith("Test"))
    if not inventory or len(inventory) != len(set(inventory)):
        raise SystemExit("renderer test inventory is empty or duplicated")
    selected = sorted(inventory)[int(shard)::shards]
    for package in sorted({item[0] for item in selected}):
        tests = [test for pkg, test in selected if pkg == package]
        go_test([package, "-run", run_pattern(tests), "-timeout", "40m", "-count=1"], tests,
                package=package, module="tools/render")


def suites_from_plan(raw):
    plan = json.loads(raw)
    suites = plan["suites"]
    if (plan.get("version") != 1 or not isinstance(suites, list)
            or any(not isinstance(suite, str) for suite in suites)
            or len(set(suites)) != len(suites)
            or ("full" in suites and "broad" not in suites)):
        raise ValueError("invalid suite selection")
    if "reused" in plan:
        proof = plan["reused"]
        if (suites or not isinstance(proof, dict) or
                type(proof.get("pull_request")) is not int or proof["pull_request"] <= 0 or
                type(proof.get("run")) is not int or proof["run"] <= 0 or
                not isinstance(proof.get("tested"), str) or not re.fullmatch(r"[0-9a-f]{40}", proof["tested"]) or
                not isinstance(proof.get("suites"), list) or not proof["suites"] or
                any(not isinstance(suite, str) for suite in proof["suites"]) or
                len(set(proof["suites"])) != len(proof["suites"]) or
                {"broad", "full"}.intersection(proof["suites"])):
            raise ValueError("invalid reused pull request proof")
        matrix_for(proof["suites"])
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
        if "reused" in json.loads(outputs["plan"]) and event != "push":
            raise ValueError("pull request reuse is only valid for a push")
        expected_matrix = matrix_for(suites)
        if json.loads(outputs["matrix"]) != expected_matrix or outputs["count"] != str(len(expected_matrix["include"])):
            raise ValueError("matrix does not match plan")
        if event in ("schedule", "workflow_dispatch") and "broad" not in suites:
            raise ValueError("full event did not select broad coverage")
    except (KeyError, TypeError, ValueError, json.JSONDecodeError) as exc:
        return errors + [f"planner did not report a valid selection: {exc}"]

    expected = {"worker": "success" if expected_matrix["include"] else "skipped"}
    full = "full" in suites or event in ("schedule", "workflow_dispatch")
    expected["records-validation"] = "success" if full else "skipped"
    for job in BROAD_JOBS:
        enabled = "broad" in suites
        if job == "bevel-pr":
            enabled &= not full
        if job in PR_RENDER_JOBS:
            enabled &= not full and "renderer-all" not in suites
        if job in ("bevel-full", "renderer-full", "planetary-full"):
            enabled &= full
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
        if suite == "renderer-all":
            run_renderer_pr(os.environ["CASE"])
        elif suite == "render-tool":
            run_renderer_pr(os.environ["CASE"], [MODULE + "/tools/render"], RENDER_TOOL_SHARDS)
        elif command[0] == "render-package":
            run_render_case(command[1], command[2])
        elif suite == "renderer-compile":
            subprocess.run(["go", "test", "-run", "^$", *renderer_packages()], check=True)
        elif suite.startswith("isolated-"):
            directory, name, test, _ = isolated_suite(suite)
            package = "github.com/lestrrat-3d/decad-gallery/" + directory + "/" + name
            allowed = frozenset()
            if directory == "models" and name == "handcrankfan" and test in ("TestHandCrankFanDefault", "TestHandCrankFanVariant") \
                    and not os.environ.get("HANDCRANKFAN_FULL"):
                allowed = frozenset((test,))
            if directory == "models" and name == "planetarygearbox" and test == "TestKitGearboxes" and not os.environ.get("PLANETARY_ALL_KITS"):
                allowed = frozenset(("TestKitGearboxes/planet608-shaft605", "TestKitGearboxes/planet608-shaft608"))
            if directory == "tools":
                go_test(command[2:], [test], package=package, allowed_skips=allowed, module="tools/render")
            elif allowed:
                go_test(command[2:], [test], package=package, allowed_skips=allowed)
            else:
                go_test(command[2:], [test], package=package)
        elif suite in RENDER_TESTS:
            run_renderer(suite, command)
        else:
            subprocess.run(command, check=True)
    elif len(sys.argv) > 1 and sys.argv[1] == "renderer-full":
        run_renderer_full()
    else:
        errors = check(json.loads(os.environ["NEEDS_JSON"]), os.environ["EVENT_NAME"])
        if errors:
            sys.exit("\n".join(errors))
        print("All planned jobs passed")


if __name__ == "__main__":
    main()
