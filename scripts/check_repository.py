#!/usr/bin/env python3
"""Check repository files and the required artifacts for each model."""

from pathlib import Path
import re
import sys


ROOT = Path(__file__).resolve().parent.parent
REQUIRED = ("README.md", "AGENTS.md", "LICENSE", "models/README.md", "docs/model-pipeline.md")
MODEL_FILES = ("README.md", "instructions.md", "steps.md")


def unsorted_model_rows() -> list[str]:
    """Return the model names in models/README.md that are out of alphabetical order."""
    path = ROOT / "models/README.md"
    if not path.is_file():
        return []
    names = []
    for line in path.read_text().splitlines():
        match = re.match(r"\| \[([^\]]+)\]", line)
        if match:
            names.append(match.group(1))
    return [a for a, b in zip(names, sorted(names, key=str.lower)) if a != b]


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

    unsorted = unsorted_model_rows()
    if unsorted:
        print("models/README.md rows are not in alphabetical order by model name", file=sys.stderr)
        return 1

    if missing:
        for name in missing:
            print(f"missing: {name}", file=sys.stderr)
        return 1

    print("Repository structure is valid.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
