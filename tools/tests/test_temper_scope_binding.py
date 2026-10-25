# SPDX-License-Identifier: BSD-3-Clause
"""Externally reviewed exact-source admission; data cannot choose commands.

LAB execution requires an authorized BSD. These are synthetic data fixtures,
not independent review attestations or actual kernel custody evidence.
"""
import copy
import importlib.util
import json
import platform
import tempfile
import unittest
from types import SimpleNamespace
from pathlib import Path
from unittest.mock import patch

if platform.system() not in {'FreeBSD', 'OpenBSD'}:
    raise RuntimeError('This LAB contract requires an authorized BSD')

ROOT = Path(__file__).resolve().parents[2]


class TemperScopeBindingTests(unittest.TestCase):
    def setUp(self):
        spec = importlib.util.spec_from_file_location('scope_gate', ROOT / 'tools/review-gate.py')
        self.gate = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.gate)
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.directory = Path(self.temp.name)
        self.root = self.directory / 'candidate'
        self.root.mkdir()
        self.path = self.directory / 'reviewed-scope.json'
        self.record = {'base_commit': 'a' * 40, 'source_commit': 'b' * 40, 'source_tree': 'c' * 40}
        self.entries = [{'path': 'PROJECT.md', 'old_mode': '100644', 'new_mode': '100644',
                         'old_blob': 'd' * 40, 'new_blob': 'e' * 40, 'status': 'M'},
                        {'path': 'tools/review-gate.py', 'old_mode': '100644', 'new_mode': '100644',
                         'old_blob': 'f' * 40, 'new_blob': '1' * 40, 'status': 'M'}]
        self.binding = dict(schema=1, route='temper-validation-tooling-v1', **self.record,
                            changed_entries=self.entries,
                            commands=['metadata', 'temper-source', 'review-regressions'],
                            review_reference='https://github.com/gonz0w/horvium/issues/424#issuecomment-123')

    def admission(self, binding=None, entries=None):
        self.path.write_text(json.dumps(self.binding if binding is None else binding))
        with patch.object(self.gate, 'changed_entries', return_value=self.entries if entries is None else entries):
            return self.gate.scope_binding(self.root, self.path, self.record)

    def test_valid_external_complete_binding_admits_fixed_route(self):
        result = self.admission()
        self.assertEqual(result['route'], 'temper-validation-tooling-v1')
        self.assertEqual(result['sha256'], self.gate.PLAN.digest(self.path))
        self.assertEqual(result['review_reference'], self.binding['review_reference'])

    def test_absent_stale_wrong_identity_or_incomplete_entries_fall_back(self):
        self.assertIsNone(self.gate.scope_binding(self.root, None, self.record))
        self.assertIsNone(self.gate.scope_binding(self.root, self.directory / 'missing', self.record))
        for key in self.record:
            bad = copy.deepcopy(self.binding)
            bad[key] = '2' * 40
            self.assertIsNone(self.admission(bad), key)
        for key in ('old_mode', 'new_mode', 'old_blob', 'new_blob', 'path', 'status'):
            bad = copy.deepcopy(self.binding)
            bad['changed_entries'][0][key] = 'wrong'
            self.assertIsNone(self.admission(bad), key)
        bad = copy.deepcopy(self.binding)
        bad['changed_entries'].pop()
        self.assertIsNone(self.admission(bad))

    def test_exact_binding_cannot_admit_product_shared_hook_or_unknown_inputs(self):
        for path in ('internal/console/main.go', 'tools/owned-process.py', '.githooks/pre-commit',
                     'Makefile', 'tools/check-repo.py', 'tools/unreviewed.py', '.codex/config.toml'):
            bad = copy.deepcopy(self.binding)
            bad['changed_entries'][0]['path'] = path
            self.assertIsNone(self.admission(bad, bad['changed_entries']), path)

    def test_duplicate_paths_nonregular_modes_and_arbitrary_commands_refused(self):
        bad = copy.deepcopy(self.binding)
        bad['changed_entries'].append(copy.deepcopy(bad['changed_entries'][0]))
        self.assertIsNone(self.admission(bad, bad['changed_entries']))
        for mode in ('120000', '160000', '100755'):
            bad = copy.deepcopy(self.binding)
            bad['changed_entries'][0]['new_mode'] = mode
            self.assertIsNone(self.admission(bad, bad['changed_entries']))
        for commands in ([], ['metadata'], ['metadata', 'temper-source', 'review-regressions', 'sh'],
                         ['sh', '-c', 'fictional command']):
            bad = dict(self.binding, commands=commands)
            self.assertIsNone(self.admission(bad))
        self.assertIsNone(self.admission(dict(self.binding, arbitrary_argv=['fictional'])))

    def test_nonstandard_duplicate_and_oversized_json_refused(self):
        for text in ('{"schema":1,"schema":1}', '{"schema":NaN}', '{"schema":Infinity}',
                     '[]', 'null', '{', ' ' * ((1 << 20) + 1)):
            self.path.write_text(text)
            with patch.object(self.gate, 'changed_entries', return_value=self.entries):
                self.assertIsNone(self.gate.scope_binding(self.root, self.path, self.record))

    def test_binding_inside_candidate_and_symlink_refused(self):
        self.path.write_text(json.dumps(self.binding))
        inside = self.root / 'scope.json'
        inside.write_bytes(self.path.read_bytes())
        with patch.object(self.gate, 'changed_entries', return_value=self.entries):
            self.assertIsNone(self.gate.scope_binding(self.root, inside, self.record))
            link = self.directory / 'link.json'
            link.symlink_to(self.path)
            self.assertIsNone(self.gate.scope_binding(self.root, link, self.record))

    def test_changed_entries_reads_raw_complete_no_rename_diff(self):
        raw = b':100644 100644 ' + b'd' * 40 + b' ' + b'e' * 40 + b' M\x00PROJECT.md\x00'
        with patch.object(self.gate.subprocess, 'check_output', return_value=raw) as command:
            self.assertEqual(self.gate.changed_entries(self.root, 'base', 'head'), self.entries[:1])
            command.assert_called_once_with(['git', 'diff', '--raw', '--no-abbrev', '--no-renames',
                                             '-z', 'base', 'head'], cwd=self.root)


