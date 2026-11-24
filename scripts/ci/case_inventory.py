#!/usr/bin/env python3
"""Discover opted-in test cases by running their real test tables."""

import json
import hashlib
import os
from pathlib import Path
import re
import subprocess
import sys

import run_tests
import reconcile_records


def execution_identity():
    revision = subprocess.run(["git", "rev-parse", "HEAD"], check=True, capture_output=True, text=True).stdout.strip()
    status = subprocess.run(["git", "diff", "--quiet", "HEAD", "--"], capture_output=True)
    if status.returncode:
        raise ValueError("checkout contains uncommitted tracked files")
    if os.environ.get("GITHUB_SHA", revision) != revision:
        raise ValueError("checkout does not match the workflow revision")
    profile = subprocess.run(["go", "env", "-json", "GOOS", "GOARCH", "CGO_ENABLED", "GOFLAGS", "GOVERSION"],
                             check=True, capture_output=True, text=True)
    return {"revision": revision, "run_id": os.environ.get("GITHUB_RUN_ID", "local"),
            "attempt": os.environ.get("GITHUB_RUN_ATTEMPT", "1"), "profile": json.loads(profile.stdout)}


def source_digest(name):
    root = Path("models", name)
    files = sorted(root.glob("*_test.go"))
    if not files or any(path.is_symlink() for path in files):
        raise ValueError("invalid test source inventory")
    status = subprocess.run(["git", "status", "--porcelain=v1", "--untracked-files=all", "--", str(root)],
                            check=True, capture_output=True, text=True).stdout
    tracked = subprocess.run(["git", "ls-files", "-z", "--", str(root)], check=True, capture_output=True).stdout
    tracked_tests = {os.fsdecode(path) for path in tracked.split(b"\0") if path and
                     Path(os.fsdecode(path)).parent == root and os.fsdecode(path).endswith("_test.go")}
    if status or tracked_tests != {str(path) for path in files}:
        raise ValueError("test sources do not match the checked-out revision")
    return file_digest(files)


def file_digest(files):
    digest = hashlib.sha256()
    for path in sorted(files):
        name_bytes = path.name.encode("utf-8")
        data = path.read_bytes()
        digest.update(len(name_bytes).to_bytes(8, "big"))
        digest.update(name_bytes)
        digest.update(len(data).to_bytes(8, "big"))
        digest.update(data)
    return digest.hexdigest()


# Both reviewed parents create a document and parameters per child. Their local
# helpers do not access defaultDriveFixture. Pin the entire local Go package
# so changing a parent, helper, or package initializer needs a fresh review.
# Updating the model's inventory fingerprint alone cannot authorize a split.
CYCLOIDAL_REVIEWED_GO = "684eafbcc84332696858c1137067d23a01b3c681b4994eb5cc50b5e18fa72309"


def reviewed_cases(name, parents):
    if name != "cycloidal" or parents != ["TestBuildOptions", "TestMeasuredFits"]:
        return False
    files = list(Path("models", name).glob("*.go"))
    return (all(path.is_file() and not path.is_symlink() for path in files) and
            file_digest(files) == CYCLOIDAL_REVIEWED_GO)


