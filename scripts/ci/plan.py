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
MODELS = set("ballbearing bevelgear clamparm compressionspring crankhandle cycloidal escapement handcrankfan handcranksiren "
             "fastener helicalgear herringbonegear onewayclutch pinring "
             "planetarygearbox plaquestand screwgear sevensegment spinningwheel spiralspring spurgear swisslever turbinefan".split())
INTERNAL = set("clamparmcapacity hexnut edgefinish cornercheck m5fastener gearroot fastenertool m2fastener splitcheck meshgap "
               "placement rootcheck sketchcheck renderkit".split())
FEATURES = {"dovetail"}
TEST_PACKAGE_DIRS = ({"models/" + name for name in MODELS} | {"internal/" + name for name in INTERNAL} |
                    {"features/" + name for name in FEATURES} | {"tools/render"} |
                    {"models/" + name + "/render" for name in MODELS} |
                    {"features/" + name + "/render" for name in FEATURES})
R_CORE = ROOT + "/internal/renderkit"
R_TOOL = ROOT + "/tools/render"
CI_FILES = {".github/workflows/repository.yml", ".github/workflows/ci-probe.yml",
            ".github/workflows/fan-variants.yml", ".github/workflows/update-decad.yml"} | {
    "scripts/ci/" + name + suffix
    for name in ("full_partition", "gate", "inventory", "plan", "probe", "reconcile_records", "replay", "reuse_main",
                 "run_tests", "select_models")
    for suffix in (".py",)
} | {
    "scripts/ci/test_" + name + ".py"
    for name in ("full_partition", "gate", "inventory", "plan", "probe", "reconcile_records", "replay", "reuse_main",
                 "run_tests", "select_models")
} | {"scripts/ci/test_workflow_records.py", "scripts/ci/testdiff.go", "scripts/ci/test_testdiff.py",
     "scripts/ci/check_isolated_tests.py", "scripts/render_compare.py", "scripts/test_render_compare.py",
     "scripts/check_repository.py", "scripts/modelgen/spurgear.py"}
HEAVY = {"bevelgear": "bevel-pr", "escapement": "escapement", "onewayclutch": "onewayclutch",
         "clamparm": "clamparm", "sevensegment": "sevensegment", "turbinefan": "turbinefan", "spinningwheel": "wheel",
         "handcranksiren": "siren"}
RENDER = {"ballbearing": ("bearing-render",), "bevelgear": ("bevel-render",),
          "clamparm": ("clamparm-render",), "escapement": ("escapement-exploded",),
          "handcrankfan": ("fan-motion", "fan-view"), "handcranksiren": ("siren-render",),
          "planetarygearbox": ("planetary-motion", "planetary-render"),
          "sevensegment": ("sevensegment-render",),
          "spinningwheel": ("wheel-motion", "wheel-parts", "wheel-block"),
          "swisslever": ("swiss-render",)}
BROAD_WORKERS = {"bevel-pr", "clamparm", "escapement", "onewayclutch", "turbinefan", "wheel", "siren"}
BROAD_PACKAGES = {"package-" + name for name in
                  ("meshgap", "gearroot", "edgefinish", "m2fastener", "cornercheck", "splitcheck")}
IMAGE_SUFFIXES = {".png", ".jpg", ".jpeg", ".gif", ".webp"}
BROAD_RENDER = {"planetary-motion", "wheel-motion", "wheel-parts", "fan-motion", "escapement-exploded"}


def render_pkg(owner):
    """Return the item name when owner is a registered item's render package, else None."""
    match = re.fullmatch(re.escape(ROOT) + r"/(models|features)/([a-z0-9]+)/render", owner)
    if not match:
        return None
    return match.group(2) if match.group(2) in {"models": MODELS, "features": FEATURES}[match.group(1)] else None


def render_dependents(seeds, before, after):
    """Return the render packages that import a seed, directly or through other render packages."""
    graph = {}
    for snapshot in (before, after):
        for name, item in snapshot.items():
            graph.setdefault(name, set()).update(item["imports"])
    found = set(seeds)
    while True:
        more = {name for name, imports in graph.items()
                if render_pkg(name) and name not in found and imports & found}
        if not more:
            return found
        found |= more



