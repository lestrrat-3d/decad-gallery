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
    lines = path.read_text(encoding="utf-8").splitlines()
    unknown = [line.split("\t", 2)[2].partition(" ")[2] for line in lines
               if line.startswith("plan\tUNKNOWN STEP\t") and len(line.split("\t", 2)) == 3]
    if unknown:
        # Some GitHub job logs omit step names. Bound evidence by the runner's step headers.
        headers = [(index, message) for index, message in enumerate(unknown)
                   if message.startswith("##[group]Run ")]
        checkout = [i for i, (_, message) in enumerate(headers) if message == "##[group]Run actions/checkout@v4"]
        planner = [i for i, (_, message) in enumerate(headers)
                   if message == "##[group]Run python3 scripts/ci/plan.py"]
        if (len(checkout) != 1 or len(planner) != 1 or checkout[0] >= planner[0] or
                checkout[0] + 1 >= len(headers) or planner[0] + 1 >= len(headers)):
            return None
        checkout_lines = unknown[headers[checkout[0]][0]:headers[checkout[0] + 1][0]]
        planner_lines = unknown[headers[planner[0]][0]:headers[planner[0] + 1][0]]
    else:
        checkout_lines = [line.split("\t", 2)[2].partition(" ")[2] for line in lines
                          if line.startswith("plan\tRun actions/checkout@v4\t")]
        planner_lines = [line.split("\t", 2)[2].partition(" ")[2] for line in lines
                         if line.startswith("plan\tRun python3 scripts/ci/plan.py\t")]
    fetches = [match.group(1) for message in checkout_lines if "[command]" in message and " fetch " in message
               if (match := re.search(r" \+([0-9a-f]{40}):refs/remotes/pull/" + pr_number + r"/merge(?:\s|$)", message))]
    checks = [i for i, message in enumerate(checkout_lines)
              if re.search(r"\[command\].*/git log -1 --format=%H$", message)]
    checkouts = ([checkout_lines[checks[0] + 1]] if len(checks) == 1 and checks[0] + 1 < len(checkout_lines)
                 and SHA.fullmatch(checkout_lines[checks[0] + 1]) else [])
    for message in planner_lines:
        if message.startswith("{"):
            try:
                item = json.loads(message)
            except json.JSONDecodeError:
                return None
            if isinstance(item, dict) and item.get("version") == 1 and "tested" in item:
                plans.append(item)
    if len(plans) == len(fetches) == len(checkouts) == 1 and fetches == checkouts == [plans[0]["tested"]]:
        return plans[0]
    return None


def find_reusable_run(result, before, after):
    """Return proof for a matching successful PR, or None to run the push checks normally."""
    suites = result.get("suites", [])
    changed = result.get("changed_paths", [])
    full = suites == ["broad", "full"]
    if (not suites or ({"broad", "full"}.intersection(suites) and not full) or
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
                if full:
                    validation = completed_job(jobs, "records-validation")
                    if (not validation or any(not completed_job(validation.get("steps", []), step) for step in
                                              ("Validate full execution records", "Check producer results")) or
                            not completed_job(required.get("steps", []), "Check planned results") or
                            any(not completed_job(plan_job.get("steps", []), step) for step in
                                ("Run actions/checkout@v4", "Run python3 scripts/ci/plan.py"))):
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