def discover(name):
    if not re.fullmatch(r"[a-z][a-z0-9]*", name):
        raise ValueError("invalid model name")
    config_path = Path("models", name, "ci_cases.json")
    if not config_path.exists():
        return None
    try:
        config = json.loads(config_path.read_text())
    except (UnicodeError, json.JSONDecodeError):
        return None
    if (not isinstance(config, dict) or set(config) != {"version", "parents", "source_sha256"} or config["version"] != 1 or
            not isinstance(config["parents"], list) or not isinstance(config["source_sha256"], str) or
            not re.fullmatch(r"[0-9a-f]{64}", config["source_sha256"])):
        return None
    if not reviewed_cases(name, config["parents"]) or source_digest(name) != config["source_sha256"]:
        return None
    if any("TestMain" in path.read_text() for path in Path("models", name).glob("*_test.go")):
        return None
    parents = config["parents"]
    if (not parents or len(parents) != len(set(parents)) or
            any(not isinstance(parent, str) or not re.fullmatch(r"Test[A-Za-z0-9_]+", parent) for parent in parents)):
        raise ValueError("invalid case parents")
    package = "github.com/lestrrat-3d/decad-gallery/models/" + name
    listing = subprocess.run(["go", "test", "-list", "^(Test|Example|Fuzz)", "./models/" + name],
                             check=True, capture_output=True, text=True).stdout.splitlines()
    if any(line.startswith(("Example", "Fuzz")) or
           (line.startswith("Test") and not re.fullmatch(r"Test[A-Za-z0-9_]+", line)) for line in listing):
        return None
    tests = [line for line in listing if re.fullmatch(r"Test[A-Za-z0-9_]+", line)]
    if len(tests) != len(set(tests)) or not set(parents) <= set(tests) or len(tests) == len(parents):
        return None
    pattern = "^(" + "|".join(parents) + ")$"
    env = dict(os.environ, CI_DISCOVER_CASES="1")
    result = subprocess.run(["go", "test", "-json", "-parallel=1", "-count=1", "-timeout=4m", "-run", pattern,
                             "./models/" + name], check=True, capture_output=True, text=True, env=env, timeout=300)
    found = {}
    starts = []
    terminal = []
    package_pass = 0
    for line in result.stdout.splitlines():
        event = json.loads(line)
        if event.get("Package") != package:
            raise ValueError("unexpected discovery package")
        test = event.get("Test")
        action = event.get("Action")
        if test and (test not in parents or "/" in test):
            raise ValueError("discovery executed an unexpected test or child")
        if test and action == "run":
            starts.append(test)
        if test and action in {"pass", "fail", "skip"}:
            terminal.append((test, action))
        if not test and action == "pass":
            package_pass += 1
        output = event.get("Output", "")
        if action == "output" and output.startswith("CI_CASES "):
            data = json.loads(output.removeprefix("CI_CASES "))
            if set(data) != {"parent", "names"} or data["parent"] != test or test in found:
                raise ValueError("invalid or duplicate case manifest")
            names = data["names"]
            if (not isinstance(names, list) or not names or len(names) != len(set(names)) or
                    any(not isinstance(case, str) or not re.fullmatch(r"[A-Za-z0-9_]+", case) for case in names)):
                raise ValueError("invalid case names")
            found[test] = names
    if (sorted(starts) != sorted(parents) or
            sorted(terminal) != sorted((parent, "pass") for parent in parents) or package_pass != 1 or
            set(found) != set(parents)):
        raise ValueError("discovery is incomplete")
    return {"version": 1, "package": package, "parents": found, "tests": sorted(tests),
            "source_sha256": config["source_sha256"],
            "identity": execution_identity()}


def run_case(manifest, parent, case):
    run_group(manifest, parent, [case])


def validate_manifest(manifest):
    prefix = "github.com/lestrrat-3d/decad-gallery/models/"
    if not isinstance(manifest, dict) or not isinstance(manifest.get("package"), str):
        raise ValueError("invalid case manifest")
    package = manifest["package"]
    if not package.startswith(prefix):
        raise ValueError("invalid case package")
    name = package.removeprefix(prefix)
    if discover(name) != manifest:
        raise ValueError("case manifest does not match the current checkout and execution profile")


def build_plan(manifest):
    """Assign each discovered child exactly once, with one audit and one remainder."""
    validate_manifest(manifest)
    return {"version": 1, "manifest": manifest, "jobs": assignments(manifest)}


def assignments(manifest):
    jobs = [{"case": "audit", "kind": "audit"}, {"case": "remainder", "kind": "remainder"}]
    for parent, names in sorted(manifest["parents"].items()):
        for shard in range(min(2, len(names))):
            selected = sorted(names)[shard::2]
            jobs.append({"case": f"{parent}-{shard}", "kind": "group", "parent": parent, "names": selected})
    return jobs


def run_planned(plan, case):
    expected = build_plan(plan["manifest"])
    if plan != expected:
        raise ValueError("case plan is incomplete or altered")
    if os.environ.get("CI_TEST_CASE", case) != case:
        raise ValueError("worker case does not match its assignment")
    jobs = [job for job in plan["jobs"] if job["case"] == case]
    if len(jobs) != 1:
        raise ValueError("case is not uniquely assigned")
    job = jobs[0]
    if job["kind"] == "audit":
        audit_undiscovered(plan["manifest"])
    elif job["kind"] == "remainder":
        run_remainder(plan["manifest"])
    else:
        run_group(plan["manifest"], job["parent"], job["names"])
    receipt_path = os.environ.get("CI_CASE_RECEIPT")
    if receipt_path:
        encoded = json.dumps(plan, sort_keys=True, separators=(",", ":")).encode()
        receipt = {"version": 1, "case": case, "kind": job["kind"],
                   "plan_sha256": hashlib.sha256(encoded).hexdigest(), "identity": plan["manifest"]["identity"]}
        if job["kind"] != "audit":
            receipt["record_sha256"] = record_digest(Path(os.environ["CI_TEST_RECORDS"]))
        Path(receipt_path).write_text(json.dumps(receipt, sort_keys=True) + "\n")


