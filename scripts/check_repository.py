#!/usr/bin/env python3
"""Check repository files and the required artifacts for each model."""

from pathlib import Path
import re
import sys


ROOT = Path(__file__).resolve().parent.parent
REQUIRED = ("README.md", "AGENTS.md", "LICENSE", "models/README.md", "docs/model-pipeline.md")
MODEL_FILES = ("README.md", "instructions.md", "steps.md")
# Models whose printed parts are not yet held to "Corners and roots" (docs/model-pipeline.md).
# Remove a model in the change that adds its corner test.
CORNER_PENDING = {
    "bevelgear", "cycloidal", "handcrankfan",
    "planetarygearbox", "screwgear",
}
# Models that build only purchased or standard parts kept at nominal size.
CORNER_NOMINAL = {"ballbearing", "m2bolt", "m2nut"}
# Models whose instructions.md states, under Materials, that no part is printed.
CORNER_UNPRINTED = {"spinningwheel"}


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


def has_corner_test(model: Path) -> bool:
    """Return whether a test file of the model calls the corner check."""
    return any("cornercheck." in path.read_text() for path in model.glob("*_test.go"))


def corner_problems(models: Path) -> list[str]:
    """Return the models that break the corners and roots enforcement."""
    names = {item.name for item in models.iterdir() if item.is_dir()}
    problems = []
    for name in sorted(CORNER_PENDING | CORNER_NOMINAL | CORNER_UNPRINTED):
        if name not in names:
            problems.append(f"corner list names {name}, which has no model directory")
    for name in sorted(names):
        tested = has_corner_test(models / name)
        if name in CORNER_PENDING and tested:
            problems.append(f"remove {name} from CORNER_PENDING")
        elif name in CORNER_UNPRINTED:
            if tested:
                problems.append(f"remove {name} from CORNER_UNPRINTED")
            instructions = models / name / "instructions.md"
            if not instructions.is_file() or not re.search(
                r"^## Materials\s*$", instructions.read_text(), re.MULTILINE
            ):
                problems.append(f"models/{name}/instructions.md: missing ## Materials section")
        elif name not in CORNER_PENDING | CORNER_NOMINAL and not tested:
            problems.append(f"missing: models/{name}/*_test.go (cornercheck)")
    return problems


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

    if models.is_dir():
        problems = corner_problems(models)
        if problems:
            for problem in problems:
                print(problem, file=sys.stderr)
            return 1

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
