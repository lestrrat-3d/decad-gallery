#!/usr/bin/env python3
"""Check that the renderer writes byte-identical images at a base revision and at the working tree."""
import argparse
from dataclasses import dataclass
import filecmp
from pathlib import Path
import shutil
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
WORK = ROOT / ".tmp/render-compare"


@dataclass(frozen=True)
class Job:
    model: str
    shot: str
    kit: str  # empty when the job has no kit
    path: str  # slash-separated output path relative to the repository root


def parse_jobs(text):
    """Parse `-list` output: model, shot, kit-or-"-" and path, separated by tabs."""
    jobs = []
    for line in text.splitlines():
        if not line.strip():
            continue
        fields = line.split("\t")
        if len(fields) != 4 or not all(fields):
            raise ValueError(f"malformed job line: {line!r}")
        model, shot, kit, path = fields
        jobs.append(Job(model, shot, "" if kit == "-" else kit, path))
    return jobs


def files_under(path):
    """Return the files below path (or path itself) keyed by their path relative to path."""
    if path.is_file():
        return {Path(path.name): path}
    if not path.is_dir():
        return {}
    return {item.relative_to(path): item for item in sorted(path.rglob("*")) if item.is_file()}


def compare_trees(left, right):
    """Return (missing, differing): files present on one side only, and files whose bytes differ."""
    a, b = files_under(left), files_under(right)
    missing = sorted(str(name) for name in a.keys() ^ b.keys())
    differing = sorted(str(name) for name in a.keys() & b.keys()
                       if not filecmp.cmp(a[name], b[name], shallow=False))
    return missing, differing


def judge(base_error, head_error, comparison, base_repeat=None):
    """Classify one job. Return (status, detail); mismatch is the only failing status."""
    if base_error and head_error:
        return "both-fail", f"base: {base_error}; head: {head_error}"
    if base_error or head_error:
        return "mismatch", f"base: {base_error or 'ok'}; head: {head_error or 'ok'}"
    missing, differing = comparison
    if missing:
        return "mismatch", "file set differs: " + ", ".join(missing)
    if not differing:
        return "same", ""
    if base_repeat is None:
        raise ValueError("a differing result needs a repeat of the base run")
    base_missing, base_differing = base_repeat
    unstable = set(base_missing) | set(base_differing)
    real = [name for name in differing if name not in unstable]
    if real:
        return "mismatch", "bytes differ: " + ", ".join(real)
    return "nondeterministic", "base differs from itself: " + ", ".join(differing)


def run(command, cwd):
    """Run command and return its error text, or an empty string on success."""
    proc = subprocess.run(command, cwd=cwd, capture_output=True, text=True)
    if proc.returncode == 0:
        return ""
    return (proc.stderr.strip() or proc.stdout.strip() or f"exit {proc.returncode}").splitlines()[-1]


def build(args, cwd):
    subprocess.run(args, cwd=cwd, check=True)


def render_job(binary, cwd, job, out):
    shutil.rmtree(out, ignore_errors=True)
    out.parent.mkdir(parents=True, exist_ok=True)
    command = [str(binary), "-model", job.model, "-shot", job.shot]
    if job.kit:
        command += ["-kit", job.kit]
    return run(command + ["-out", str(out)], cwd)


def build_binaries(base):
    shutil.rmtree(WORK, ignore_errors=True)
    head = WORK / "head/render"
    build(["go", "build", "-o", str(head), "./tools/render"], ROOT)
    source = WORK / "base-src"
    source.mkdir(parents=True)
    archive = subprocess.Popen(["git", "archive", base], cwd=ROOT, stdout=subprocess.PIPE)
    subprocess.run(["tar", "-x", "-C", str(source)], stdin=archive.stdout, check=True)
    if archive.wait() != 0:
        raise SystemExit(f"git archive {base} failed")
    binary = WORK / "base/render"
    if (source / "tools/render/go.mod").exists():
        build(["go", "-C", str(source / "tools/render"), "build", "-o", str(binary), "."], ROOT)
    else:
        build(["go", "-C", str(source), "build", "-o", str(binary), "./tools/render"], ROOT)
    return binary, head, source


def list_jobs(head, models):
    command = [str(head), "-list"]
    for model in models:
        command += ["-model", model]
    proc = subprocess.run(command, cwd=ROOT / "tools/render", capture_output=True, text=True)
    if proc.returncode != 0 or not proc.stdout.strip():
        raise SystemExit("head binary has no -list; run after Task 3\n" + proc.stderr.strip())
    return parse_jobs(proc.stdout)


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base", required=True, help="git revision to compare against")
    parser.add_argument("--model", action="append", default=[], help="limit to a model; repeatable")
    args = parser.parse_args(argv)
    base_binary, head_binary, base_source = build_binaries(args.base)
    jobs = list_jobs(head_binary, args.model)
    results = []
    for job in jobs:
        base_out = WORK / "out-base" / job.path
        head_out = WORK / "out-head" / job.path
        base_error = render_job(base_binary, base_source / "tools/render", job, base_out)
        head_error = render_job(head_binary, ROOT / "tools/render", job, head_out)
        comparison = compare_trees(base_out, head_out) if not (base_error or head_error) else ([], [])
        repeat = None
        if not (base_error or head_error) and comparison[1]:
            again = WORK / "out-base2" / job.path
            render_job(base_binary, base_source / "tools/render", job, again)
            repeat = compare_trees(base_out, again)
        status, detail = judge(base_error, head_error, comparison, repeat)
        results.append((job, status, detail))
        print(f"{status}\t{job.model}\t{job.shot}\t{job.kit or '-'}\t{job.path}", flush=True)
    print()
    counts = {}
    for _, status, _ in results:
        counts[status] = counts.get(status, 0) + 1
    for _, status, detail in results:
        if detail:
            print(f"{status}: {detail}")
    print("summary: " + ", ".join(f"{count} {status}" for status, count in sorted(counts.items())))
    return 1 if counts.get("mismatch") else 0


if __name__ == "__main__":
    sys.exit(main())