def record_digest(path):
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def verify_worker_records(path, expected_digest, metadata):
    if record_digest(path) != expected_digest:
        raise ValueError("worker records differ from receipt")
    package = metadata["package"]
    opened = set()
    passed = set()
    started = closed = complete = False
    invocation = None
    with path.open() as stream:
        for line in stream:
            row = json.loads(line)
            if (complete or not isinstance(row, dict) or row.get("version") != 1 or
                    any(row.get(k) != v for k, v in metadata.items() if k != "package") or
                    not reconcile_records.valid_id(row.get("invocation"))):
                raise ValueError("invalid worker records")
            if invocation is None:
                invocation = row["invocation"]
            if row["invocation"] != invocation:
                raise ValueError("mixed case invocations")
            kind, test = row.get("kind"), row.get("test")
            if kind == "complete" and closed:
                complete = True
            elif row.get("package") != package or closed:
                raise ValueError("unexpected package in worker records")
            elif kind == "start" and test is None and not started:
                started = True
            elif (kind == "start" and started and reconcile_records.valid_id(test)
                  and test not in opened and test not in passed):
                opened.add(test)
            elif kind == "test" and test in opened and row.get("status") == "pass":
                opened.remove(test)
                passed.add(test)
            elif kind == "package" and test is None and started and not opened and passed and row.get("status") == "pass":
                closed = True
            else:
                raise ValueError("incomplete, duplicate, or unsuccessful worker records")
    if not complete:
        raise ValueError("incomplete worker records")
    return passed


def verify_execution(plan, receipts, records):
    """Require every planned case, audit, remainder and record exactly once."""
    if plan != build_plan(plan["manifest"]):
        raise ValueError("case plan is incomplete or altered")
    jobs = {job["case"]: job for job in plan["jobs"]}
    if set(receipts) != set(jobs) or set(records) != {case for case, job in jobs.items() if job["kind"] != "audit"}:
        raise ValueError("case artifacts do not match the plan")
    encoded = json.dumps(plan, sort_keys=True, separators=(",", ":")).encode()
    digest = hashlib.sha256(encoded).hexdigest()
    identity = plan["manifest"]["identity"]
    package = plan["manifest"]["package"]
    parents = set(plan["manifest"]["parents"])
    expected_remainder = set(plan["manifest"]["tests"]) - parents
    all_leaves = {}
    for case, job in jobs.items():
        path = Path(receipts[case])
        if path.stat().st_size > 16384:
            raise ValueError("case receipt is too large")
        receipt = json.loads(path.read_text())
        expected = {"version": 1, "case": case, "kind": job["kind"], "plan_sha256": digest, "identity": identity}
        if job["kind"] != "audit":
            expected["record_sha256"] = receipt.get("record_sha256")
        if receipt != expected:
            raise ValueError("case receipt does not match the plan")
        if job["kind"] == "audit":
            continue
        path = Path(records[case])
        metadata = {path: {"job": "case-workers", "case": case, "revision": identity["revision"],
                           "run_id": identity["run_id"], "run_attempt": identity["attempt"], "module": "."}}
        verify_worker_records(path, receipt["record_sha256"], {**metadata[path], "package": package})
        leaves, passed, packages, errors, _ = reconcile_records.read_records([path], metadata, identity["attempt"])
        if errors or packages != {(".", package)}:
            raise ValueError("case records are invalid")
        tests = {test for module, pkg, test in passed if module == "." and pkg == package}
        roots = {test for test in tests if "/" not in test}
        if job["kind"] == "group":
            expected_tests = {job["parent"], *(job["parent"] + "/" + name for name in job["names"])}
            if tests != expected_tests:
                raise ValueError("case group does not match its assignment")
        elif roots != expected_remainder:
            raise ValueError("case remainder does not match its assignment")
        for leaf, count in leaves.items():
            all_leaves[leaf] = all_leaves.get(leaf, 0) + count
    if any(count != 1 for count in all_leaves.values()):
        raise ValueError("case execution contains duplicate leaves")
    return True


