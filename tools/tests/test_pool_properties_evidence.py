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
        self.legacy_log='--- PASS: TestNativePoolPropertyCollection (1.0s)\nPOOL_PROPERTY_COLLECTION_RESULT '+' '.join(c+'=pass' for c in sorted(self.module.CASES))+'\n'
        self.log=self.legacy_log+self.claim_log()

    def claim_log(self):
        parent='TestNativePropertyClaimCollection'
        names=['independent-backing','queued','applying','foreign-selection','closed','reopened','preserved','teardown']
        return ('--- PASS: '+parent+' (1.0s)\n    --- PASS: '+parent+'/native-claim-source (1.0s)\n'
                + ''.join('        --- PASS: '+parent+'/native-claim-source/'+n+' (0.1s)\n' for n in names)
                + 'CLAIM_PROPERTY_COLLECTION_RESULT fixture_teardown=pass\n')

    def test_legacy_only_log_is_incomplete(self):
        with self.assertRaises(ValueError):self.module.validate(self.legacy_log)

    def test_claim_cases_require_complete_once_only_execution(self):
        full=self.log
        self.module.validate(full)
        for name in ['independent-backing','queued','applying','foreign-selection','closed','reopened','preserved','teardown']:
            line='        --- PASS: TestNativePropertyClaimCollection/native-claim-source/'+name+' (0.1s)\n'
            for bad in [full.replace(line,''),full.replace(line,line*2),full.replace(line,line.replace('PASS','FAIL')),full.replace(line,line.replace('PASS','SKIP'))]:
                with self.subTest(name=name),self.assertRaises(ValueError):self.module.validate(bad)
        for bad in [full.replace('CLAIM_PROPERTY_COLLECTION_RESULT fixture_teardown=pass\n',''),full.replace('fixture_teardown=pass','fixture_teardown=fail')]:
            with self.assertRaises(ValueError):self.module.validate(bad)

    def test_complete_fixed_inventory_accepted(self):
        self.assertEqual(self.module.CASES,{'complete','pool_identity','dataset_identity','property_values','inheritance','snapshot_owner','features','topology','eligibility_denied','wrong_guid_denied','rename_fresh','readonly_denied','volume_denied','preservation','backing_preserved','teardown','policy_bound','policy_missing_denied','policy_change_denied','policy_scope_denied','policy_rename_denied','policy_closed_denied','system_root_backing','system_pool_backing','safety_bound','safety_system_mount_denied','safety_fresh_restored','system_swap_observed','safety_swap_disjoint','system_mount_preserved','system_fixture_teardown','unsupported_md_denied','providers_correlated','provider_geometry','providers_duplicate_denied','providers_missing_denied','provider_fixtures_teardown','enrollment_bound','enrollment_legacy_denied','enrollment_member_drift_denied','enrollment_newer_denied','enrollment_restored','enrollment_catalog','enrollment_legacy_read','enrollment_publication','enrollment_publication_preserved','state_backing_gate','state_composed_gate','state_policy_drift_denied','state_unknown_denied','state_job_hold_denied','state_terminal_hold_denied','state_orphan_denied','state_uncertainty_latched','state_preserved','state_missing_policy_denied','enrollment_review_bound','enrollment_review_expired','enrollment_review_context_denied','enrollment_review_policy_drift','enrollment_review_native_drift','enrollment_review_missing_scope','enrollment_review_immutable','enrollment_review_preserved','enrollment_state_bound','enrollment_state_absent','enrollment_state_predecessor_denied','enrollment_state_record_drift','enrollment_state_unrelated_drift','enrollment_state_unknown_denied','enrollment_state_job_denied','enrollment_state_orphan_denied','enrollment_state_expired_denied','enrollment_state_context_denied','enrollment_state_preserved','enrollment_state_fixture_teardown','enrollment_source_review','enrollment_source_revalidate','enrollment_source_policy_denied','enrollment_source_member_denied','enrollment_source_cancelled','enrollment_source_context_denied','enrollment_source_store_uncertain','enrollment_source_preserved','enrollment_publish_native_first','enrollment_publish_native_successor','enrollment_publish_native_repeat_denied','enrollment_publish_native_preserved'})
        self.assertTrue({'providers_correlated','provider_geometry','providers_duplicate_denied','providers_missing_denied','provider_fixtures_teardown'}.issubset(self.module.CASES))
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
        self.assertEqual(selected,['^TestNative(PoolPropertyCollection|PropertyClaimCollection)$'])
        self.assertIn('func TestNativePropertyClaimCollection(', (ROOT/'internal/host/freebsd/property_claim_source_freebsd_test.go').read_text())
        self.assertIn('135*time.Second', (ROOT/'internal/host/freebsd/property_claim_source_freebsd_test.go').read_text())
        self.assertIn('5*time.Minute', (ROOT/'internal/host/freebsd/pool_properties_freebsd_test.go').read_text())
        self.assertIn('func TestNativePoolPropertyCollection(', (ROOT/'internal/host/freebsd/pool_properties_freebsd_test.go').read_text())

    def test_guest_emitted_inventory_matches_host_checker(self):
        import re
        native=(ROOT/'internal/host/freebsd/pool_properties_freebsd_test.go').read_text()
        blocks=re.findall(r'cases := \[\]string\{([^}]+)\}|cases = append\(cases,([^)]*)\)',native)
        cases=[name for block in blocks for part in block for name in re.findall(r'"([a-z_]+)"',part)]
        self.assertEqual(len(cases),len(self.module.CASES))
        self.assertEqual(set(cases),self.module.CASES)
