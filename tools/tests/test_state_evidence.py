# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
from pathlib import Path
import unittest

ROOT=Path(__file__).resolve().parents[2]
SPEC=importlib.util.spec_from_file_location('state_evidence',ROOT/'tools/check-state-evidence.py')
CASES=['root_directory', 'private_lock', 'cloexec', 'second_writer_denial', 'close_release', 'reopen', 'stable_inode', 'first_writer_race', 'directory_sync', 'lease_close', 'path_denial', 'no_directory_creation', 'directory_mode_denial', 'directory_owner_denial', 'ancestor_denial', 'directory_symlink_denial', 'lock_mode_denial', 'lock_owner_denial', 'lock_nonempty_denial', 'lock_hardlink_denial', 'lock_symlink_denial', 'lock_fifo_denial', 'lock_directory_denial', 'unsupported_fs_denial', 'mount_teardown', 'kernel_uid_denial', 'process_lock', 'process_denial', 'process_kill', 'process_reaped', 'death_release', 'death_inode_preservation', 'final_close', 'foreign_preservation', 'fixture_teardown']
RESULT='STATE_DIRECTORY_RESULT '+' '.join(c+'=pass' for c in CASES)+'\n'
PASS='--- PASS: TestNativeStateDirectory (1.0s)\n'

class StateEvidenceTests(unittest.TestCase):
    def module(self):
        m=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(m);return m

    def test_complete_executed_native_suite_required(self):
        m=self.module();m.validate(PASS+RESULT)
        for bad in ['',PASS,RESULT,PASS+PASS+RESULT,PASS+RESULT+RESULT,PASS+RESULT+'FAIL\n',PASS+RESULT+'--- SKIP: TestNativeStateDirectory (0.0s)\n']:
            with self.subTest(bad=bad),self.assertRaises(ValueError):m.validate(bad)

    def test_missing_duplicate_unknown_failed_or_oversized_cases_denied(self):
        m=self.module()
        for case in CASES:
            for bad in [RESULT.replace(case+'=pass',''),RESULT.replace(case+'=pass',case+'=fail'),RESULT.rstrip()+' '+case+'=pass\n']:
                with self.subTest(case=case),self.assertRaises(ValueError):m.validate(PASS+bad)
        for bad in [RESULT.rstrip()+' invented=pass\n',RESULT+'x'*(1<<20)]:
            with self.assertRaises(ValueError):m.validate(PASS+bad)
