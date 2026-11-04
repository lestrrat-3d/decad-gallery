"""Failure injection for the updater's four-file rollback boundary."""

from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import update_decad


class UpdateDecadTests(unittest.TestCase):
    def setUp(self):
        source = Path(update_decad.__file__).resolve().parent.parent
        temporary = source / ".tmp"
        temporary.mkdir(exist_ok=True)
        scratch = tempfile.TemporaryDirectory(prefix="module-update-test-", dir=temporary)
        self.addCleanup(scratch.cleanup)
        self.project = Path(scratch.name)
        self.original = {}
        for relative in update_decad.MODULE_ROOTS:
            for filename in ("go.mod", "go.sum"):
                path = self.project / relative / filename
                path.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(source / relative / filename, path)
                self.original[path] = path.read_bytes()
        self.failure = ""
        self.get_calls = 0

    def assert_restored(self):
        for path, expected in self.original.items():
            self.assertTrue(path.read_bytes() == expected, f"{path.relative_to(self.project)} was not restored")

    def go_with_failure(self, directory, *arguments, capture=False, direct=False):
        if arguments[:3] == ("mod", "edit", "-json"):
            replacements = []
            if self.failure == "local-replace":
                replacements = [{"Old": {"Path": update_decad.MODULE}, "New": {"Path": "../decad"}}]
            return {"Require": [{"Path": update_decad.MODULE, "Version": "v1.2.3"}], "Replace": replacements}
        if arguments[0] == "list":
            if "@main" in arguments[-1] and self.failure == "resolver":
                (directory / "go.sum").write_bytes(b"resolver touched the checksum file\n")
                raise subprocess.CalledProcessError(1, ["go", *arguments])
            version = "v1.2.3"
            if self.failure == "stale-proxy" and "@main" in arguments[-1] and not direct:
                version = "v1.2.2"
            if self.failure == "selection" and directory == self.project / "tools/render" and "@" not in arguments[-1]:
                version = "v1.2.4"
            return {"Path": update_decad.MODULE, "Version": version}
        if arguments[0] == "get":
            self.get_calls += 1
            (directory / "go.mod").write_bytes(b"changed by the module updater\n")
            (directory / "go.sum").write_text(
                f"{update_decad.MODULE} v1.2.3 h1:test\n{update_decad.MODULE} v1.2.3/go.mod h1:test\n"
            )
            if directory == self.project / "tools/render" and self.failure == "renderer":
                raise subprocess.CalledProcessError(1, ["go", *arguments])
        return None

    def test_second_module_failure_restores_both_modules(self):
        self.failure = "renderer"
        with patch.object(update_decad, "run_go", self.go_with_failure):
            with self.assertRaises(subprocess.CalledProcessError):
                update_decad.update(self.project, "main")
        self.assertEqual(self.get_calls, 2)
        self.assert_restored()

    def test_resolution_failure_restores_its_checksum_write(self):
        self.failure = "resolver"
        with patch.object(update_decad, "run_go", self.go_with_failure):
            with self.assertRaises(subprocess.CalledProcessError):
                update_decad.update(self.project, "main")
        self.assertEqual(self.get_calls, 0)
        self.assert_restored()

    def test_selection_mismatch_restores_both_modules(self):
        self.failure = "selection"
        with patch.object(update_decad, "run_go", self.go_with_failure):
            with self.assertRaisesRegex(ValueError, "does not select published Decad"):
                update_decad.update(self.project, "main")
        self.assertEqual(self.get_calls, 2)
        self.assert_restored()

    def test_local_override_refuses_before_updating(self):
        self.failure = "local-replace"
        with patch.object(update_decad, "run_go", self.go_with_failure):
            with self.assertRaisesRegex(ValueError, "replaces Decad"):
                update_decad.update(self.project, "main")
        self.assertEqual(self.get_calls, 0)
        self.assert_restored()

    def test_branch_resolution_uses_fresh_source_metadata(self):
        self.failure = "stale-proxy"
        with patch.object(update_decad, "run_go", self.go_with_failure):
            result = update_decad.update(self.project, "main")
        self.assertEqual(result["version"], "v1.2.3")
        self.assertEqual(set(result["selected"].values()), {"v1.2.3"})


if __name__ == "__main__":
    unittest.main()
