#!/usr/bin/env python3
"""Resolve one published Decad version and update both Gallery Go modules."""

import argparse
import json
import os
from pathlib import Path
import subprocess
import sys


MODULE = "github.com/lestrrat-3d/decad"
MODULE_ROOTS = (".", "tools/render")


def run_go(directory, *arguments, capture=False, direct=False):
    environment = os.environ.copy()
    environment["GOWORK"] = "off"
    if direct:
        environment["GOPROXY"] = "direct"
    result = subprocess.run(
        ["go", *arguments],
        cwd=directory,
        env=environment,
        check=True,
        text=True,
        stdout=subprocess.PIPE if capture else None,
    )
    return json.loads(result.stdout) if capture else None


def module_manifest(directory):
    manifest = run_go(directory, "mod", "edit", "-json", capture=True)
    if any(entry["Old"]["Path"] == MODULE for entry in manifest.get("Replace") or []):
        raise ValueError(f"{directory}/go.mod replaces Decad; remove the local override before updating")
    requirements = {entry["Path"]: entry["Version"] for entry in manifest.get("Require") or []}
    if MODULE not in requirements:
        raise ValueError(f"{directory}/go.mod does not require {MODULE}")
    return requirements


def update(project, revision):
    directories = [project / relative for relative in MODULE_ROOTS]
    paths = [directory / filename for directory in directories for filename in ("go.mod", "go.sum")]
    original = {path: path.read_bytes() if path.exists() else None for path in paths}
    try:
        for directory in directories:
            module_manifest(directory)
        resolved = run_go(
            project, "list", "-mod=readonly", "-m", "-json", f"{MODULE}@{revision}", capture=True, direct=True
        )
        version = resolved.get("Version")
        if resolved.get("Path") != MODULE or not version or resolved.get("Replace") or resolved.get("Error"):
            raise ValueError("Go did not resolve a published Decad version")
        for directory in directories:
            run_go(directory, "get", f"{MODULE}@{version}")
            run_go(directory, "mod", "tidy")
        selected = {}
        for relative, directory in zip(MODULE_ROOTS, directories):
            requirements = module_manifest(directory)
            actual = run_go(directory, "list", "-mod=readonly", "-m", "-json", MODULE, capture=True)
            if requirements[MODULE] != version or actual.get("Version") != version or actual.get("Replace"):
                raise ValueError(f"{directory} does not select published Decad {version}")
            sums = (directory / "go.sum").read_text().splitlines()
            for suffix in ("", "/go.mod"):
                if not any(line.startswith(f"{MODULE} {version}{suffix} h1:") for line in sums):
                    raise ValueError(f"{directory}/go.sum has no checksum for Decad {version}{suffix}")
            selected[relative] = actual["Version"]
    except BaseException:
        for path, contents in original.items():
            if contents is None:
                path.unlink(missing_ok=True)
            else:
                path.write_bytes(contents)
        raise

    return {
        "module": MODULE,
        "revision": revision,
        "version": version,
        "origin": resolved.get("Origin"),
        "selected": selected,
        "changed": [str(path.relative_to(project)) for path in paths if path.read_bytes() != original[path]],
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--revision", default="main", help="published branch, version or commit; defaults to main")
    arguments = parser.parse_args()
    project = Path(__file__).resolve().parent.parent
    try:
        result = update(project, arguments.revision)
    except (OSError, ValueError, subprocess.CalledProcessError) as error:
        print(f"Decad update failed: {error}", file=sys.stderr)
        return 1
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
