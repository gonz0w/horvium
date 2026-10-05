# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
import unittest
from pathlib import Path

SOURCE = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location('property_evidence', SOURCE / 'tools/check-property-evidence.py')
RESULT = ('PROPERTY_OBSERVATION_RESULT default=pass local=pass inherited=pass '
          'received=pass shadowed=pass no_limit=pass rename=pass name_reuse=pass '
          'foreign_preservation=pass eligibility_denied=pass teardown=pass '
          'identity_binding=pass complete_filesystems=pass snapshot_owner=pass '
          'bookmark_owner=pass replicated_guid=pass snapshot_rename=pass '
          'snapshot_reuse=pass volume_denial=pass stable_enumeration=pass '
          'state_binding=pass mount_sources=pass readonly_denial=pass '
          'encrypted_loaded_denial=pass encrypted_unloaded_denial=pass '
          'encryption_root_binding=pass state_drift=pass state_preservation=pass '
          'pool_identity_binding=pass pool_source_metadata=pass pool_bootfs_denial=pass '
          'pool_readonly_denial=pass import_incarnation=pass reimport_drift=pass '
          'pool_preservation=pass pool_name_reuse=pass feature_binding=pass '
          'feature_sources=pass feature_preservation=pass installed_profile=pass '
          'common_feature_set=pass feature_drift=pass')
PASS = '--- PASS: TestNativePropertyObservations (1.0s)\n'

class PropertyEvidenceTests(unittest.TestCase):
    def validate(self, text):
        module = importlib.util.module_from_spec(SPEC)
        SPEC.loader.exec_module(module)
        module.validate(text)

    def test_complete_native_assertions_accepted(self):
        self.validate('    properties_freebsd_test.go:90: ' + RESULT + '\n' + PASS)

    def test_boot_unit_pass_skip_or_empty_cannot_qualify(self):
        for text in ['', 'VM booted\nPASS\n', RESULT, PASS,
                     RESULT + '\n--- SKIP: TestNativePropertyObservations (0.0s)\n',
                     RESULT + '\n' + PASS + 'FAIL\n']:
            with self.subTest(text=text), self.assertRaises(ValueError):
                self.validate(text)

    def test_duplicate_missing_unknown_or_failed_cases_denied(self):
        for record in [RESULT + '\n' + RESULT,
                       RESULT.replace('teardown=pass', ''),
                       RESULT.replace('identity_binding=pass', ''),
                       RESULT.replace('state_binding=pass', ''),
                       RESULT.replace('import_incarnation=pass', ''),
                       RESULT.replace('installed_profile=pass', ''),
                       RESULT.replace('teardown=pass', 'teardown=fail'),
                       RESULT + ' override=pass', RESULT + ' teardown=pass']:
            with self.subTest(record=record), self.assertRaises(ValueError):
                self.validate(record + '\n' + PASS)
