# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
import unittest
from pathlib import Path

ROOT=Path(__file__).resolve().parents[2]
SPEC=importlib.util.spec_from_file_location('pool_evidence',ROOT/'tools/check-pool-properties-evidence.py')

class PoolPropertiesEvidenceTests(unittest.TestCase):
    def setUp(self):
        self.module=importlib.util.module_from_spec(SPEC)
        SPEC.loader.exec_module(self.module)
        self.log='--- PASS: TestNativePoolPropertyCollection (1.0s)\nPOOL_PROPERTY_COLLECTION_RESULT '+' '.join(c+'=pass' for c in sorted(self.module.CASES))+'\n'

    def test_complete_fixed_inventory_accepted(self):
        self.assertEqual(self.module.CASES,{'complete','pool_identity','dataset_identity','property_values','inheritance','snapshot_owner','features','topology','eligibility_denied','wrong_guid_denied','rename_fresh','readonly_denied','volume_denied','preservation','backing_preserved','teardown','policy_bound','policy_missing_denied','policy_change_denied','policy_scope_denied','policy_rename_denied','policy_closed_denied'})
        self.module.validate(self.log)

    def test_missing_duplicate_failed_or_extra_case_refused(self):
        for case in self.module.CASES:
            for bad in [self.log.replace(case+'=pass',''),self.log.replace(case+'=pass',case+'=fail'),self.log.replace(case+'=pass',case+'=pass '+case+'=pass')]:
                with self.assertRaises(ValueError):self.module.validate(bad)
        with self.assertRaises(ValueError):self.module.validate(self.log.replace('teardown=pass','teardown=pass invented=pass'))

    def test_unexecuted_skipped_failed_duplicate_or_oversized_refused(self):
        for bad in ['',self.log.replace('--- PASS:','--- SKIP:'),self.log.replace('--- PASS:','--- FAIL:'),self.log+self.log,'x'*(1<<20)+self.log]:
            with self.assertRaises(ValueError):self.module.validate(bad)

    def test_script_selects_only_actual_native_collector(self):
        import re
        script=(ROOT/'tools/bsd-pool-properties-prebuilt.sh').read_text()
        selected=re.findall(r"-test.run='([^']+)'",script)
        self.assertEqual(selected,['^TestNativePoolPropertyCollection$'])
        self.assertIn('func TestNativePoolPropertyCollection(', (ROOT/'internal/host/freebsd/pool_properties_freebsd_test.go').read_text())
