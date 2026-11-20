#!/usr/bin/env python3
"""Check repository files and the required artifacts for each model and feature."""

from pathlib import Path
import re
import sys


ROOT = Path(__file__).resolve().parent.parent
REQUIRED = ("README.md", "AGENTS.md", "LICENSE", "models/README.md", "docs/model-pipeline.md")
MODEL_FILES = ("README.md", "instructions.md", "steps.md")
# A feature (docs/features.md) has the same files as a model, plus an index image and an exploded view.
FEATURE_FILES = MODEL_FILES + ("images/index.png", "images/exploded.png")
# Models whose printed parts are not yet held to "Corners and roots" (docs/model-pipeline.md).
# Remove a model in the change that adds its corner test.
CORNER_PENDING = {
    "bevelgear", "handcrankfan", "screwgear",
}
# Models that build only purchased or standard parts kept at nominal size.
CORNER_NOMINAL = {"ballbearing", "compressionspring", "fastener"}
# Pending models whose corner test already runs but skips parts that wait on decad. Each must have a corner test, and
# stays in CORNER_PENDING until the test skips nothing.
CORNER_PARTIAL = {"bevelgear", "handcrankfan", "screwgear"}
# Models whose instructions.md states, under Materials, that no part is printed.
CORNER_UNPRINTED = {"spinningwheel"}
# Models and features nobody has built and tested as a physical object yet. Remove one once someone has built it, and
# change its README status line and its models/README.md or features/README.md row in the same change.
UNBUILT = {"clamparm", "compressionspring", "dovetail", "escapement", "handcrankfan", "handcranksiren", "spinningwheel", "spiralspring", "swisslever", "turbinefan"}


def index_rows(path: Path) -> dict[str, str]:
    """Return the Build status of each row of an index README, by the directory its first column links to."""
    rows = {}
    for line in (path.read_text() if path.is_file() else "").splitlines():
        match = re.match(r"\| \[[^\]]+\]\(([^/]+)/\) \|.*\| (Built|Not built yet) \|$", line)
        if match:
            rows[match.group(1)] = match.group(2)
    return rows


def build_status_problems(items: Path, index: Path) -> list[str]:
    """Return the items whose README or index row disagrees with UNBUILT. items is models/ or features/."""
    problems = []
    rows = index_rows(index)
    where = items.relative_to(ROOT)
    for item in sorted(items.iterdir()):
        if not item.is_dir():
            continue
        name = item.name
        built = name not in UNBUILT
        readme = item / "README.md"
        want = "**Build status:** Built and tested." if built else "**Build status:** Not built yet."
        if readme.is_file() and want not in readme.read_text():
            problems.append(f"{where}/{name}/README.md: missing the line starting '{want}'")
        if rows.get(name) != ("Built" if built else "Not built yet"):
            problems.append(f"{index.relative_to(ROOT)}: {name} row needs Build status '{'Built' if built else 'Not built yet'}'")
    return problems


def unbuilt_problems(*roots: Path) -> list[str]:
    """Return the UNBUILT names that no model or feature directory has."""
    return [
        f"UNBUILT names {name}, which has no model or feature directory"
        for name in sorted(UNBUILT)
        if not any((root / name).is_dir() for root in roots)
    ]


def unsorted_rows(path: Path) -> list[str]:
    """Return the names in an index README that are out of alphabetical order."""
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
        if name in CORNER_PARTIAL:
            if name not in CORNER_PENDING:
                problems.append(f"{name} is in CORNER_PARTIAL but not CORNER_PENDING")
            elif not tested:
                problems.append(f"missing: models/{name}/*_test.go (cornercheck)")
        elif name in CORNER_PENDING and tested:
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


def missing_item_files(root: Path, files: tuple[str, ...]) -> list[str]:
    """Return the files that each item directory of root lacks."""
    missing = []
    if not root.is_dir():
        return missing
    for item in sorted(root.iterdir()):
        if item.is_dir():
            for name in files:
                if not (item / name).is_file():
                    missing.append(f"{item.relative_to(ROOT)}/{name}")
            if not (item / "images/hero.png").is_file():
                missing.append(f"{item.relative_to(ROOT)}/images/hero.png")
            if not any(not path.name.endswith("_test.go") for path in item.glob("*.go")):
                missing.append(f"{item.relative_to(ROOT)}/*.go")
            if not any(item.glob("*_test.go")):
                missing.append(f"{item.relative_to(ROOT)}/*_test.go")
            render = item / "render"
            if render.is_dir():
                if not any(not path.name.endswith("_test.go") for path in render.glob("*.go")):
                    missing.append(f"{render.relative_to(ROOT)}/*.go")
                if not any(render.glob("*_test.go")):
                    missing.append(f"{render.relative_to(ROOT)}/*_test.go")
    return missing


def feature_problems(models: Path, features: Path) -> list[str]:
    """Return the problems of features/: a name shared with a model, a missing corner test and the index."""
    problems = []
    if not features.is_dir():
        return problems
    for item in sorted(features.iterdir()):
        if not item.is_dir():
            continue
        if (models / item.name).is_dir():
            problems.append(f"{item.name} exists in both models/ and features/")
        if not has_corner_test(item):
            problems.append(f"missing: features/{item.name}/*_test.go (cornercheck)")
    problems += build_status_problems(features, ROOT / "features/README.md")
    if not (ROOT / "features/README.md").is_file():
        problems.append("missing: features/README.md")
    return problems


def main() -> int:
    missing = [name for name in REQUIRED if not (ROOT / name).is_file()]
    models = ROOT / "models"
    features = ROOT / "features"
    missing += missing_item_files(models, MODEL_FILES)
    missing += missing_item_files(features, FEATURE_FILES)

    problems = unbuilt_problems(models, features)
    if models.is_dir():
        problems += corner_problems(models) + build_status_problems(models, ROOT / "models/README.md")
    problems += feature_problems(models, features)
    if problems:
        for problem in problems:
            print(problem, file=sys.stderr)
        return 1

    for index in (ROOT / "models/README.md", ROOT / "features/README.md"):
        if unsorted_rows(index):
            print(f"{index.relative_to(ROOT)} rows are not in alphabetical order by name", file=sys.stderr)
            return 1

    if missing:
        for name in missing:
            print(f"missing: {name}", file=sys.stderr)
        return 1

    print("Repository structure is valid.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
