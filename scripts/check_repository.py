#!/usr/bin/env python3
"""Check the repository files and per-model README convention."""

from pathlib import Path
import sys


ROOT = Path(__file__).resolve().parent.parent
REQUIRED = ("README.md", "AGENTS.md", "LICENSE", "models/README.md")


def main() -> int:
    missing = [name for name in REQUIRED if not (ROOT / name).is_file()]
    models = ROOT / "models"
    if models.is_dir():
        for item in sorted(models.iterdir()):
            if item.is_dir() and not (item / "README.md").is_file():
                missing.append(f"{item.relative_to(ROOT)}/README.md")

    if missing:
        for name in missing:
            print(f"missing: {name}", file=sys.stderr)
        return 1

    print("Repository structure is valid.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
