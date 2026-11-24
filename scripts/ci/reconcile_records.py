#!/usr/bin/env python3
"""Compare terminal Go test records; reject test skips and failed outcomes."""

import argparse
from collections import Counter, defaultdict
import json
from pathlib import Path
import re
import sys

import full_partition


SAMPLE_LIMIT = 10
IDENTITY = re.compile(r"^[^\s\x00-\x1f\x7f]+$")
META = ("version", "invocation", "module", "job", "case", "revision", "run_id", "run_attempt")
FULL_JOBS = {
    "package-workers": (), "renderer-full": tuple(str(i) for i in range(8)), "escapement": ("",),
    "bevel-remainder": ("",), "handcrankfan-full": ("default", "flat-three", "steep-three", "rest"), "swisslever-full": ("fork", "rest"),
    "planetary-full": ("planet605-shaft608", "planet608-shaft605", "planet608-shaft608",
                       "rest-0", "rest-1", "rest-2", "rest-3", "rest-4"),
    "onewayclutch": ("",), "clamparm": ("",), "sevensegment": ("parts", "assembly", "placement-x", "placement-offset", "placement-quarter"), "turbinefan": ("",),
    "wheel": ("default-verify", "default-linkage", "native", "small-seven-verify",
              "small-seven-linkage", "large-nine-verify", "large-nine-linkage", "twelve",
              "small-seventeen", "rest"),
    "siren": ("default", "spike", "slots12", "slots24", "rest"),
    "bevel-full": tuple("abcdefgh"),
}


def valid_id(value):
    return isinstance(value, str) and bool(IDENTITY.fullmatch(value))


def valid_package(value):
    return (valid_id(value) and not value.startswith(("/", ".")) and
            not any(part in ("", ".", "..") for part in value.split("/")) and
            not any(char in value for char in "\\:"))


def read_records(paths, expected_metadata=None, max_attempt=None):
    """Return leaf outcomes and bounded errors for complete invocation streams."""
    invocations = {}
    errors = Counter()
    examples = defaultdict(list)

    def error(reason, detail):
        errors[reason] += 1
        if len(examples[reason]) < SAMPLE_LIMIT:
            examples[reason].append(str(detail)[:200])

    for path in paths:
        artifact_attempt = None
        try:
            stream = open(path, encoding="utf-8", errors="replace")
        except OSError as exc:
            error("unreadable file", str(exc))
            continue
        with stream:
            for number, line in enumerate(stream, 1):
                location = f"{path}:{number}"
                try:
                    record = json.loads(line)
                except (ValueError, UnicodeError):
                    error("invalid record", location)
                    continue
                if not isinstance(record, dict) or record.get("version") != 1:
                    error("invalid record", location)
                    continue
                invocation = record.get("invocation")
                module = record.get("module")
                if not valid_id(invocation) or not valid_id(module):
                    error("invalid metadata", location)
                    continue
                metadata = tuple(record.get(key) for key in META)
                if any(not isinstance(record.get(key), str) for key in META[3:]):
                    error("invalid metadata", location)
                    continue
                if expected_metadata is not None:
                    expected = expected_metadata[path]
                    if any(record.get(key) != value for key, value in expected.items()):
                        error("unexpected metadata", location)
                        continue
                    attempt = record.get("run_attempt")
                    if (not isinstance(attempt, str) or not re.fullmatch(r"[1-9][0-9]*", attempt) or
                            (len(attempt), attempt) > (len(max_attempt), max_attempt) or
                            (artifact_attempt is not None and artifact_attempt != attempt)):
                        error("unexpected metadata", location)
                        continue
                    artifact_attempt = attempt
                state = invocations.setdefault(invocation, {"metadata": metadata, "complete": False,
                                                              "tests": [], "packages": set()})
                if state["metadata"] != metadata or state["complete"]:
                    error("inconsistent invocation", location)
                    continue
                kind = record.get("kind")
                if kind == "complete":
                    state["complete"] = True
                elif kind == "start" and record["job"] == "case-workers":
                    pass  # The case validator checks start/terminal multiplicity and ordering.
                elif kind == "malformed":
                    error("malformed producer event", location)
                elif kind in ("test", "package"):
                    package, test, status = record.get("package"), record.get("test"), record.get("status")
                    if (not valid_package(package) or status not in ("pass", "fail", "skip") or
                            (kind == "test" and not valid_id(test)) or (kind == "package" and test is not None)):
                        error("invalid terminal event", location)
                        continue
                    if status == "fail":
                        error("failed outcome", f"{module}:{package}:{test or '<package>'}")
                    if kind == "test" and status == "skip":
                        error("skipped test", f"{module}:{package}:{test}")
                    if kind == "package":
                        state["packages"].add(package)
                    else:
                        state["tests"].append((package, test, status))
                else:
                    error("invalid record kind", location)

    leaves = Counter()
    passed = set()
    packages_seen = set()
    for invocation, state in invocations.items():
        if not state["complete"]:
            error("incomplete invocation", invocation)
        if not state["packages"]:
            error("missing package outcome", invocation)
        module = state["metadata"][2]
        packages_seen.update((module, package) for package in state["packages"])
        parents = defaultdict(set)
        for package, test, _ in state["tests"]:
            parts = test.split("/")
            for end in range(1, len(parts)):
                parents[package].add("/".join(parts[:end]))
        for package, test, _ in state["tests"]:
            if package not in state["packages"]:
                error("missing package outcome", f"{module}:{package}")
        for package, test, status in state["tests"]:
            if status == "pass":
                passed.add((module, package, test))
            if test not in parents[package]:
                leaves[(module, package, test, status)] += 1
    if not invocations:
        error("no invocations", "no records found")
    return leaves, passed, packages_seen, errors, dict(examples)


