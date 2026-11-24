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
        output = subprocess.run(base + ["test", "-list", "^(Test|Example|Fuzz)", package],
                                check=True, capture_output=True, text=True).stdout
        names = output.splitlines()
        unsupported = [name for name in names if name.startswith(("Example", "Fuzz"))]
        if unsupported:
            raise ValueError(f"{package} has examples or fuzz tests outside the CI inventory: "
                             + ", ".join(unsupported))
        result[package] = sorted(line for line in names if line.startswith("Test"))
    return result


def report(root, renderer):
    # Use the same partition validation as the full runner.
    ordinary, _, bevel_remainder = full_partition.select({package: set(tests) for package, tests in root.items()})
    assignments = {}
    def add(package, name, job):
        assignments.setdefault(package, {}).setdefault(name, []).append(job)

    for index, packages in enumerate(full_partition.package_shards(ordinary)):
        for package in packages:
            for test in root[package]:
                add(package, test, f"package-workers/{index}")
    for name, job in (("onewayclutch", "onewayclutch"), ("turbinefan", "turbinefan"),
                      ("clamparm", "clamparm"), ("fastener", "clamparm")):
        package = full_partition.ROOT + "/models/" + name
        for test in root[package]:
            add(package, test, job)
    for shard, tests in run_tests.BEVEL_TESTS["full"].items():
        for test in tests:
            add(full_partition.BEVEL, test, "bevel-full/" + shard)
    for test in bevel_remainder:
        add(full_partition.BEVEL, test, "bevel-remainder")
    package = full_partition.ROOT + "/models/handcrankfan"
    for case in (*full_partition.FAN_CASES, "rest"):
        for test in full_partition.fan_tests(set(root[package]), case):
            add(package, test, "handcrankfan-full/" + case)
    package = full_partition.ROOT + "/models/swisslever"
    for case in ("fork", "rest"):
        for test in full_partition.swiss_tests(set(root[package]), case):
            add(package, test, "swisslever-full/" + case)
    for index, tests in enumerate(run_tests.ESCAPEMENT_SHARDS, 1):
        for test in tests:
            add(full_partition.ROOT + "/models/escapement", test, f"escapement/{index}")
    planetary = full_partition.ROOT + "/models/planetarygearbox"
    planetary_tests = root[planetary]
    if "TestKitGearboxes" in planetary_tests:
        for kit in run_tests.PLANETARY_KITS:
            add(planetary, "TestKitGearboxes", "planetary-full/" + kit)
    for index, tests in enumerate(run_tests.planetary_test_shards(
            [test for test in planetary_tests if test != "TestKitGearboxes"],
            run_tests.PLANETARY_FULL_REST_SHARDS)):
        for test in tests:
            add(planetary, test, f"planetary-full/rest-{index}")
    for case, tests in run_tests.WHEEL_EXPECTED.items():
        for test in tests:
            add(full_partition.ROOT + "/models/spinningwheel", test, "wheel/" + case)
    for case, (_, tests) in run_tests.SIREN_CASES.items():
        for test in tests:
            add(full_partition.ROOT + "/models/handcranksiren", test, "siren/" + case)
    for shard, tests in run_tests.SEVENSEGMENT_SHARDS.items():
        for test in tests:
            cases = run_tests.SEVENSEGMENT_PLACEMENTS if shard == "placement" else (shard,)
            for case in cases:
                add(full_partition.ROOT + "/models/sevensegment", test, "sevensegment/" + case)

    for package, label in ((full_partition.ROOT + "/models/spinningwheel", "wheel/rest"),
                           (full_partition.ROOT + "/models/handcranksiren", "siren/rest")):
        assigned = assignments.get(package, {})
        for test in root[package]:
            if test not in assigned:
                add(package, test, label)

    variants = [{"package": package, "test": test} for package, test in sorted(full_partition.FULL_VARIANTS)]
    variants.extend({"package": planetary, "test": "TestKitGearboxes/" + kit}
                    for kit in run_tests.PLANETARY_KITS)
    variants.extend({"package": full_partition.ROOT + "/models/sevensegment",
                     "test": "TestSevenSegmentPlacement/" + name}
                    for _, name in run_tests.SEVENSEGMENT_PLACEMENTS.values())
    renderer_shards = gate.renderer_full_shards(renderer)
    return {
        "version": 1,
        "modules": {"root": root, "renderer": renderer},
        "known_assignments": {
            "root": assignments,
            "renderer": {package: {test: [f"renderer-full/{index}"]
                                   for index, shard in enumerate(renderer_shards)
                                   for pkg, names in shard if pkg == package for test in names}
                         for package in renderer},
        },
        "full_environment": full_partition.FULL_ENV,
        "required_variants": variants,
        "renderer_named_checks": {name: list(tests) for name, (_, _, tests) in gate.RENDER_TESTS.items()},
        "subtests": "Only explicitly named subtests are shown; other subtests are discovered at runtime.",
    }


def split_renderer(discovered):
    """Split a root-module inventory into ordinary root packages and the renderer group."""
    root = {package: tests for package, tests in discovered.items() if not run_tests.RENDERER_PACKAGE.match(package)}
    renderer = {package: tests for package, tests in discovered.items() if run_tests.RENDERER_PACKAGE.match(package)}
    return root, renderer


def main():
    print(json.dumps(report(*split_renderer(discover(["go"]))), indent=2, sort_keys=True))


if __name__ == "__main__":
    main()
