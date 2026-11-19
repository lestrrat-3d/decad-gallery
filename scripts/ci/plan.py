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
INTERNAL = set("clamparmcapacity hexnut edgefinish cornercheck m5fastener gearroot fastenertool m2fastener splitcheck meshgap "
               "placement rootcheck sketchcheck".split())
FEATURES = {"dovetail"}
CI_FILES = {".github/workflows/repository.yml", ".github/workflows/ci-probe.yml",
            ".github/workflows/fan-variants.yml", ".github/workflows/update-decad.yml"} | {
    "scripts/ci/" + name + suffix
    for name in ("full_partition", "gate", "inventory", "plan", "probe", "reconcile_records", "replay",
                 "run_tests", "select_models")
    for suffix in (".py",)
} | {
    "scripts/ci/test_" + name + ".py"
    for name in ("full_partition", "gate", "inventory", "plan", "probe", "reconcile_records", "replay",
                 "run_tests", "select_models")
} | {"scripts/ci/test_workflow_records.py", "scripts/ci/testdiff.go", "scripts/ci/test_testdiff.py",
     "scripts/ci/check_isolated_tests.py"}
HEAVY = {"bevelgear": "bevel-pr", "escapement": "escapement", "onewayclutch": "onewayclutch",
         "clamparm": "clamparm", "turbinefan": "turbinefan", "spinningwheel": "wheel",
         "handcranksiren": "siren"}
RENDER = {"ballbearing": ("bearing-render",), "bevelgear": ("bevel-render",),
          "clamparm": ("clamparm-render",), "escapement": ("escapement-exploded",),
          "handcrankfan": ("fan-motion", "fan-view"), "handcranksiren": ("siren-render",),
          "planetarygearbox": ("planetary-motion", "planetary-render"),
          "spinningwheel": ("wheel-motion", "wheel-parts", "wheel-block"),
          "swisslever": ("swiss-render",)}
BROAD_WORKERS = {"bevel-pr", "clamparm", "escapement", "onewayclutch", "turbinefan", "wheel", "siren"}
BROAD_PACKAGES = {"package-" + name for name in
                  ("meshgap", "gearroot", "edgefinish", "m2fastener", "cornercheck", "splitcheck")}
BROAD_RENDER = {"planetary-motion", "wheel-motion", "wheel-parts", "fan-motion", "escapement-exploded"}


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
        if not owner or owner not in after:
            raise ValueError("deleted or unknown package: " + path)
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
    return selected, production