def registration_only(old, new, name):
    """Accept only one model import, registration and job-count change, plus at most one gallery tile.

    The gallery tile is optional: a model may have no tile. old and new hold the file texts by base name;
    new["gallery.go"] may equal old["gallery.go"] or add exactly one tile line for the model.
    """
    def lines(text):
        return [line for line in text.splitlines()
                if not line.startswith("// galleryTiles puts the spinning wheel in a tall cell")]

    old_models, new_models = lines(old["models.go"]), lines(new["models.go"])
    imports = [line for line in new_models if re.fullmatch(
        r'\s*([a-zA-Z][a-zA-Z0-9_]*) "' + re.escape(ROOT) + r'/models/' + re.escape(name) + r'/render"', line)]
    if len(imports) != 1:
        return False
    alias = imports[0].split()[0]
    calls = [line for line in new_models if line.strip() == alias + ".Model(),"]
    if len(calls) != 1:
        return False
    new_models.remove(imports[0])
    new_models.remove(calls[0])
    if new_models != old_models:
        return False

    old_gallery, new_gallery = lines(old["gallery.go"]), lines(new["gallery.go"])
    tile = [line for line in new_gallery if re.fullmatch(
        r'\s*\{"' + re.escape(name) + r'", image\.Rect\(\d+, \d+, \d+, \d+\)\},', line)]
    if len(tile) > 1:
        return False
    if tile:
        new_gallery.remove(tile[0])
    if new_gallery != old_gallery:
        return False

    old_test, new_test = lines(old["registry_test.go"]), lines(new["registry_test.go"])
    count = r"\s*require\.Len\(t, paths, \d+\)"
    old_counts = [line for line in old_test if re.fullmatch(count, line)]
    new_counts = [line for line in new_test if re.fullmatch(count, line)]
    if len(old_counts) != 1 or len(new_counts) != 1:
        return False
    old_test.remove(old_counts[0])
    new_test.remove(new_counts[0])
    return old_test == new_test