def load_inventory(path):
    try:
        with open(path, encoding="utf-8") as stream:
            inventory = json.load(stream)
    except (OSError, ValueError, UnicodeError) as exc:
        raise ValueError(f"cannot read inventory: {str(exc)[:200]}") from exc
    if not isinstance(inventory, dict) or inventory.get("version") != 1:
        raise ValueError("invalid inventory version")
    modules = inventory.get("modules")
    if not isinstance(modules, dict) or set(modules) != {"root", "renderer"}:
        raise ValueError("inventory must contain root and renderer modules")
    tests, packages = set(), set()
    for name, module in (("root", "."), ("renderer", "tools/render")):
        mapping = modules[name]
        if not isinstance(mapping, dict) or not mapping:
            raise ValueError(f"invalid {name} package inventory")
        for package, names in mapping.items():
            if (not valid_package(package) or not isinstance(names, list) or
                    any(not valid_id(test) or "/" in test or not test.startswith("Test") for test in names) or
                    len(names) != len(set(names))):
                raise ValueError(f"invalid {name} package or test inventory")
            packages.add((module, package))
            tests.update((module, package, test) for test in names)
    variants = inventory.get("required_variants")
    if not isinstance(variants, list):
        raise ValueError("invalid required variants")
    for variant in variants:
        if not isinstance(variant, dict) or set(variant) != {"package", "test"}:
            raise ValueError("invalid required variant")
        package, test = variant["package"], variant["test"]
        if (not valid_package(package) or not valid_id(test) or
                any(not part for part in test.split("/")) or
                (".", package, test.split("/")[0]) not in tests):
            raise ValueError("required variant does not match root inventory")
        tests.add((".", package, test))
    return tests, packages