def isolated_tests(base, tested, changed):
    """Select changed tests only when both revisions opt in and helpers are unchanged."""
    code = [path for path in changed if path.endswith(".go") and path not in CI_FILES]
    if len(code) != 1:
        return []
    path = code[0]
    parts = Path(path).parts
    if len(parts) != 3 or parts[0] != "models" or parts[1] not in MODELS or not path.endswith("_test.go"):
        return []
    try:
        old = subprocess.check_output(["git", "show", f"{base}:{path}"], stderr=subprocess.PIPE)
        new = subprocess.check_output(["git", "show", f"{tested}:{path}"], stderr=subprocess.PIPE)
        if b"// CI: isolated-tests" not in old or b"// CI: isolated-tests" not in new:
            return []
        with tempfile.TemporaryDirectory(dir=".tmp") as directory:
            root = Path(directory)
            (root / "old.go").write_bytes(old)
            (root / "new.go").write_bytes(new)
            binary = root / "testdiff"
            subprocess.run(["go", "build", "-o", str(binary), "scripts/ci/testdiff.go"], check=True,
                           stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            names = json.loads(subprocess.check_output([str(binary), str(root / "old.go"), str(root / "new.go")]))
        if not isinstance(names, list) or not names or any(not re.fullmatch(r"Test[A-Za-z0-9_]+", n) for n in names):
            return []
        return ["isolated-" + parts[1] + "-" + name for name in names]
    except (OSError, ValueError, subprocess.CalledProcessError):
        return []


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
        if all(is_presentation(path) or path in CI_FILES for path in changed):
            suites = {}
            if wheel_parts:
                suites["wheel-parts"] = ["check changed wheel part documentation"]
            result.update(suites=sorted(suites), reasons=suites, reason="presentation or CI-only change")
            return result
        if not wheel_parts and all(is_presentation(path) or path in CI_FILES or path.endswith("_test.go")
                                   for path in changed):
            isolated = isolated_tests(base, tested, changed)
            if isolated:
                reason = "changed isolated test functions"
                result.update(suites=isolated, reasons={suite: [reason] for suite in isolated}, reason=reason)
                return result
        code = [path for path in changed if path.endswith(".go") and path not in CI_FILES]
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
        new_suites = {}
        if new_packages:
            for name in sorted(new_packages):
                directory = after[name]["dir"]
                parts = directory.split("/")
                if len(parts) != 2 or parts[0] not in {"models", "internal", "features"} or not parts[1].isalnum():
                    raise RuntimeError("new package cannot be dispatched: " + directory)
                prefix = {"models": "new-model-", "internal": "new-package-", "features": "new-feature-"}[parts[0]]
                registry = {"models": MODELS, "internal": INTERNAL, "features": FEATURES}[parts[0]]
                if parts[1] not in registry or name != ROOT + "/" + directory:
                    new_suites[name] = prefix + parts[1]
        if set(before) - set(after):
            reason = "deleted package requires full coverage"
            suites = {"broad": [reason], "full": [reason]}
            result.update(suites=sorted(suites), reasons=suites, reason=reason)
            return result
        unknown_input = any(not is_presentation(path) and not path.endswith(".go") and path not in CI_FILES
                            for path in changed)
        owners, production = affected_packages(code, before, after)
        owners.update(new_packages)
        suites = {}
        if new_suites:
            reason = "new packages require broad coverage and direct tests"
            suites["broad"] = [reason]
            for suite in new_suites.values():
                suites[suite] = [reason]
        if unknown_input:
            suites.setdefault("broad", []).append("unregistered or shared input changed")
        if wheel_parts:
            suites["wheel-parts"] = ["check changed wheel part documentation"]
        model_owners = set()
        for owner in sorted(owners):
            if owner == ROOT + "/tools/render":
                continue
            if owner in new_suites:
                continue
            if owner.startswith(ROOT + "/models/") and owner.rsplit("/", 1)[-1] in MODELS:
                name = owner.rsplit("/", 1)[-1]
                if owner in production:
                    model_owners.add(name)
                suite = HEAVY.get(name, "model-" + name)
                if name == "cycloidal":
                    suite = "cycloidal-pr"
            elif owner.startswith(ROOT + "/internal/") and owner.rsplit("/", 1)[-1] in INTERNAL:
                suite = "package-" + owner.rsplit("/", 1)[-1]
            elif owner.startswith(ROOT + "/features/") and owner.rsplit("/", 1)[-1] in FEATURES:
                suite = "feature-" + owner.rsplit("/", 1)[-1]
            else:
                reason = "unregistered package requires full coverage: " + owner
                suites.setdefault("broad", []).append(reason)
                suites.setdefault("full", []).append(reason)
                continue
            suites.setdefault(suite, []).append("affected package " + owner)
        if any(path.startswith("tools/render/") for path in code):
            suites["renderer-all"] = ["renderer source or tests changed"]
            if unknown_input:
                suites.setdefault("full", []).append("renderer source and unregistered input changed")
        if model_owners or ROOT + "/tools/render" in production:
            suites["renderer-compile"] = ["compile renderer against changed model APIs"]
            for name in sorted(model_owners & RENDER.keys()):
                for suite in RENDER[name]:
                    suites.setdefault(suite, []).append("renderer check for " + name)
        if "full" in suites or "renderer-all" in suites:
            covered = {suite for checks in RENDER.values() for suite in checks} | {"renderer-compile"}
            if "full" in suites:
                covered.add("renderer-all")
            for suite in covered:
                suites.pop(suite, None)
        if "full" in suites:
            suites = {suite: reasons for suite, reasons in suites.items()
                      if suite in {"broad", "full"}}
        elif "broad" in suites:
            covered = BROAD_WORKERS | BROAD_PACKAGES
            if "renderer-all" not in suites:
                covered |= BROAD_RENDER
            for suite in covered:
                suites.pop(suite, None)
        result.update(suites=sorted(suites), reasons=suites, reason="affected packages selected")
    except (subprocess.CalledProcessError, ValueError, OSError, KeyError, tarfile.TarError) as exc:
        reason = "conservative fallback: " + str(exc)
        result.update(suites=["broad", "full"], reason=reason, reasons={"broad": [reason], "full": [reason]})
    if ("broad" in result["suites"] and "full" not in result["suites"] and
            "renderer-all" not in result["suites"] and
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
    if event in {"pull_request", "push"} and os.environ.get("CI_BROAD_OVERRIDE") == "true":
        reason = "CI_BROAD_OVERRIDE selects broad PR/push coverage"
        suites = ["broad"] + [suite for suite in result["suites"] if suite != "broad"]
        result.update(suites=suites, reasons={suite: ([reason] if suite == "broad" else result["reasons"][suite])
                                             for suite in suites}, reason=reason)
    encoded = json.dumps(result, sort_keys=True, separators=(",", ":"))
    print(encoded)
    if os.environ.get("GITHUB_OUTPUT"):
        with open(os.environ["GITHUB_OUTPUT"], "a") as output:
            print("pilot=" + str(result["suites"] == ["cycloidal-pr"]).lower(), file=output)
            print("plan=" + encoded, file=output)


if __name__ == "__main__":
    main()
