#!/usr/bin/env python3
"""Prepare and check the source-to-Go handoff for the spur gear."""

import argparse
import hashlib
import json
from datetime import datetime, timezone
from pathlib import Path
import shutil
import sys


ROOT = Path(__file__).resolve().parents[2]
INPUTS = (
    "models/spurgear/instructions.md",
    "models/spurgear/steps.md",
    "scripts/modelgen/spurgear_prompt.md",
    "internal/sketchcheck/verify.go",
    "go.mod",
    "go.sum",
)
OUTPUTS = ("involute.go", "spurgear.go", "tooth_section.go")
RECORD = ROOT / "models/spurgear/generation.json"


def digest(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def input_hashes() -> dict[str, str]:
    return {name: digest(ROOT / name) for name in INPUTS}


def read_json(path: Path) -> dict:
    return json.loads(path.read_text(encoding="utf-8"))


def prepare() -> int:
    stamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%S%fZ")
    bundle = ROOT / ".tmp/modelgen/spurgear" / stamp
    for name in INPUTS:
        target = bundle / "inputs" / name
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(ROOT / name, target)
    (bundle / "output").mkdir()
    (bundle / "bundle.json").write_text(
        json.dumps({"inputs": input_hashes(), "outputs": OUTPUTS}, indent=2) + "\n",
        encoding="utf-8",
    )
    print(bundle)
    return 0


def record(bundle: Path) -> int:
    source = read_json(bundle / "bundle.json")
    if source["inputs"] != input_hashes():
        raise ValueError("generator inputs changed after this bundle was prepared")
    if source["outputs"] != list(OUTPUTS):
        raise ValueError("generator output list differs from this workflow")
    for name, expected in source["inputs"].items():
        if digest(bundle / "inputs" / name) != expected:
            raise ValueError(f"bundled input changed after preparation: {name}")
    generated = {}
    for name in OUTPUTS:
        candidate = bundle / "output" / name
        checked_in = ROOT / "models/spurgear" / name
        if digest(candidate) != digest(checked_in):
            raise ValueError(f"{name} differs from the generated candidate")
        generated[name] = digest(candidate)
    RECORD.write_text(
        json.dumps({"inputs": source["inputs"], "outputs": generated}, indent=2) + "\n",
        encoding="utf-8",
    )
    print(RECORD)
    return 0


def check() -> int:
    source = read_json(RECORD)
    if source["inputs"] != input_hashes():
        raise ValueError("spur generator inputs changed; prepare and regenerate the Go files")
    generated = {name: digest(ROOT / "models/spurgear" / name) for name in OUTPUTS}
    if source["outputs"] != generated:
        raise ValueError("spur Go files differ from the recorded generated output")
    print("Spur generation inputs and outputs match the record.")
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("prepare", "record", "check"))
    parser.add_argument("bundle", nargs="?", type=Path)
    args = parser.parse_args()
    try:
        if args.command == "prepare":
            return prepare()
        if args.command == "record":
            if args.bundle is None:
                parser.error("record requires the prepared bundle path")
            return record(args.bundle.resolve())
        return check()
    except (OSError, KeyError, ValueError, json.JSONDecodeError) as exc:
        print(f"spur generation: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
