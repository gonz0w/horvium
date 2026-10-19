# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
import json
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location('executor_evidence', ROOT / 'tools/check-executor-recovery-evidence.py')


class ExecutorRecoveryEvidenceTests(unittest.TestCase):
    def setUp(self):
        self.checker = importlib.util.module_from_spec(SPEC)
        SPEC.loader.exec_module(self.checker)

    def proof(self):
        parent = 'TestNativeExecutorOwnershipFences'
        text = f'--- PASS: {parent} (1.0s)\n'
        text += ''.join(f'    --- PASS: {parent}/{name} (0.1s)\n' for name in sorted(self.checker.FENCES))
        for phase, cases in [('Before', self.checker.BEFORE), ('After', self.checker.AFTER)]:
            parent = 'TestNativeExecutorRecovery' + phase
            text += f'--- PASS: {parent} (1.0s)\n'
            text += ''.join(f'    --- PASS: {parent}/{name} (0.1s)\n' for name in sorted(cases))
            text += 'EXECUTOR_RECOVERY_RESULT ' + phase.lower() + '=pass\n'
        for mode in ['owner-death', 'stopped-guard', 'guard-first']:
            text += 'EXECUTOR_CONTAINMENT ' + json.dumps({'mode': mode, 'singleton_reacquired': True,
                'original_probe_live': mode == 'guard-first', 'original_child_live': False, 'cleanup_verified': True}) + '\n'
        text += '--- PASS: TestNativeExecutorSupervision (1.0s)\n'
        text += ''.join(f'    --- PASS: TestNativeExecutorSupervision/{name} (0.1s)\n' for name in sorted(self.checker.SUPERVISOR))
        text += '--- PASS: TestNativeSupervisorBootAfter (0.1s)\nSUPERVISOR_RESULT before=pass\nSUPERVISOR_RESULT after=pass\n'
        text += 'SUPERVISOR_NATIVE_ABI status=80 kill=80 pid=0 acquire=2 statuscmd=4 killcmd=6\n'
        parent = 'TestNativeSupervisorPropertyControls'
        text += f'--- PASS: {parent} (1.0s)\n'
        text += ''.join(f'    --- PASS: {parent}/{name} (0.1s)\n' for name in sorted(self.checker.CONTROLS))
        text += ''.join(f'    --- PASS: {parent}/guard-quiescence/{name} (0.1s)\n' for name in ['same-group','escaped'])
        text += 'SUPERVISOR_CONTROLS_RESULT pass\n'
        return text + 'EXECUTOR_REBOOT {"before":[100,1],"after":[200,2]}\n'

    def test_complete_research_requires_actual_reboot_and_all_cases(self):
        self.checker.validate(self.proof())
        for bad in ['', self.proof().replace('effect-cut', 'unknown'), self.proof().replace('clock-independent', 'unknown'),
                    self.proof().replace('SUPERVISOR_NATIVE_ABI', 'MISSING_ABI'), self.proof().replace('SUPERVISOR_RESULT', 'MISSING_SUPERVISOR'),
                    self.proof().replace('--- PASS: TestNativeExecutorRecoveryAfter ', '--- SKIP: TestNativeExecutorRecoveryAfter '),
                    self.proof().replace('"after":[200,2]', '"after":[100,1]'),
                    self.proof().replace('"after":[200,2]', '"after":[true,2]'),
                    self.proof().replace('EXECUTOR_REBOOT ', 'MISSING_REBOOT '),
                    self.proof().replace('EXECUTOR_CONTAINMENT ', 'MISSING_CONTAINMENT '),
                    self.proof().replace('"original_probe_live": true', '"original_probe_live": false'),
                    self.proof() + '--- PASS: TestNativeExecutorRecoveryBefore/effect-cut (0.1s)\n',
                    self.proof().replace('after=pass', 'after=fail')]:
            with self.subTest(bad=bad[-90:]):
                with self.assertRaises(ValueError):
                    self.checker.validate(bad)

    def test_native_ownership_fences_are_required_without_waiving_old_cuts(self):
        proof = self.proof()
        for case in ('guard-status-one', 'pending-proof'):
            line = f'    --- PASS: TestNativeExecutorOwnershipFences/{case} (0.1s)\n'
            for changed in ('', line.replace('PASS','FAIL'), line.replace('PASS','SKIP'),
                            line + line, line.replace(case,'unknown')):
                with self.subTest(case=case,changed=changed), self.assertRaises(ValueError):
                    self.checker.validate(proof.replace(line,changed))
        # A new fence result never substitutes for a blocked historical cut.
        for case in ('stopped-guard','guard-first','supervisor-first'):
            with self.subTest(cut=case), self.assertRaises(ValueError):
                self.checker.validate(proof.replace('/'+case+' ', '/blocked-cut '))
