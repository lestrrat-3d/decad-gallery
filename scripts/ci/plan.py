#!/usr/bin/env python3
"""Plan CI work from a verified diff, falling back when selection is uncertain."""
import io
import json
import os
from pathlib import Path
import re
import subprocess
import tarfile
import tempfile

ROOT = "github.com/lestrrat-3d/decad-gallery"
MODELS = set("ballbearing bevelgear clamparm crankhandle cycloidal escapement handcrankfan handcranksiren "
             "helicalgear herringbonegear m2bolt m2nut m5bolt m5nut onewayclutch pinring "
             "planetarygearbox plaquestand screwgear spinningwheel spiralspring spurgear swisslever turbinefan".split())
INTERNAL = set("hexnut edgefinish cornercheck m5fastener gearroot fastenertool m2fastener splitcheck meshgap "
               "placement rootcheck sketchcheck".split())
FEATURES = {"dovetail"}
HEAVY = {"bevelgear": "bevel-pr", "escapement": "escapement", "onewayclutch": "onewayclutch",
         "clamparm": "clamparm", "turbinefan": "turbinefan", "spinningwheel": "wheel",
         "handcranksiren": "siren"}
RENDER = {"ballbearing": ("bearing-render",), "bevelgear": ("bevel-render",),
          "clamparm": ("clamparm-render",), "escapement": ("escapement-exploded",),
          "handcrankfan": ("fan-motion", "fan-view"), "handcranksiren": ("siren-render",),
          "planetarygearbox": ("planetary-motion", "planetary-render"),
          "spinningwheel": ("wheel-motion", "wheel-parts", "wheel-block"),
          "swisslever": ("swiss-render",)}


def check_wheel_inventory(root=Path(".")):
    """Check documented part images without building the wheel."""
    wheel = root / "models/spinningwheel"
    readme = (wheel / "README.md").read_text()
    references = re.findall(r'<img src="images/parts/([^"/]+)"', readme)
    files = {p.name for p in (wheel / "images/parts").glob("*.png")}
    if not files or len(references) != len(set(references)) or set(references) != files:
        raise ValueError("spinning wheel README part rows do not match the part-image inventory")


def git(*args):
    return subprocess.check_output(["git", *args], stderr=subprocess.PIPE).strip()


def metadata(revision):
    """Read package and test imports from both modules at one revision."""
    archive = subprocess.check_output(["git", "archive", revision], stderr=subprocess.PIPE)
    Path(".tmp").mkdir(exist_ok=True)
    with tempfile.TemporaryDirectory(dir=".tmp") as tmp:
        root = Path(tmp).resolve()
        with tarfile.open(fileobj=io.BytesIO(archive)) as tree:
            tree.extractall(root, filter="data")
        packages = {}
        env = dict(os.environ, GOFLAGS="-mod=readonly", GOWORK="off")
        for module in (".", "tools/render"):
            data = subprocess.check_output(["go", "list", "-json", "./..."], cwd=root / module,
                                           env=env, stderr=subprocess.PIPE).decode()
            decoder = json.JSONDecoder()
            while data.strip():
                item, end = decoder.raw_decode(data.lstrip())
                data = data.lstrip()[end:]
                directory = Path(item["Dir"]).relative_to(root).as_posix()
                packages[item["ImportPath"]] = {
                    "dir": directory,
                    "imports": set(item.get("Imports", [])),
                    "test_imports": set(item.get("TestImports", []) + item.get("XTestImports", [])),
                }
        return packages


def affected_packages(changed, before, after):
    """Return test owners reached by production imports, stopping at test edges."""
    graph = {}
    directories = {}
    for snapshot in (before, after):
        for name, item in snapshot.items():
            merged = graph.setdefault(name, {"imports": set(), "test_imports": set()})
            merged["imports"].update(item["imports"])
            merged["test_imports"].update(item["test_imports"])
            directories[item["dir"]] = name
    selected, production = set(), set()
    for path in changed:
        owner = directories.get(str(Path(path).parent))
        if not owner or owner not in before or owner not in after:
            raise ValueError("new, deleted, or unknown package: " + path)
        selected.add(owner)
        if not path.endswith("_test.go"):
            production.add(owner)
    while True:
        more = {name for name, item in graph.items() if item["imports"] & production} - production
        if not more:
            break
        production.update(more)
    selected.update(production)
    selected.update(name for name, item in graph.items() if item["test_imports"] & production)
    return selected


