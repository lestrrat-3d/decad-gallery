#!/usr/bin/env python3
"""Find a completed PR run that tested the exact tree and suites of a main push."""

import json
from pathlib import Path
import re
import subprocess
import tempfile


SHA = re.compile(r"[0-9a-f]{40}")


def git(*args):
    return subprocess.check_output(["git", *args], stderr=subprocess.DEVNULL, text=True, timeout=20).strip()


def gh_file(directory, name, *args):
    path = directory / name
    with path.open("w", encoding="utf-8") as output:
        subprocess.run(["gh", *args], stdout=output, stderr=subprocess.DEVNULL, check=True, text=True, timeout=20)
    return path


def completed_job(jobs, name):
    matches = [job for job in jobs if job.get("name") == name]
    if len(matches) != 1 or matches[0].get("status") != "completed" or matches[0].get("conclusion") != "success":
        return None
    return matches[0]


def plan_from_log(path, pr_number):
    plans = []
    fetched = set()
    checked_out = set()
    for line in path.read_text(encoding="utf-8").splitlines():
        if line.startswith("plan\tRun actions/checkout@v4\t"):
            message = line.split("\t", 2)[2].partition(" ")[2]
            if SHA.fullmatch(message):
                checked_out.add(message)
            if message.startswith("[command]") and " fetch " in message:
                match = re.search(r" \+([0-9a-f]{40}):refs/remotes/pull/" + pr_number + r"/merge$", message)
                if match:
                    fetched.add(match.group(1))
        if "\tRun python3 scripts/ci/plan.py\t" not in line:
            continue
        start = line.find("{")
        if start >= 0:
            try:
                item = json.loads(line[start:])
            except json.JSONDecodeError:
                continue
            if isinstance(item, dict) and item.get("version") == 1 and "tested" in item:
                plans.append(item)
    if len(plans) == 1 and fetched == checked_out == {plans[0]["tested"]}:
        return plans[0]
    return None


def find_reusable_run(result, before, after):
    """Return proof for a matching successful PR, or None to run the push checks normally."""
    suites = result.get("suites", [])
    changed = result.get("changed_paths", [])
    if (not suites or {"broad", "full"}.intersection(suites) or
            any(path.startswith((".github/", "scripts/ci/")) for path in changed) or
            not SHA.fullmatch(before) or not SHA.fullmatch(after)):
        return None
    try:
        parents = git("rev-list", "--parents", "-n", "1", after).split()
        if parents != [after, before]:
            return None
        subject = git("show", "-s", "--format=%s", after)
        match = re.search(r"\(#([0-9]+)\)$", subject)
        if not match:
            return None
        Path(".tmp").mkdir(exist_ok=True)
        with tempfile.TemporaryDirectory(dir=".tmp") as tmp:
            directory = Path(tmp)
            pr = json.loads(gh_file(directory, "pr.json", "pr", "view", match.group(1), "--json",
                                    "state,mergeCommit,baseRefName,headRefName,headRefOid").read_text())
            if (pr.get("state") != "MERGED" or pr.get("baseRefName") != "main" or
                    (pr.get("mergeCommit") or {}).get("oid") != after or
                    not SHA.fullmatch(pr.get("headRefOid", "")) or not pr.get("headRefName")):
                return None
            runs = json.loads(gh_file(directory, "runs.json", "run", "list", "--workflow", "repository.yml",
                                      "--branch", pr["headRefName"], "--event", "pull_request", "--limit", "30",
                                      "--json", "databaseId,headSha,status,conclusion,attempt").read_text())
            for run in runs:
                if (run.get("headSha") != pr["headRefOid"] or run.get("status") != "completed" or
                        run.get("conclusion") != "success" or run.get("attempt") != 1):
                    continue
                run_id = str(run["databaseId"])
                details = json.loads(gh_file(directory, "jobs.json", "run", "view", run_id,
                                             "--json", "jobs").read_text())
                jobs = details["jobs"]
                plan_job = completed_job(jobs, "plan")
                required = completed_job(jobs, "model-checks")
                if not plan_job or not required:
                    continue
                if any(job.get("name") == "gate" for job in jobs):
                    if not completed_job(jobs, "gate"):
                        continue
                elif not completed_job(required.get("steps", []), "Check planned results"):
                    # An older forwarding job alone cannot prove the planned checks passed.
                    continue
                log = gh_file(directory, "plan.log", "run", "view", "--job", str(plan_job["databaseId"]), "--log")
                prior = plan_from_log(log, match.group(1))
                if (not prior or prior.get("base") != before or prior.get("suites") != suites or
                        prior.get("changed_paths") != changed or not SHA.fullmatch(prior.get("tested", ""))):
                    continue
                tested = prior["tested"]
                subprocess.run(["git", "fetch", "--no-tags", "origin", tested], check=True,
                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=20)
                if (git("rev-list", "--parents", "-n", "1", tested).split() !=
                        [tested, before, pr["headRefOid"]]):
                    continue
                if git("rev-parse", tested + "^{tree}") != git("rev-parse", after + "^{tree}"):
                    continue
                return {"pull_request": int(match.group(1)), "run": int(run_id), "tested": tested}
    except (OSError, ValueError, KeyError, TypeError, subprocess.SubprocessError):
        return None
    return None
