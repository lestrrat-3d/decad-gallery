import copy
import json
from pathlib import Path
import unittest
from unittest import mock

import reuse_main


class ReuseMainTest(unittest.TestCase):
    def setUp(self):
        Path(".tmp").mkdir(exist_ok=True)
        self.before = "b" * 40
        self.after = "a" * 40
        self.head = "c" * 40
        self.tested = "d" * 40
        self.result = {"suites": ["model-ballbearing"], "changed_paths": ["models/ballbearing/build.go"]}
        self.pr = {"state": "MERGED", "baseRefName": "main", "mergeCommit": {"oid": self.after},
                   "headRefName": "topic", "headRefOid": self.head}
        self.run = {"headSha": self.head, "status": "completed", "conclusion": "success", "attempt": 1,
                    "databaseId": 42}
        self.jobs = [{"name": name, "status": "completed", "conclusion": "success", "databaseId": index}
                     for index, name in enumerate(("plan", "gate", "model-checks"), 1)]
        self.prior = {"version": 1, "base": self.before, "tested": self.tested,
                      "suites": self.result["suites"].copy(), "changed_paths": self.result["changed_paths"].copy()}
        self.tested_parents = [self.tested, self.before, self.head]
        self.checkout = self.tested
        self.trees = {self.tested: "same", self.after: "same"}

    def proof(self):
        def fake_git(*args):
            if args[:3] == ("rev-list", "--parents", "-n"):
                return " ".join([self.after, self.before] if args[-1] == self.after else self.tested_parents)
            if args[0] == "show":
                return "test change (#12)"
            if args[0] == "rev-parse":
                return self.trees[args[1].removesuffix("^{tree}")]
            raise AssertionError(args)

        def fake_gh(directory, name, *args):
            path = directory / name
            if name == "plan.log":
                path.write_text("plan\tRun actions/checkout@v4\t2026-01-01Z [command]git fetch origin "
                                f"+{self.tested}:refs/remotes/pull/12/merge\n"
                                f"plan\tRun actions/checkout@v4\t2026-01-01Z {self.checkout}\n"
                                "plan\tRun python3 scripts/ci/plan.py\t2026-01-01Z " + json.dumps(self.prior) + "\n")
            else:
                payload = {"pr.json": self.pr, "runs.json": [self.run], "jobs.json": {"jobs": self.jobs}}[name]
                path.write_text(json.dumps(payload))
            return path

        with mock.patch.object(reuse_main, "git", side_effect=fake_git), \
                mock.patch.object(reuse_main, "gh_file", side_effect=fake_gh), \
                mock.patch.object(reuse_main.subprocess, "run"):
            return reuse_main.find_reusable_run(self.result, self.before, self.after)

    def test_reuses_only_the_exact_successful_run(self):
        self.assertEqual(self.proof(), {"pull_request": 12, "run": 42, "tested": self.tested})

    def test_rejects_missing_or_different_evidence(self):
        cases = (
            lambda: self.pr["mergeCommit"].update(oid="f" * 40),
            lambda: self.run.update(conclusion="failure"),
            lambda: setattr(self, "checkout", "f" * 40),
            lambda: self.run.update(attempt=2),
            lambda: self.jobs[1].update(conclusion="failure"),
            lambda: self.jobs[2].update(conclusion="skipped"),
            lambda: self.prior.update(base="f" * 40),
            lambda: self.prior.update(suites=["model-m2bolt"]),
            lambda: self.prior.update(changed_paths=["models/ballbearing/other.go"]),
            lambda: self.tested_parents.__setitem__(1, "f" * 40),
            lambda: self.trees.update({self.tested: "different"}),
        )
        for case in cases:
            with self.subTest(case=case):
                self.setUp()
                case()
                self.assertIsNone(self.proof())

    def test_never_reuses_full_or_ci_changes(self):
        for result in ({"suites": ["broad", "full"], "changed_paths": ["go.mod"]},
                       {"suites": ["model-ballbearing"], "changed_paths": ["scripts/ci/gate.py"]}):
            with self.subTest(result=result):
                self.result = copy.deepcopy(result)
                self.assertIsNone(self.proof())