def plan(event, base, tested):
    result = {"version": 1, "tested": tested, "base": base, "changed_paths": [],
              "suites": ["broad"], "reasons": {"broad": ["event requires broad coverage"]},
              "reason": "event requires broad coverage"}
    if event not in {"pull_request", "push"}:
        return result
    try:
        head = git("rev-parse", "HEAD").decode()
        tested = git("rev-parse", "--verify", tested + "^{commit}").decode()
        base = git("rev-parse", "--verify", base + "^{commit}").decode()
        if head != tested:
            raise ValueError("tested revision differs from checkout")
        if event == "pull_request":
            parents = git("rev-list", "--parents", "-n", "1", tested).decode().split()
            if len(parents) != 3 or parents[1] != base:
                raise ValueError("pull request merge base parent is not verified")
        paths = subprocess.check_output(["git", "diff", "--no-renames", "--name-only", "-z", base, tested])
        changed = [os.fsdecode(path) for path in paths.split(b"\0") if path]
        result.update(tested=tested, base=base, changed_paths=changed)
        dependency_files = {"go.mod", "go.sum", "tools/render/go.mod", "tools/render/go.sum"}
        if dependency_files.intersection(changed):
            reason = "module dependency files changed; full coverage required"
            result.update(suites=["broad", "full"], reasons={"broad": [reason], "full": [reason]}, reason=reason)
            return result
        wheel_parts = any(path.startswith("models/spinningwheel/images/parts/") and
                          Path(path).suffix.lower() == ".png" for path in changed)
        def is_presentation(path):
            return (path.lower().endswith(".md") or
                    (path.startswith("models/spinningwheel/images/parts/") and
                     Path(path).suffix.lower() == ".png"))
        if all(is_presentation(path) for path in changed):
            suites = {}
            if wheel_parts:
                suites["wheel-parts"] = ["check changed wheel part documentation"]
            result.update(suites=sorted(suites), reasons=suites, reason="presentation-only change")
            return result
        code = [path for path in changed if path.endswith(".go")]
        try:
            before, after = metadata(base), metadata(tested)
        except (subprocess.CalledProcessError, ValueError, OSError, KeyError, tarfile.TarError) as exc:
            known = ({f"models/{name}" for name in MODELS} | {f"internal/{name}" for name in INTERNAL}
                     | {f"features/{name}" for name in FEATURES})
            known.add("tools/render")
            if any(str(Path(path).parent) not in known for path in code):
                raise RuntimeError("package metadata unavailable for an unregistered Go directory") from exc
            raise
        new_packages = set(after) - set(before)
        if new_packages:
            new_suites = []
            for name in sorted(new_packages):
                directory = after[name]["dir"]
                parts = directory.split("/")
                if len(parts) != 2 or parts[0] not in {"models", "internal", "features"} or not parts[1].isalnum():
                    raise RuntimeError("new package cannot be dispatched: " + directory)
                prefix = {"models": "new-model-", "internal": "new-package-", "features": "new-feature-"}[parts[0]]
                new_suites.append(prefix + parts[1])
            reason = "new packages require broad coverage and direct tests"
            suites = ["broad", *new_suites]
            if any(path.startswith("tools/render/") for path in code):
                suites.append("renderer-all")
            result.update(suites=suites, reasons={suite: [reason] for suite in suites}, reason=reason)
            return result
        if any(not is_presentation(path) and not path.endswith(".go") for path in changed):
            raise ValueError("unregistered or shared input changed")
        owners = affected_packages(code, before, after)
        suites = {}
        if wheel_parts:
            suites["wheel-parts"] = ["check changed wheel part documentation"]
        model_owners = set()
        for owner in sorted(owners):
            if owner == ROOT + "/tools/render":
                continue
            if owner.startswith(ROOT + "/models/") and owner.rsplit("/", 1)[-1] in MODELS:
                name = owner.rsplit("/", 1)[-1]
                model_owners.add(name)
                suite = HEAVY.get(name, "model-" + name)
                if name == "cycloidal":
                    suite = "cycloidal-pr"
            elif owner.startswith(ROOT + "/internal/") and owner.rsplit("/", 1)[-1] in INTERNAL:
                suite = "package-" + owner.rsplit("/", 1)[-1]
            elif owner.startswith(ROOT + "/features/") and owner.rsplit("/", 1)[-1] in FEATURES:
                suite = "feature-" + owner.rsplit("/", 1)[-1]
            else:
                raise ValueError("unregistered package: " + owner)
            suites.setdefault(suite, []).append("affected package " + owner)
        if any(path.startswith("tools/render/") for path in code):
            raise ValueError("renderer source requires broad coverage")
        if any(not path.endswith("_test.go") for path in code) and (
                model_owners or ROOT + "/tools/render" in owners):
            suites["renderer-compile"] = ["compile renderer against changed model APIs"]
            for name in sorted(model_owners & RENDER.keys()):
                for suite in RENDER[name]:
                    suites.setdefault(suite, []).append("renderer check for " + name)
        result.update(suites=sorted(suites), reasons=suites, reason="affected packages selected")
    except (subprocess.CalledProcessError, ValueError, OSError, KeyError, tarfile.TarError) as exc:
        reason = "conservative fallback: " + str(exc)
        result.update(reason=reason, reasons={"broad": [reason]})
    if ("broad" in result["suites"] and
            any(path.startswith("tools/render/") and path.endswith(".go") for path in result["changed_paths"])):
        result["suites"].append("renderer-all")
        result["reasons"]["renderer-all"] = ["renderer source or tests changed"]
    return result


def main():
    check_wheel_inventory()
    event_path = os.environ.get("GITHUB_EVENT_PATH")
    event_data = json.loads(Path(event_path).read_text()) if event_path else {}
    event = os.environ.get("GITHUB_EVENT_NAME", "")
    tested = os.environ.get("GITHUB_SHA", "HEAD")
    base = (event_data.get("pull_request", {}).get("base", {}).get("sha", "")
            if event == "pull_request" else event_data.get("before", ""))
    result = plan(event, base, tested)
    encoded = json.dumps(result, sort_keys=True, separators=(",", ":"))
    print(encoded)
    if os.environ.get("GITHUB_OUTPUT"):
        with open(os.environ["GITHUB_OUTPUT"], "a") as output:
            print("pilot=" + str(result["suites"] == ["cycloidal-pr"]).lower(), file=output)
            print("plan=" + encoded, file=output)


if __name__ == "__main__":
    main()
