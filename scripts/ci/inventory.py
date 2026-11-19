#!/usr/bin/env python3
"""Report the scheduled Go test inventory without executing test bodies."""

import json
import subprocess

import full_partition
import gate
import run_tests


def discover(base):
    packages = subprocess.run(base + ["list", "./..."], check=True, capture_output=True, text=True).stdout.splitlines()
    if not packages or len(packages) != len(set(packages)):
        raise ValueError("package inventory is empty or duplicated")
    result = {}
    for package in sorted(packages):
        output = subprocess.run(base + ["test", "-list", "^Test", package],
                                check=True, capture_output=True, text=True).stdout
        result[package] = sorted(line for line in output.splitlines() if line.startswith("Test"))
    return result


def report(root, renderer):
    # Use the same partition validation as the full runner.
    ordinary, _, bevel_remainder = full_partition.select({package: set(tests) for package, tests in root.items()})
    assignments = {}
    def add(package, name, job):
        assignments.setdefault(package, {}).setdefault(name, []).append(job)

    for package in ordinary:
        for test in root[package]:
            add(package, test, "repository")
    for name, job in (("onewayclutch", "onewayclutch"), ("turbinefan", "turbinefan"),
                      ("clamparm", "clamparm"), ("m5bolt", "clamparm"), ("m5nut", "clamparm")):
        package = full_partition.ROOT + "/models/" + name
        for test in root[package]:
            add(package, test, job)
    for shard, tests in run_tests.BEVEL_TESTS["full"].items():
        for test in tests:
            add(full_partition.BEVEL, test, "bevel-full/" + shard)
    for test in bevel_remainder:
        add(full_partition.BEVEL, test, "repository/bevel-remainder")
    for index, tests in enumerate(run_tests.ESCAPEMENT_SHARDS, 1):
        for test in tests:
            add(full_partition.ROOT + "/models/escapement", test, f"escapement/{index}")
    for case, tests in run_tests.WHEEL_EXPECTED.items():
        for test in tests:
            add(full_partition.ROOT + "/models/spinningwheel", test, "wheel/" + case)
    for case, (_, tests) in run_tests.SIREN_CASES.items():
        for test in tests:
            add(full_partition.ROOT + "/models/handcranksiren", test, "siren/" + case)

    for package, label in ((full_partition.ROOT + "/models/spinningwheel", "wheel/rest"),
                           (full_partition.ROOT + "/models/handcranksiren", "siren/rest")):
        assigned = assignments.get(package, {})
        for test in root[package]:
            if test not in assigned:
                add(package, test, label)

    variants = [{"package": package, "test": test} for package, test in sorted(full_partition.FULL_VARIANTS)]
    return {
        "version": 1,
        "modules": {"root": root, "renderer": renderer},
        "known_assignments": {
            "root": assignments,
            "renderer": {package: {test: ["renderer-full"] for test in tests}
                         for package, tests in renderer.items()},
        },
        "full_environment": full_partition.FULL_ENV,
        "required_variants": variants,
        "renderer_named_checks": {name: list(tests) for name, (_, tests) in gate.RENDER_TESTS.items()},
        "subtests": "Only explicitly named subtests are shown; other subtests are discovered at runtime.",
    }


def main():
    print(json.dumps(report(discover(["go"]), discover(["go", "-C", "tools/render"])), indent=2, sort_keys=True))


if __name__ == "__main__":
    main()
