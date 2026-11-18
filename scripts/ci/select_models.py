#!/usr/bin/env python3
"""Select dedicated model checks for the commit tested by a PR workflow."""

import json
import os
from pathlib import Path
import subprocess
import sys


MODULE = "github.com/lestrrat-3d/decad-gallery"
CHECKS = {
    "planetary-motion": "planetarygearbox",
    "wheel-motion": "spinningwheel",
    "fan-motion": "handcrankfan",
    "escapement": "escapement",
    "escapement-exploded": "escapement",
    "onewayclutch": "onewayclutch",
    "clamparm": "clamparm",
    "turbinefan": "turbinefan",
    "wheel": "spinningwheel",
    "siren": "handcranksiren",
    "bevel-pr": "bevelgear",
}
MODELS = set(CHECKS.values())
REPOSITORY_ONLY_PATHS = {
    "README.md",
    "models/README.md",
    "docs/images/gallery.png",
    "scripts/check_repository.py",
}


def run(*args):
    return subprocess.check_output(args, stderr=subprocess.PIPE)


def changed_models(raw, known):
    """Return changed model names; reject changes the selector cannot classify."""
    fields = raw.split(b"\0")
    if fields[-1] != b"" or len(fields) % 2 != 1:
        raise ValueError("invalid git diff output")
    changed = set()
    for metadata, path in zip(fields[0:-1:2], fields[1:-1:2]):
        parts = metadata.decode("ascii").split()
        if len(parts) != 5 or parts[0] not in (":100644", ":100755"):
            raise ValueError("unsupported diff metadata")
        if parts[1] != parts[0][1:] or parts[4] != "M":
            raise ValueError("addition, deletion, rename, or mode change")
        name = path.decode("utf-8")
        if name in REPOSITORY_ONLY_PATHS:
            continue
        path_parts = name.split("/")
        if len(path_parts) < 3 or path_parts[0] != "models" or path_parts[1] not in known:
            raise ValueError("shared or unknown path: " + name)
        changed.add(path_parts[1])
    if not changed:
        raise ValueError("no changed model")
    return changed


def packages_from_json(data):
    decoder = json.JSONDecoder()
    packages = []
    pos = 0
    while pos < len(data):
        while pos < len(data) and data[pos].isspace():
            pos += 1
        if pos == len(data):
            break
        package, pos = decoder.raw_decode(data, pos)
        packages.append(package)
    if not packages:
        raise ValueError("empty Go package list")
    return packages


def affected_models(packages, changed):
    by_name = {p["ImportPath"]: p for p in packages}
    prefix = MODULE + "/models/"
    model_packages = {}
    for name in by_name:
        if name.startswith(prefix):
            model = name[len(prefix):].split("/")[0]
            model_packages.setdefault(model, set()).add(name)
    if not changed <= model_packages.keys() or not MODELS <= model_packages.keys():
        raise ValueError("missing model package")
    reverse = {}
    for name, package in by_name.items():
        for key in ("Imports", "TestImports", "XTestImports"):
            for dependency in package.get(key, []):
                if dependency.startswith(MODULE + "/") and dependency not in by_name:
                    raise ValueError("missing local dependency: " + dependency)
                reverse.setdefault(dependency, set()).add(name)
    affected = set().union(*(model_packages[model] for model in changed))
    pending = list(affected)
    while pending:
        for consumer in reverse.get(pending.pop(), set()) - affected:
            affected.add(consumer)
            pending.append(consumer)
    return {model for model, names in model_packages.items() if names & affected}


def select():
    if os.environ.get("GITHUB_EVENT_NAME") != "pull_request":
        return MODELS, "Non-PR event runs all dedicated checks."
    base = os.environ["PR_BASE_SHA"]
    head = os.environ["PR_HEAD_SHA"]
    parents = run("git", "rev-list", "--parents", "-n", "1", "HEAD").decode().split()
    if len(parents) != 3 or parents[1:] != [base, head]:
        raise ValueError("tested merge parents do not match PR base and head")
    run("git", "merge-base", base, head)
    diff = run("git", "diff", "--raw", "--no-renames", "-z", base, "HEAD", "--")
    packages = packages_from_json(run("go", "list", "-json", "./...").decode())
    known = {
        p["ImportPath"][len(MODULE + "/models/"):].split("/")[0]
        for p in packages if p["ImportPath"].startswith(MODULE + "/models/")
    }
    changed = changed_models(diff, known)
    affected = affected_models(packages, changed)
    return MODELS & affected, "Changed models: " + ", ".join(sorted(changed))


def main():
    try:
        selected, reason = select()
    except (KeyError, ValueError, OSError, subprocess.CalledProcessError) as error:
        selected = MODELS
        reason = "Running all dedicated checks because selection failed: " + str(error)
    print(reason)
    print("Selected models: " + ", ".join(sorted(selected)))
    with Path(os.environ["GITHUB_OUTPUT"]).open("a", encoding="utf-8") as output:
        output.write("models=" + json.dumps(sorted(selected), separators=(",", ":")) + "\n")
        output.write("reason=" + reason.replace("\n", " ") + "\n")


def summarize():
    needs = json.loads(os.environ["JOB_RESULTS"])
    selector = needs["select"]
    print(selector["outputs"].get("reason", "Selection did not complete."))
    print("Selected models:", selector["outputs"].get("models", "unavailable"))
    errors = []
    if selector["result"] != "success":
        errors.append("selector: " + selector["result"])
        selected = MODELS
    else:
        selected = set(json.loads(selector["outputs"]["models"]))
        if not selected <= MODELS:
            errors.append("selector returned unknown models")
    is_pr_or_push = os.environ["GITHUB_EVENT_NAME"] in ("pull_request", "push")
    expected = {"repository": True, "bevel-full": not is_pr_or_push}
    expected.update({job: model in selected for job, model in CHECKS.items()})
    expected["bevel-pr"] &= is_pr_or_push
    for job, should_run in expected.items():
        result = needs[job]["result"]
        print(f"{job}: {result} (expected {'success' if should_run else 'skipped'})")
        if result != ("success" if should_run else "skipped"):
            errors.append(job + ": " + result)
    if errors:
        print("Unexpected results: " + ", ".join(errors), file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    if sys.argv[1:] == ["--summary"]:
        summarize()
    else:
        main()
