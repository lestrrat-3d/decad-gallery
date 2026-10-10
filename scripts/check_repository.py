#!/usr/bin/env python3
"""Check repository files and the required artifacts for each model."""

from pathlib import Path
import sys


ROOT = Path(__file__).resolve().parent.parent
REQUIRED = ("README.md", "AGENTS.md", "LICENSE", "models/README.md", "docs/model-pipeline.md")
MODEL_FILES = ("README.md", "instructions.md", "steps.md")


def main() -> int:
    missing = [name for name in REQUIRED if not (ROOT / name).is_file()]
    models = ROOT / "models"
    if models.is_dir():
        for item in sorted(models.iterdir()):
            if item.is_dir():
                for name in MODEL_FILES:
                    if not (item / name).is_file():
                        missing.append(f"{item.relative_to(ROOT)}/{name}")
                if not (item / "images/hero.png").is_file():
                    missing.append(f"{item.relative_to(ROOT)}/images/hero.png")
                if not any(not path.name.endswith("_test.go") for path in item.glob("*.go")):
                    missing.append(f"{item.relative_to(ROOT)}/*.go")
                if not any(item.glob("*_test.go")):
                    missing.append(f"{item.relative_to(ROOT)}/*_test.go")

    if missing:
        for name in missing:
            print(f"missing: {name}", file=sys.stderr)
        return 1

    print("Repository structure is valid.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