class MetadataFingerprintTests(unittest.TestCase):
    def setUp(self):
        spec = importlib.util.spec_from_file_location('metadata_gate', ROOT / 'tools/review-gate.py')
        self.gate = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.gate)
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        files = self.gate.TOOLING_PATHS | self.gate.PLAN.TEMPER_PATHS | {
            'Makefile', '.gitleaks.toml', '.python-version', 'tools/check-repo.py',
            'tools/check-repo.sh', 'tools/check-dco.py', 'tools/owned-process.py',
            'tools/pins.json', 'tools/requirements-ci.txt', '.tools/bin/gitleaks',
            '.tools/venv/bin/python3', '.tools/venv/pyvenv.cfg', 'fixture/git', 'fixture/make', 'fixture/sh', 'fixture/yaml/__init__.py'}
        for name in files:
            path = self.root / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text('fictional fixture bytes\n')
        (self.root / '.tools/venv/bin/python3').chmod(0o700)

    def fingerprint(self, scope=None, executable=None, prefix=None):
        gate = self.gate
        with patch.object(gate.sys, 'executable', str(executable or self.root / '.tools/venv/bin/python3')), \
                patch.object(gate.sys, 'prefix', str(prefix or self.root / '.tools/venv')), \
                patch.object(gate.shutil, 'which', side_effect=lambda name: str(self.root / 'fixture' / name)), \
                patch.object(gate.importlib.util, 'find_spec', return_value=SimpleNamespace(origin=str(self.root / 'fixture/yaml/__init__.py'))), \
                patch.object(gate.PLAN, 'go_environment', side_effect=AssertionError('eager Go')), \
                patch.object(gate.PLAN, 'environment_fingerprint', side_effect=AssertionError('Go compiler')):
            return gate.metadata_fingerprint(self.root, scope)

    def test_metadata_identity_binds_actual_tools_tests_checker_yaml_and_scope(self):
        before, inputs = self.fingerprint({'sha256': 'before'})
        self.assertIn('python', inputs)
        self.assertIn('git', inputs)
        self.assertIn('gitleaks', inputs)
        self.assertNotEqual(before, self.fingerprint({'sha256': 'after'})[0])
        for name in ('.tools/venv/bin/python3', '.tools/venv/pyvenv.cfg', 'fixture/git', '.tools/bin/gitleaks',
                     'fixture/yaml/__init__.py', 'tools/check-temper-source.py',
                     'tools/tests/test_temper_gate_dispatch.py', 'tools/owned-process.py'):
            path = self.root / name
            original = path.read_text()
            path.write_text(original + 'changed\n')
            self.assertNotEqual(before, self.fingerprint({'sha256': 'before'})[0], name)
            path.write_text(original)
        self.assertEqual(before, self.fingerprint({'sha256': 'before'})[0])

    def test_candidate_venv_parent_alias_preserves_metadata_identity(self):
        alias = self.root / 'fixture/venv-alias'
        alias.symlink_to(self.root / '.tools/venv', target_is_directory=True)
        self.assertEqual(self.fingerprint(), self.fingerprint(executable=alias / 'bin/python3', prefix=alias))

    def test_different_baseline_interpreter_cannot_supply_metadata_identity(self):
        with self.assertRaisesRegex(ValueError, 'pinned candidate venv interpreter'):
            self.fingerprint(executable=self.root / 'fixture/other-python')
        with self.assertRaisesRegex(ValueError, 'pinned candidate venv interpreter'):
            self.fingerprint(prefix=self.root / 'fixture/other-venv')
        baseline = self.root / '.tools/venv/bin/python3'
        outside = self.root / 'fixture/outside/bin/python3'
        outside.parent.mkdir(parents=True)
        outside.symlink_to(baseline)
        wrong_name = baseline.with_name('python')
        wrong_name.symlink_to(baseline)
        for executable in (outside, wrong_name):
            with self.assertRaisesRegex(ValueError, 'pinned candidate venv interpreter'):
                self.fingerprint(executable=executable)
        (self.root / '.tools/venv/bin/python3').chmod(0o600)
        with self.assertRaisesRegex(ValueError, 'pinned candidate venv interpreter'):
            self.fingerprint()

    def test_missing_metadata_tool_or_checker_refuses_instead_of_qualifying(self):
        for name in ('fixture/git', 'tools/check-temper-source.py'):
            path = self.root / name
            path.unlink()
            with self.assertRaises(OSError):
                self.fingerprint()
            path.write_text('fictional fixture bytes\n')


if __name__ == '__main__':
    unittest.main()