def only_new_registration(base, tested, code, new_render):
    """Accept a change that only registers one new model renderer, with or without a gallery tile."""
    required = {"tools/render/models.go", "tools/render/registry_test.go"}
    paths = required | {"tools/render/gallery.go"}
    tool_code = {path for path in code if path.startswith("tools/render/")}
    if len(new_render) != 1 or tool_code not in (required, paths):
        return False
    paths = tool_code
    try:
        snapshots = []
        for rev in (base, tested):
            snapshots.append({Path(path).name: subprocess.check_output(
                ["git", "show", f"{rev}:{path}"], stderr=subprocess.PIPE).decode() for path in paths})
        for snapshot in snapshots:
            snapshot.setdefault("gallery.go", "")
        return registration_only(*snapshots, next(iter(new_render)))
    except (OSError, UnicodeError, subprocess.CalledProcessError):
        return False

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
    """Read package and test imports from every module that exists at one revision."""
    archive = subprocess.check_output(["git", "archive", revision], stderr=subprocess.PIPE)
    Path(".tmp").mkdir(exist_ok=True)
    with tempfile.TemporaryDirectory(dir=".tmp") as tmp:
        root = Path(tmp).resolve()
        with tarfile.open(fileobj=io.BytesIO(archive)) as tree:
            tree.extractall(root, filter="data")
        packages = {}
        env = dict(os.environ, GOFLAGS="-mod=readonly", GOWORK="off")
        for module in (".", "tools/render"):
            if not (root / module / "go.mod").is_file():
                continue
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
    if not code:
        return []
    try:
        inputs = []
        for path in code:
            parts = Path(path).parts
            if not path.endswith("_test.go"):
                return []
            if len(parts) == 3 and parts[0] == "models" and parts[1] in MODELS:
                owner = parts[1]
            elif len(parts) == 3 and parts[0] == "internal" and parts[1] in INTERNAL:
                owner = "internal-" + parts[1]
            elif len(parts) == 3 and parts[0] == "features" and parts[1] in FEATURES:
                owner = "feature-" + parts[1]
            elif len(parts) == 3 and parts[:2] == ("tools", "render"):
                owner = "render"
            elif len(parts) == 4 and parts[0] == "models" and parts[1] in MODELS and parts[2] == "render":
                owner = "modelrender-" + parts[1]
            elif len(parts) == 4 and parts[0] == "features" and parts[1] in FEATURES and parts[2] == "render":
                owner = "featurerender-" + parts[1]
            else:
                return []
            old = subprocess.check_output(["git", "show", f"{base}:{path}"], stderr=subprocess.PIPE)
            new = subprocess.check_output(["git", "show", f"{tested}:{path}"], stderr=subprocess.PIPE)
            predicate = all(b"// CI: isolated-predicate" in text for text in (old, new))
            if predicate and len(code) != 1:
                return []
            if not predicate and (b"// CI: isolated-tests" not in old or b"// CI: isolated-tests" not in new):
                return []
            tagged = any(tag in text for text in (old, new) for tag in (b"//go:build", b"// +build"))
            if tagged and path != "models/spinningwheel/head_leg_coplanar_test.go":
                return []
            if tagged:
                owner = "tag-coplanar_test-" + owner
            inputs.append((owner, path, predicate, old, new))
        suites = set()
        Path(".tmp").mkdir(exist_ok=True)
        with tempfile.TemporaryDirectory(dir=".tmp") as directory:
            root = Path(directory)
            binary = root / "testdiff"
            subprocess.run(["go", "build", "-o", str(binary), "scripts/ci/testdiff.go"], check=True,
                           stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            for model, path, predicate, old, new in inputs:
                (root / "old.go").write_bytes(old)
                (root / "new.go").write_bytes(new)
                args = [str(binary)]
                if predicate:
                    # Inspect the exact tested package, including files outside this diff.
                    package = str(Path(path).parent)
                    archive = subprocess.check_output(["git", "archive", tested, package])
                    with tarfile.open(fileobj=io.BytesIO(archive)) as tree:
                        tree.extractall(root, filter="data")
                    args.extend(["--predicate", str(root / package), Path(path).name])
                names = json.loads(subprocess.check_output(args + [str(root / "old.go"), str(root / "new.go")]))
                if names is None:
                    return []
                if not isinstance(names, list) or any(not re.fullmatch(r"Test[A-Za-z0-9_]*", n) for n in names):
                    return []
                suites.update("isolated-" + model + "-" + name for name in names)
        return sorted(suites)
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
        dependency_files = {"go.mod", "go.sum"}
        if dependency_files.intersection(changed):
            reason = "module dependency files changed; full coverage required"
            result.update(suites=["broad", "full"], reasons={"broad": [reason], "full": [reason]}, reason=reason)
            return result
        wheel_parts = any(path.startswith("models/spinningwheel/images/parts/") and
                          Path(path).suffix.lower() == ".png" for path in changed)
        def is_presentation(path):
            # No test reads gallery images; tests write images to temporary directories.
            # Images under testdata stay inputs, because a test could read them as fixtures.
            return (path.lower().endswith(".md") or
                    (Path(path).suffix.lower() in IMAGE_SUFFIXES and "testdata" not in Path(path).parts))
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
                reason = "changed isolated tests or test predicates"
                result.update(suites=isolated, reasons={suite: [reason] for suite in isolated}, reason=reason)
                return result
        code = [path for path in changed if path.endswith(".go") and path not in CI_FILES]
        try:
            before, after = metadata(base), metadata(tested)
        except (subprocess.CalledProcessError, ValueError, OSError, KeyError, tarfile.TarError) as exc:
            if any(str(Path(path).parent) not in TEST_PACKAGE_DIRS for path in code):
                raise RuntimeError("package metadata unavailable for an unregistered Go directory") from exc
            raise
        new_packages = set(after) - set(before)
        new_suites = {}
        new_render = set()
        if new_packages:
            for name in sorted(new_packages):
                directory = after[name]["dir"]
                parts = directory.split("/")
                if render_pkg(name) and name == ROOT + "/" + directory:
                    new_render.add(parts[1])
                    continue
                if (len(parts) != 2 or parts[0] not in {"models", "internal", "features"} or
                        not re.fullmatch(r"[a-z][a-z0-9]*", parts[1]) or name != ROOT + "/" + directory):
                    raise RuntimeError("new package cannot be dispatched: " + directory)
                prefix = {"models": "new-model-", "internal": "new-package-", "features": "new-feature-"}[parts[0]]
                registry = {"models": MODELS, "internal": INTERNAL, "features": FEATURES}[parts[0]]
                if parts[1] not in registry:
                    new_suites[name] = prefix + parts[1]
        if set(before) - set(after):
            reason = "deleted package requires full coverage"
            suites = {"broad": [reason], "full": [reason]}
            result.update(suites=sorted(suites), reasons=suites, reason=reason)
            return result
        unknown_input = any(not is_presentation(path) and not path.endswith(".go") and path not in CI_FILES
                            and not path.startswith("tools/render/fonts/") for path in changed)
        owners, production = affected_packages(code, before, after)
        owners.update(new_packages)
        suites = {}
        if new_suites:
            reason = "new packages require direct tests"
            for suite in new_suites.values():
                suites[suite] = [reason]
        if unknown_input:
            suites.setdefault("broad", []).append("unregistered or shared input changed")
        if wheel_parts:
            suites["wheel-parts"] = ["check changed wheel part documentation"]
        model_owners = set()
        for owner in sorted(owners):
            if owner == R_TOOL or render_pkg(owner):
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
                item = after.get(owner)
                directory = item["dir"] if item else ""
                parts = directory.split("/")
                if (len(parts) != 2 or parts[0] not in {"models", "internal", "features"} or
                        not re.fullmatch(r"[a-z][a-z0-9]*", parts[1]) or owner != ROOT + "/" + directory):
                    reason = "unregistered package requires full coverage: " + owner
                    suites.setdefault("broad", []).append(reason)
                    suites.setdefault("full", []).append(reason)
                    continue
                prefix = {"models": "new-model-", "internal": "new-package-", "features": "new-feature-"}[parts[0]]
                suite = prefix + parts[1]
                if parts[0] == "models" and owner in production:
                    model_owners.add(parts[1])
            suites.setdefault(suite, []).append("affected package " + owner)
        core = [path for path in code if path.startswith("internal/renderkit/") and not path.endswith("_test.go")]
        if core:
            suites["renderer-all"] = ["shared render core changed"]
            if unknown_input:
                suites.setdefault("full", []).append("renderer source and unregistered input changed")
        render_suites = {}
        seeds = set()
        for path in code:
            parent = Path(path).parent
            for owner in (ROOT + "/" + str(parent),):
                name = render_pkg(owner)
                if name:
                    render_suites.setdefault(name, "changed render package " + owner)
                    if not path.endswith("_test.go"):
                        seeds.add(owner)
        for owner in sorted(render_dependents(seeds, before, after) - seeds):
            render_suites.setdefault(render_pkg(owner), "imports a changed render package")
        for name in sorted(new_render):
            render_suites.setdefault(name, "new render package")
        for path in code:
            parts = Path(path).parts
            if (len(parts) == 3 and parts[0] in ("models", "features") and not path.endswith("_test.go")
                    and ROOT + "/" + parts[0] + "/" + parts[1] + "/render" in after):
                render_suites.setdefault(parts[1], "renderer check for " + parts[1])
        for name, reason in sorted(render_suites.items()):
            suites.setdefault("render-" + name, []).append(reason)
        if any(path.startswith("tools/render/") and (path.endswith(".go") or path.startswith("tools/render/fonts/"))
               for path in changed):
            if not any(path.startswith("tools/render/fonts/") for path in changed) and \
                    only_new_registration(base, tested, code, new_render):
                suites["render-registration"] = ["new model registry entry"]
            else:
                suites["render-tool"] = ["render tool source or fonts changed"]
        if model_owners or R_TOOL in production:
            suites["renderer-compile"] = ["compile renderer against changed model APIs"]
            for name in sorted(model_owners & RENDER.keys()):
                for suite in RENDER[name]:
                    suites.setdefault(suite, []).append("renderer check for " + name)
        for name in list(RENDER):
            if "render-" + name in suites:
                for suite in RENDER[name]:
                    suites.pop(suite, None)
        if "full" in suites or "renderer-all" in suites:
            covered = {suite for checks in RENDER.values() for suite in checks} | {"renderer-compile"}
            if "renderer-all" in suites:
                covered |= {"package-renderkit"} | {suite for suite in suites if suite.startswith("render-")}
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
    if event == "push" and os.environ.get("GITHUB_ACTIONS") == "true":
        from reuse_main import find_reusable_run
        proof = find_reusable_run(result, base, result["tested"])
        if proof:
            proof["suites"] = result["suites"]
            result.update(suites=[], reasons={}, reused=proof,
                          reason="matching successful pull request already tested this tree")
    encoded = json.dumps(result, sort_keys=True, separators=(",", ":"))
    print(encoded)
    if os.environ.get("GITHUB_OUTPUT"):
        with open(os.environ["GITHUB_OUTPUT"], "a") as output:
            print("pilot=" + str(result["suites"] == ["cycloidal-pr"]).lower(), file=output)
            print("plan=" + encoded, file=output)


if __name__ == "__main__":
    main()
