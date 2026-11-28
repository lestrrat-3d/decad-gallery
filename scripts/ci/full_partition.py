#!/usr/bin/env python3
"""Run the scheduled root-module tests not assigned to dedicated jobs."""

import os
import json
import math
import subprocess
import sys

import run_tests
from runner_cpu import print_runner_cpu


ROOT = "github.com/lestrrat-3d/decad-gallery"
CYCLOIDAL = ROOT + "/models/cycloidal"
BEVEL = ROOT + "/models/bevelgear"
# Dedicated jobs cover whole packages or partition their tests and remainder.
DEDICATED = {ROOT + "/models/" + name for name in (
    "escapement", "onewayclutch", "clamparm", "sevensegment", "turbinefan", "fastener", "spinningwheel", "handcranksiren",
    "planetarygearbox", "handcrankfan", "swisslever",
)}


FULL_ENV = {"HANDCRANKFAN_FULL": "1", "PLANETARY_ALL_KITS": "1"}
FULL_VARIANTS = {
    (ROOT + "/models/handcrankfan", "TestHandCrankFanVariant/flat-three"),
    (ROOT + "/models/handcrankfan", "TestHandCrankFanVariant/steep-three"),
}
FAN_VARIANTS = ("flat-three", "steep-three")
FAN_CASES = {"default": "TestHandCrankFanDefault",
             **{name: "TestHandCrankFanVariant" for name in FAN_VARIANTS}}
SWISS_FORK = "TestForkAction"
PACKAGES_PER_SHARD = 4
MAX_PACKAGE_SHARDS = 16


def package_shards(packages, case_plan=None):
    """Assign each ordinary package to exactly one bounded, deterministic shard."""
    packages = sorted(packages)
    if not packages or len(packages) != len(set(packages)):
        raise ValueError("ordinary package inventory is empty or duplicated")
    if case_plan is not None:
        package = CYCLOIDAL
        if case_plan["manifest"]["package"] != package or package not in packages:
            raise ValueError("case plan does not own the cycloidal package")
        packages.remove(package)
    if not packages:
        return []
    count = min(MAX_PACKAGE_SHARDS, math.ceil(len(packages) / PACKAGES_PER_SHARD))
    return [packages[index::count] for index in range(count)]


def ordinary_packages():
    packages = subprocess.run(["go", "list", "./..."], check=True, capture_output=True, text=True).stdout.splitlines()
    packages = [package for package in packages if not run_tests.RENDERER_PACKAGE.match(package)]
    if len(packages) != len(set(packages)) or not DEDICATED <= set(packages) or BEVEL not in packages:
        raise ValueError("root package inventory is missing dedicated packages or contains duplicates")
    return sorted(set(packages) - DEDICATED - {BEVEL})


def package_matrix():
    return {"include": [{"case": str(index), "packages": packages}
                        for index, packages in enumerate(package_shards(ordinary_packages(), json.loads(os.environ.get("CASE_PLAN", "null"))))]}


def swiss_tests(tests, case):
    """Keep shared-fixture tests together and isolate the fork sweep."""
    if SWISS_FORK not in tests:
        raise ValueError("Swiss inventory is missing the fork sweep")
    if case == "fork":
        return [SWISS_FORK]
    if case == "rest":
        return sorted(tests - {SWISS_FORK})
    raise ValueError("invalid Swiss case")


def fan_tests(tests, case):
    """Keep shared-fixture tests together and isolate the two full builds."""
    if not set(FAN_CASES.values()) <= tests:
        raise ValueError("fan inventory is missing a full build test")
    if case in FAN_CASES:
        return [FAN_CASES[case]]
    if case == "rest":
        return sorted(tests - set(FAN_CASES.values()))
    raise ValueError("invalid fan case")


def select(inventory):
    """Return the ordinary package tests and unassigned bevel tests."""
    if not inventory or not DEDICATED <= inventory.keys() or BEVEL not in inventory:
        raise ValueError("root module inventory is missing a dedicated package")
    remainder = bevel_remainder(inventory[BEVEL])
    packages = sorted(set(inventory) - DEDICATED - {BEVEL})
    expected = {(package, test) for package in packages for test in inventory[package]}
    return packages, expected, remainder


def bevel_remainder(tests):
    assigned = [test for tests in run_tests.BEVEL_TESTS["full"].values() for test in tests]
    if len(assigned) != len(set(assigned)):
        raise ValueError("full bevel shards contain duplicate test names")
    unknown = set(assigned) - tests
    if unknown:
        raise ValueError("unknown full bevel shard tests: " + ", ".join(sorted(unknown)))
    return sorted(tests - set(assigned))


def discover(package):
    prefixes = ("Test", "Example", "Fuzz") if package == CYCLOIDAL else ("Test",)
    output = subprocess.run(["go", "test", "-list", "^(" + "|".join(prefixes) + ")", package],
                            check=True, capture_output=True, text=True).stdout
    tests = [line for line in output.splitlines() if line.startswith(prefixes)]
    if len(tests) != len(set(tests)):
        raise ValueError("duplicate test names in " + package)
    return set(tests)