def strict_run(command, env, package, parents, exact_tests=None):
    proc = subprocess.Popen(command, stdout=subprocess.PIPE, text=True, env=env)
    outcomes = {}
    audit = {"test": False, "invalid": False, "packages": set(), "outcomes": [], "no_tests": set()}
    finished = run_tests.finished_tests(proc.stdout, qualified=True, package_outcomes=outcomes, event_audit=audit)
    status = proc.wait()
    starts = audit.get("test_starts", {})
    terminals = audit.get("test_terminals", {})
    roots = {name for pkg, name in starts if pkg == package and "/" not in name}
    if (status or audit["invalid"] or audit["packages"] != {package} or
            audit["outcomes"] != [(package, "pass")] or roots != set(parents) or starts != terminals or
            any(count != 1 for count in starts.values()) or set(finished) != set(starts) or
            any(outcome != "pass" for outcome in finished.values()) or
            (exact_tests is not None and set(starts) != {(package, name) for name in exact_tests})):
        raise ValueError("test run produced incomplete, duplicate or unexpected events")


def run_group(manifest, parent, cases):
    validate_manifest(manifest)
    if not cases or len(cases) != len(set(cases)) or not set(cases) <= set(manifest["parents"].get(parent, [])):
        raise ValueError("case group is not in the discovered manifest")
    qualified = manifest["package"]
    package = "./" + qualified.removeprefix("github.com/lestrrat-3d/decad-gallery/")
    pattern = "^" + re.escape(parent) + "$/^(" + "|".join(map(re.escape, cases)) + ")$"
    expected = [parent + "/" + case for case in cases]
    env = os.environ.copy()
    env.pop("CI_DISCOVER_CASES", None)
    command = ["go", "test", "-json", package, "-parallel=1", "-count=1", "-timeout=10m", "-run", pattern]
    strict_run(command, env, qualified, [parent], [parent, *expected])


def run_remainder(manifest):
    validate_manifest(manifest)
    qualified = manifest["package"]
    package = "./" + qualified.removeprefix("github.com/lestrrat-3d/decad-gallery/")
    env = os.environ.copy()
    env.pop("CI_DISCOVER_CASES", None)
    tests = manifest["tests"]
    parents = set(manifest["parents"])
    if len(tests) != len(set(tests)) or not parents <= set(tests):
        raise ValueError("parent inventory changed")
    remaining = sorted(set(tests) - parents)
    if not remaining:
        raise ValueError("empty remainder requires an unfiltered package run")
    skip = "|".join("^" + re.escape(parent) + "$" for parent in sorted(parents))
    command = ["go", "test", "-json", package, "-parallel=1", "-count=1", "-timeout=20m", "-skip", skip]
    strict_run(command, env, qualified, remaining)


def audit_undiscovered(manifest):
    """Run each parent while skipping its discovered children; reject any other child."""
    validate_manifest(manifest)
    package = manifest["package"]
    relative = "./" + package.removeprefix("github.com/lestrrat-3d/decad-gallery/")
    env = os.environ.copy()
    env.pop("CI_DISCOVER_CASES", None)
    for parent, names in sorted(manifest["parents"].items()):
        skip = "^" + re.escape(parent) + "$/^(" + "|".join(map(re.escape, names)) + ")$"
        command = ["go", "test", "-json", "-parallel=1", "-count=1", "-timeout=4m", "-run", "^" + parent + "$",
                   "-skip", skip, relative]
        result = subprocess.run(command, check=True, capture_output=True, text=True, env=env, timeout=300)
        starts = []
        terminal = []
        package_pass = 0
        for line in result.stdout.splitlines():
            event = json.loads(line)
            if event.get("Package") != package:
                raise ValueError("unexpected audit package")
            test = event.get("Test")
            if test and test != parent:
                raise ValueError("undiscovered child executed")
            if test and event.get("Action") == "run":
                starts.append(test)
            if test and event.get("Action") in {"pass", "fail", "skip"}:
                terminal.append((test, event["Action"]))
            if not test and event.get("Action") == "pass":
                package_pass += 1
        if starts != [parent] or terminal != [(parent, "pass")] or package_pass != 1:
            raise ValueError("case audit is incomplete")


if __name__ == "__main__":
    if sys.argv[1:] == ["plan"]:
        manifest = discover("cycloidal")
        plan = build_plan(manifest) if manifest is not None else None
        matrix = {"include": [{"case": job["case"]} for job in plan["jobs"]]} if plan else {"include": []}
        with open(os.environ["GITHUB_OUTPUT"], "a", encoding="utf-8") as output:
            for name, value in (("plan", plan), ("matrix", matrix)):
                print(name + "=" + json.dumps(value, separators=(",", ":")), file=output)
    elif sys.argv[1:] == ["run"]:
        run_planned(json.loads(os.environ["CASE_PLAN"]), os.environ["CI_TEST_CASE"])
    elif len(sys.argv) == 3 and sys.argv[1] == "--fingerprint":
        print(source_digest(sys.argv[2]))
    else:
        print(json.dumps(discover(sys.argv[1]), sort_keys=True))
