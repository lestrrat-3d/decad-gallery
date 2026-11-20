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

    def test_predicate_callers_and_fallbacks(self):
        with tempfile.TemporaryDirectory(dir=self.root) as tmp:
            package = Path(tmp)
            helper = package / 'pred_test.go'
            source = ('package example\nimport "testing"\n// CI: isolated-predicate\n'
                      'func outside(v float64) bool { return v > 1.0 }\n'
                      'func check(t *testing.T) { if outside(2.0) { t.Errorf("outside") } }\n')
            tests = ('package example\n// CI: isolated-tests\nimport "testing"\n'
                     'func TestOne(t *testing.T) { check(t) }\n'
                     'func TestTwo(t *testing.T) { t.Run("case", func(t *testing.T){ check(t) }) }\n'
                     'func TestOther(t *testing.T) {}\n')
            old = self.root / 'predicate-before.go'
            (package / 'callers_test.go').write_text(tests)

            def select(base, head):
                old.write_text(base)
                helper.write_text(head)
                args = [str(self.binary), '--predicate', str(package), helper.name, str(old), str(helper)]
                return json.loads(subprocess.check_output(args))

            def changed(base):
                return select(base, base.replace('1.0', '1.5', 1))

            self.assertEqual(changed(source), ['TestOne', 'TestTwo'])
            for change in ('return v > shared', 'return v > other()', 'return v > v/0',
                           'return func() bool { return true }()'):
                with self.subTest(change=change):
                    self.assertIsNone(select(source, source.replace('return v > 1.0', change)))
            for change in ('func init(){ _ = outside(1) }', 'var selected = outside',
                           'func TestMain(m *testing.M) {}', 'func uncaught(){ check(nil) }',
                           'func missing(){ absent() }', 'var queued = func(){ check(nil) }',
                           'type node struct{}; func (node) call(){ check(nil) }',
                           'type float64 = interface{}'):
                with self.subTest(change=change):
                    self.assertIsNone(changed(source + change + '\n'))
            for ordinary, replacement in [('if outside(2.0)', 'if !outside(2.0)'),
                                           ('t.Errorf("outside")', 't.FailNow()'),
                                           ('t.Errorf("outside")', 't.Errorf("%v", outside(3))')]:
                self.assertIsNone(changed(source.replace(ordinary, replacement)))
            for change in [('float64)', 'float32)'), ('import "testing"', 'import ( "testing"; "fmt")'),
                           ('package example', '//go:build extra\npackage example')]:
                self.assertIsNone(select(source, source.replace(*change)))
            tagged = package / 'extra_linux_test.go'
            tagged.write_text('package example\n')
            self.assertIsNone(changed(source))

    def test_layout_requires_isolation_for_new_test_files(self):
        path = self.root / 'layout.go'
        path.write_text('package example\nimport "testing"\nfunc TestOne(t *testing.T) {}\n')
        self.assertNotEqual(subprocess.run([str(self.binary), '--layout', str(path)]).returncode, 0)
        path.write_text('package example\n// CI: isolated-tests\nimport "testing"\n'
                        'func TestOne(t *testing.T) {}\n')
        self.assertEqual(subprocess.run([str(self.binary), '--layout', str(path)]).returncode, 0)
        path.write_text('package example\nfunc helper() {}\n')
        self.assertEqual(subprocess.run([str(self.binary), '--layout', str(path)]).returncode, 0)
        path.write_text('package example\nimport "testing"\nfunc Test(t *testing.T) {}\n')
        self.assertNotEqual(subprocess.run([str(self.binary), '--layout', str(path)]).returncode, 0)
        path.write_text('package example\n// CI: isolated-tests\nimport "testing"\n'
                        'func TestOne(t *testing.T) { TestTwo(t) }\n')
        self.assertNotEqual(subprocess.run([str(self.binary), '--layout', str(path)]).returncode, 0)


if __name__ == '__main__':
    unittest.main()