def run_package_shard(case):
    shards = package_shards(ordinary_packages(), json.loads(os.environ.get("CASE_PLAN", "null")))
    if not case or not case.isdigit() or str(int(case)) != case or int(case) >= len(shards):
        raise ValueError("invalid package shard")
    packages = shards[int(case)]
    tests = {package: discover(package) for package in packages}
    env = os.environ.copy()
    env.update(FULL_ENV)
    print_runner_cpu()
    command = ["go", "test", "-json", "-p=1", "-parallel=4", *packages, "-timeout", "60m", "-count=1"]
    proc = subprocess.Popen(command, stdout=subprocess.PIPE, text=True, env=env)
    outcomes = {}
    audit = {"test": False, "invalid": False, "packages": set(), "outcomes": [], "no_tests": set()}
    finished = run_tests.finished_tests(proc.stdout, qualified=True, package_outcomes=outcomes, event_audit=audit)
    if proc.wait() != 0:
        raise ValueError("package shard failed")
    expected = {(package, test) for package, names in tests.items() for test in names}
    if (audit["invalid"] or audit["packages"] != set(packages) or
            audit.get("test_starts", {}) != audit.get("test_terminals", {}) or
            any(count != 1 for count in audit.get("test_starts", {}).values()) or
            any(package in audit.get("test_packages", set()) for package in packages if not tests[package]) or
            len(audit["outcomes"]) != len(packages) or
            {package for package, _ in audit["outcomes"]} != set(packages) or
            any(finished.get(test) != "pass" for test in expected) or
            any(status != "pass" for status in finished.values()) or set(outcomes) != set(packages) or
            any(status != "pass" and not (status == "skip" and not tests[package] and
                                           package in audit["no_tests"])
                for package, status in outcomes.items())):
        raise ValueError("package shard has missing or failed outcomes")


def main(mode="repository", case=None):
    try:
        if mode == "package-matrix":
            matrix = package_matrix()
            with open(os.environ["GITHUB_OUTPUT"], "a", encoding="utf-8") as output:
                print("matrix=" + json.dumps(matrix, separators=(",", ":")), file=output)
            return
        if mode == "package-shard":
            run_package_shard(case)
            return
        if mode == "repository":
            packages = [package for package in subprocess.run(["go", "list", "./..."], check=True,
                        capture_output=True, text=True).stdout.splitlines()
                        if not run_tests.RENDERER_PACKAGE.match(package)]
            if len(packages) != len(set(packages)):
                raise ValueError("root module inventory contains duplicate packages")
            selected, expected, _ = select({package: discover(package) for package in packages})
        elif mode == "bevel-remainder":
            remainder = bevel_remainder(discover(BEVEL))
        elif mode in ("handcrankfan", "swisslever"):
            package = ROOT + "/models/" + mode
            inventory = {package: discover(package)}
        else:
            raise ValueError("invalid full partition")
    except ValueError as exc:
        raise SystemExit(str(exc)) from exc
    env = os.environ.copy()
    env.update(FULL_ENV)
    if mode == "repository":
        run_tests.go_test([*selected, "-timeout", "60m"], env=env, qualified_expected=expected)
    elif mode == "bevel-remainder":
        pattern = run_tests.run_pattern(remainder) if remainder else "^$"
        run_tests.go_test([BEVEL, "-run", pattern, "-timeout", "60m"], remainder, env=env, package=BEVEL,
                          require_empty_package=not remainder)
    elif mode == "handcrankfan":
        package = ROOT + "/models/handcrankfan"
        try:
            tests = fan_tests(inventory[package], case)
        except ValueError as exc:
            raise SystemExit(str(exc)) from exc
        if not tests:
            raise SystemExit("fan partition is empty")
        required = set(tests)
        pattern = run_tests.run_pattern(tests)
        if case in FAN_VARIANTS:
            required.add("TestHandCrankFanVariant/" + case)
            pattern = run_tests.run_pattern(["TestHandCrankFanVariant/" + case])
            env["HANDCRANKFAN_VARIANTS"] = ",".join(FAN_VARIANTS)
        run_tests.go_test([package, "-run", pattern, "-timeout", "60m"],
                          sorted(required), env=env, package=package)
    elif mode == "swisslever":
        package = ROOT + "/models/" + mode
        try:
            tests = swiss_tests(inventory[package], case)
        except ValueError as exc:
            raise SystemExit(str(exc)) from exc
        if not tests:
            raise SystemExit("Swiss partition is empty")
        run_tests.go_test([package, "-run", run_tests.run_pattern(tests), "-timeout", "60m"],
                          tests, env=env, package=package)
    else:
        raise SystemExit("invalid full partition")


if __name__ == "__main__":
    main(*sys.argv[1:])
