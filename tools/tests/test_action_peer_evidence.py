# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location('peer_evidence', ROOT / 'tools/check-action-peer-evidence.py')


class ActionPeerEvidenceTests(unittest.TestCase):
    def setUp(self):
        self.module = importlib.util.module_from_spec(SPEC)
        SPEC.loader.exec_module(self.module)
        self.log = ('--- PASS: TestNativeActionPeer (1.0s)\n'
                    + ''.join('    --- PASS: TestNativeActionPeer/' + n + ' (0.1s)\n' for n in sorted(self.module.CASES))
                    + 'ACTION_PEER_RESULT ' + ' '.join(n + '=pass' for n in sorted(self.module.CASES)) + '\n')

        self.peer_only = self.log
        self.log += ('--- PASS: TestNativePropertyExchange (1.0s)\n'
                     + ''.join('    --- PASS: TestNativePropertyExchange/' + n + ' (0.1s)\n' for n in sorted(self.module.EXCHANGE_CASES))
                     + 'PROPERTY_EXCHANGE_RESULT ' + ' '.join(n + '=pass' for n in sorted(self.module.EXCHANGE_CASES)) + '\n')
        self.exchange_only = self.log
        self.log += ('--- PASS: TestNativeAdmittedPropertyExchange (1.0s)\n'
                     + ''.join('    --- PASS: TestNativeAdmittedPropertyExchange/' + n + ' (0.1s)\n' for n in sorted(self.module.ADMITTED_CASES))
                     + 'ADMITTED_PROPERTY_RESULT ' + ' '.join(n + '=pass' for n in sorted(self.module.ADMITTED_CASES)) + '\n')

        self.admitted_only = self.log
        self.log += ('--- PASS: TestNativeSessionPropertyClient (1.0s)\n'
                     + ''.join('    --- PASS: TestNativeSessionPropertyClient/' + n + ' (0.1s)\n' for n in sorted(self.module.SESSION_CASES))
                     + 'SESSION_PROPERTY_RESULT ' + ' '.join(n + '=pass' for n in sorted(self.module.SESSION_CASES)) + '\n')

    def test_complete_and_matching_native_case_inventory(self):
        self.module.validate(self.log)
        source = (ROOT / 'internal/helper/action_peer_freebsd_test.go').read_text()
        cases = re.findall(r'"([a-z-]+)"', re.search(r'cases := \[\]string\{([^}]+)\}', source).group(1))
        self.assertEqual(len(cases), len(self.module.CASES))
        self.assertEqual(set(cases), self.module.CASES)

    def test_missing_duplicate_failed_or_extra_assertions(self):
        for case in self.module.CASES:
            for bad in (self.log.replace(case + '=pass', ''),
                        self.log.replace(case + '=pass', case + '=fail'),
                        self.log.replace(case + '=pass', case + '=pass ' + case + '=pass')):
                with self.assertRaises(ValueError):
                    self.module.validate(bad)
        with self.assertRaises(ValueError):
            self.module.validate(self.log + 'ACTION_PEER_RESULT extra=pass\n')

    def test_actual_parent_and_every_child_required(self):
        for bad in ('', self.log + self.log, self.log.replace('--- PASS:', '--- SKIP:'),
                    self.log.replace('--- PASS:', '--- FAIL:'), 'x' * (1 << 20) + self.log,
                    self.log.replace('TestNativeActionPeer', 'TestActionPeerPortable')):
            with self.assertRaises(ValueError):
                self.module.validate(bad)
        for name in self.module.CASES:
            line = '    --- PASS: TestNativeActionPeer/' + name + ' (0.1s)\n'
            for bad in (self.log.replace(line, ''), self.log + line):
                with self.assertRaises(ValueError):
                    self.module.validate(bad)

    def test_script_selects_only_bounded_native_parent(self):
        script = (ROOT / 'tools/bsd-action-peer-prebuilt.sh').read_text()
        self.assertEqual(re.findall(r"-test.run='([^']+)'", script), ['^TestNative(ActionPeer|PropertyExchange|AdmittedPropertyExchange|SessionPropertyClient)$'])
        self.assertIn('/usr/bin/timeout 90', script)
        self.assertIn('-test.timeout=80s', script)

    def test_exchange_inventory_and_old_proof_refused(self):
        source = (ROOT / 'internal/helper/property_exchange_freebsd_test.go').read_text()
        cases = re.findall(r'"([a-z-]+)"', re.search(r'cases := \[\]string\{([^}]+)\}', source).group(1))
        self.assertEqual(len(cases), len(self.module.EXCHANGE_CASES))
        self.assertEqual(set(cases), self.module.EXCHANGE_CASES)
        with self.assertRaises(ValueError):
            self.module.validate(self.peer_only)
        for case in self.module.EXCHANGE_CASES:
            line = '    --- PASS: TestNativePropertyExchange/' + case + ' (0.1s)\n'
            for bad in (self.log.replace(line, ''), self.log + line,
                        self.log.replace(case + '=pass', case + '=fail'),
                        self.log.replace(case + '=pass', ''),
                        self.log + 'PROPERTY_EXCHANGE_RESULT extra=pass\n'):
                with self.assertRaises(ValueError):
                    self.module.validate(bad)

    def test_admitted_inventory_and_uncomposed_proof_refused(self):
        source = (ROOT / 'internal/helper/property_admitted_freebsd_test.go').read_text()
        cases = re.findall(r'"([a-z-]+)"', re.search(r'cases := \[\]string\{([^}]+)\}', source).group(1))
        self.assertEqual(len(cases), len(self.module.ADMITTED_CASES))
        self.assertEqual(set(cases), self.module.ADMITTED_CASES)
        with self.assertRaises(ValueError):
            self.module.validate(self.exchange_only)
        for case in self.module.ADMITTED_CASES:
            line = '    --- PASS: TestNativeAdmittedPropertyExchange/' + case + ' (0.1s)\n'
            for bad in (self.log.replace(line, ''), self.log + line,
                        self.log.replace(case + '=pass', case + '=fail'),
                        self.log.replace(case + '=pass', ''),
                        self.log + 'ADMITTED_PROPERTY_RESULT extra=pass\n'):
                with self.assertRaises(ValueError):
                    self.module.validate(bad)

    def test_session_inventory_and_unguarded_proof_refused(self):
        source = (ROOT / 'internal/helper/property_client_freebsd_test.go').read_text()
        cases = re.findall(r'"([a-z-]+)"', re.search(r'cases := \[\]string\{([^}]+)\}', source).group(1))
        self.assertEqual(len(cases), len(self.module.SESSION_CASES))
        self.assertEqual(set(cases), self.module.SESSION_CASES)
        with self.assertRaises(ValueError):
            self.module.validate(self.admitted_only)
        for case in self.module.SESSION_CASES:
            line = '    --- PASS: TestNativeSessionPropertyClient/' + case + ' (0.1s)\n'
            for bad in (self.log.replace(line, ''), self.log + line,
                        self.log.replace(case + '=pass', case + '=fail'),
                        self.log.replace(case + '=pass', ''),
                        self.log + 'SESSION_PROPERTY_RESULT extra=pass\n'):
                with self.assertRaises(ValueError):
                    self.module.validate(bad)
