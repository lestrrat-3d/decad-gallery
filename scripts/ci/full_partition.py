#!/usr/bin/env python3
"""Run the scheduled root-module tests not assigned to dedicated jobs."""

import os
import subprocess

import run_tests


ROOT = "github.com/lestrrat-3d/decad-gallery"
BEVEL = ROOT + "/models/bevelgear"
# Dedicated jobs cover whole packages or partition their tests and remainder.
DEDICATED = {ROOT + "/models/" + name for name in (
    "escapement", "onewayclutch", "clamparm", "sevensegment", "turbinefan", "m5bolt", "m5nut", "spinningwheel", "handcranksiren",
    "planetarygearbox",
)}


FULL_ENV = {"HANDCRANKFAN_FULL": "1", "PLANETARY_ALL_KITS": "1"}
FULL_VARIANTS = {
    (ROOT + "/models/handcrankfan", "TestHandCrankFanVariant/flat-three"),
    (ROOT + "/models/handcrankfan", "TestHandCrankFanVariant/steep-three"),
}


def select(inventory):
    """Return the ordinary package tests and unassigned bevel tests."""
    if not inventory or not DEDICATED <= inventory.keys() or BEVEL not in inventory:
        raise ValueError("root module inventory is missing a dedicated package")
    assigned = [test for tests in run_tests.BEVEL_TESTS["full"].values() for test in tests]
    if len(assigned) != len(set(assigned)):
        raise ValueError("full bevel shards contain duplicate test names")
    unknown = set(assigned) - inventory[BEVEL]
    if unknown:
        raise ValueError("unknown full bevel shard tests: " + ", ".join(sorted(unknown)))
    packages = sorted(set(inventory) - DEDICATED - {BEVEL})
    expected = {(package, test) for package in packages for test in inventory[package]}
    return packages, expected, sorted(inventory[BEVEL] - set(assigned))


def main():
    packages = subprocess.run(["go", "list", "./..."], check=True, capture_output=True, text=True).stdout.splitlines()
    if len(packages) != len(set(packages)):
        raise SystemExit("root module inventory contains duplicate packages")
    inventory = {}
    for package in packages:
        output = subprocess.run(["go", "test", "-list", "^Test", package],
                                check=True, capture_output=True, text=True).stdout
        inventory[package] = {line for line in output.splitlines() if line.startswith("Test")}
    try:
        selected, expected, remainder = select(inventory)
    except ValueError as exc:
        raise SystemExit(str(exc)) from exc
    expected.update(FULL_VARIANTS)
    env = os.environ.copy()
    env.update(FULL_ENV)
    run_tests.go_test([*selected, "-timeout", "60m"], env=env, qualified_expected=expected)
    if remainder:
        run_tests.go_test([BEVEL, "-run", run_tests.run_pattern(remainder), "-timeout", "60m"], remainder,
                          env=env, package=BEVEL)


if __name__ == "__main__":
    main()
