# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location('executor_evidence', ROOT / 'tools/check-executor-recovery-evidence.py')


class ExecutorRecoveryEvidenceTests(unittest.TestCase):
    def setUp(self):
        self.checker = importlib.util.module_from_spec(SPEC)
        SPEC.loader.exec_module(self.checker)

    def proof(self):
        text = ''
        for phase, cases in [('Before', self.checker.BEFORE), ('After', self.checker.AFTER)]:
            parent = 'TestNativeExecutorRecovery' + phase
            text += f'--- PASS: {parent} (1.0s)\n'
            text += ''.join(f'    --- PASS: {parent}/{name} (0.1s)\n' for name in sorted(cases))
            text += 'EXECUTOR_RECOVERY_RESULT ' + phase.lower() + '=pass\n'
        return text + 'EXECUTOR_REBOOT {"before":[100,1],"after":[200,2]}\n'

    def test_complete_research_requires_actual_reboot_and_all_cases(self):
        self.checker.validate(self.proof())
        for bad in ['', self.proof().replace('effect-cut', 'unknown'),
                    self.proof().replace('--- PASS: TestNativeExecutorRecoveryAfter ', '--- SKIP: TestNativeExecutorRecoveryAfter '),
                    self.proof().replace('"after":[200,2]', '"after":[100,1]'),
                    self.proof().replace('"after":[200,2]', '"after":[true,2]'),
                    self.proof().replace('EXECUTOR_REBOOT ', 'MISSING_REBOOT '),
                    self.proof() + '--- PASS: TestNativeExecutorRecoveryBefore/effect-cut (0.1s)\n',
                    self.proof().replace('after=pass', 'after=fail')]:
            with self.subTest(bad=bad[-90:]):
                with self.assertRaises(ValueError):
                    self.checker.validate(bad)