def verify_execution(directory, inventory_path, revision, run_id, run_attempt):
    """Validate the downloaded artifacts of one full-suite run against its inventory."""
    directory = Path(directory)
    expected = {}
    shard_packages = {}
    if "package-workers" in FULL_JOBS:
        try:
            load_inventory(inventory_path)
            with open(inventory_path, encoding="utf-8") as stream:
                root = json.load(stream)["modules"]["root"]
            ordinary, _, _ = full_partition.select({package: set(tests) for package, tests in root.items()})
            shards = full_partition.package_shards(ordinary)
            for index, packages in enumerate(shards):
                case = str(index)
                name = "package-workers-" + case
                path = directory / f"ci-records-{name}" / f"{name}.jsonl"
                expected[path] = {"job": "package-workers", "case": case, "revision": revision,
                                  "run_id": run_id, "module": "."}
                shard_packages[path] = {(".", package) for package in packages}
        except (OSError, ValueError, KeyError, TypeError):
            return {"ok": False, "inventory_error": "cannot determine package shards from inventory"}
    for job, cases in FULL_JOBS.items():
        for case in cases:
            name = job + (f"-{case}" if case else "")
            path = directory / f"ci-records-{name}" / f"{name}.jsonl"
            expected[path] = {"job": job, "case": case, "revision": revision,
                              "run_id": run_id,
                              "module": "tools/render" if job == "renderer-full" else "."}
    if (not valid_id(revision) or not valid_id(run_id) or not isinstance(run_attempt, str) or
            not re.fullmatch(r"[1-9][0-9]*", run_attempt)):
        raise ValueError("revision and run ID must be nonempty identities and run attempt must be positive")
    expected_dirs = {path.parent.name for path in expected}
    try:
        actual_dirs = {path.name for path in directory.iterdir()}
    except OSError:
        actual_dirs = set()
    missing_artifacts = expected_dirs - actual_dirs
    extra_artifacts = actual_dirs - expected_dirs
    invalid_contents = []
    for path in expected:
        if path.parent.name in actual_dirs:
            try:
                if {entry.name for entry in path.parent.iterdir()} != {path.name}:
                    invalid_contents.append(path.parent.name)
            except OSError:
                invalid_contents.append(path.parent.name)
    leaves, passed, packages, errors, examples = read_records(list(expected), expected, run_attempt)
    for path, required_packages in shard_packages.items():
        _, _, seen, shard_errors, _ = read_records([path], {path: expected[path]}, run_attempt)
        if seen != required_packages or shard_errors:
            errors["invalid package shard"] += 1
    inventory_error = None
    try:
        inventory_tests, inventory_packages = load_inventory(inventory_path)
    except ValueError as exc:
        inventory_error = str(exc)
        inventory_tests, inventory_packages = set(), set()
    missing_tests = inventory_tests - passed
    missing_packages = inventory_packages - packages
    duplicates = {key: count for key, count in leaves.items() if count > 1}
    return {"ok": not (missing_artifacts or extra_artifacts or invalid_contents or errors or inventory_error or
                       missing_tests or missing_packages or duplicates),
            "errors": dict(errors), "examples": examples, "inventory_error": inventory_error,
            "missing_artifacts_count": len(missing_artifacts),
            "missing_artifacts_sample": sorted(missing_artifacts)[:SAMPLE_LIMIT],
            "extra_artifacts_count": len(extra_artifacts),
            "extra_artifacts_sample": sorted(extra_artifacts)[:SAMPLE_LIMIT],
            "invalid_artifact_contents_count": len(invalid_contents),
            "invalid_artifact_contents_sample": sorted(invalid_contents)[:SAMPLE_LIMIT],
            "missing_inventory_tests_count": len(missing_tests),
            "missing_inventory_tests_sample": sorted(missing_tests)[:SAMPLE_LIMIT],
            "missing_inventory_packages_count": len(missing_packages),
            "missing_inventory_packages_sample": sorted(missing_packages)[:SAMPLE_LIMIT],
            "duplicate_count": len(duplicates), "duplicate_sample": sorted(duplicates.items())[:SAMPLE_LIMIT]}


