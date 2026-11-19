import json
from pathlib import Path
import subprocess
import tempfile
import unittest


class TestDiff(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        Path('.tmp').mkdir(exist_ok=True)
        cls.temp = tempfile.TemporaryDirectory(dir='.tmp')
        cls.root = Path(cls.temp.name)
        cls.binary = cls.root / 'testdiff'
        subprocess.run(['go', 'build', '-o', str(cls.binary), 'scripts/ci/testdiff.go'], check=True)

    @classmethod
    def tearDownClass(cls):
        cls.temp.cleanup()

    def compare(self, before, after):
        old = self.root / 'before.go'
        new = self.root / 'after.go'
        old.write_text(before)
        new.write_text(after)
        return subprocess.run([str(self.binary), str(old), str(new)], capture_output=True, text=True)

    def test_changed_test_and_shared_declarations(self):
        source = ('package example\n// CI: isolated-tests\nimport "testing"\n'
                  'func TestOne(t *testing.T) { t.Log("old") }\n'
                  'func TestTwo(t *testing.T) {}\n')
        result = self.compare(source, source.replace('"old"', '"new"'))
        self.assertEqual(json.loads(result.stdout), ['TestOne'])
        result = self.compare(source, source + '\nfunc helper() {}\n')
        self.assertNotEqual(result.returncode, 0)
        result = self.compare(source, source.replace('import "testing"', 'import ("testing"; "fmt")'))
        self.assertIsNone(json.loads(result.stdout))
        result = self.compare(source, source.replace('// CI: isolated-tests', ''))
        self.assertNotEqual(result.returncode, 0)


if __name__ == '__main__':
    unittest.main()