def reconcile(reference_paths, candidate_paths, required=(), inventory_path=None):
    reference, ref_passed, ref_packages, ref_errors, ref_examples = read_records(reference_paths)
    candidate, passed, cand_packages, cand_errors, cand_examples = read_records(candidate_paths)
    missing_required = [identity for identity in required if identity not in passed]
    inventory_error = None
    inventory_tests, inventory_packages = set(), set()
    if inventory_path is not None:
        try:
            inventory_tests, inventory_packages = load_inventory(inventory_path)
        except ValueError as exc:
            inventory_error = str(exc)
    reference_missing_tests = inventory_tests - ref_passed
    candidate_missing_tests = inventory_tests - passed
    reference_missing_packages = inventory_packages - ref_packages
    candidate_missing_packages = inventory_packages - cand_packages
    missing = reference - candidate
    extra = candidate - reference
    duplicates = {key: count for key, count in candidate.items() if count > 1}
    return {"ok": not (ref_errors or cand_errors or missing_required or missing or extra or inventory_error or
                       reference_missing_tests or candidate_missing_tests or reference_missing_packages or
                       candidate_missing_packages),
            "reference_errors": dict(ref_errors), "reference_examples": ref_examples,
            "candidate_errors": dict(cand_errors), "candidate_examples": cand_examples,
            "inventory_error": inventory_error,
            "reference_missing_inventory_tests_count": len(reference_missing_tests),
            "reference_missing_inventory_tests_sample": sorted(reference_missing_tests)[:SAMPLE_LIMIT],
            "candidate_missing_inventory_tests_count": len(candidate_missing_tests),
            "candidate_missing_inventory_tests_sample": sorted(candidate_missing_tests)[:SAMPLE_LIMIT],
            "reference_missing_inventory_packages_count": len(reference_missing_packages),
            "reference_missing_inventory_packages_sample": sorted(reference_missing_packages)[:SAMPLE_LIMIT],
            "candidate_missing_inventory_packages_count": len(candidate_missing_packages),
            "candidate_missing_inventory_packages_sample": sorted(candidate_missing_packages)[:SAMPLE_LIMIT],
            "missing_required_count": len(missing_required),
            "missing_required_sample": sorted(missing_required)[:SAMPLE_LIMIT],
            "missing_count": sum(missing.values()), "missing_sample": sorted(missing.items())[:SAMPLE_LIMIT],
            "extra_count": sum(extra.values()), "extra_sample": sorted(extra.items())[:SAMPLE_LIMIT],
            "duplicate_count": len(duplicates), "duplicate_sample": sorted(duplicates.items())[:SAMPLE_LIMIT]}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--reference", action="append", metavar="JSONL")
    parser.add_argument("--candidate", action="append", metavar="JSONL")
    parser.add_argument("--execution-dir", metavar="DIR", help="validate one downloaded full-suite execution")
    parser.add_argument("--revision")
    parser.add_argument("--run-id")
    parser.add_argument("--run-attempt")
    parser.add_argument("--require", action="append", default=[], metavar="MODULE:PACKAGE:TEST")
    parser.add_argument("--inventory", metavar="JSON", help="require both executions to cover this inventory")
    args = parser.parse_args()
    if args.execution_dir:
        if (args.reference or args.candidate or args.require or not args.inventory or
                not all((args.revision, args.run_id, args.run_attempt))):
            parser.error("--execution-dir requires --inventory, --revision, --run-id and --run-attempt")
        try:
            result = verify_execution(args.execution_dir, args.inventory, args.revision, args.run_id, args.run_attempt)
        except ValueError as exc:
            parser.error(str(exc))
        print(json.dumps(result, indent=2, sort_keys=True))
        return 0 if result["ok"] else 1
    if not args.reference or not args.candidate or any((args.revision, args.run_id, args.run_attempt)):
        parser.error("comparison requires --reference and --candidate")
    required = []
    for value in args.require:
        parts = value.split(":")
        if len(parts) != 3 or not valid_id(parts[0]) or not valid_package(parts[1]) or not valid_id(parts[2]):
            parser.error("--require must be MODULE:PACKAGE:TEST")
        required.append(tuple(parts))
    result = reconcile(args.reference, args.candidate, required, args.inventory)
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0 if result["ok"] else 1


if __name__ == "__main__":
    sys.exit(main())
