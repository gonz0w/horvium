# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
import re
import unittest
from pathlib import Path

ROOT=Path(__file__).resolve().parents[2]
SPEC=importlib.util.spec_from_file_location('status_evidence',ROOT/'tools/check-storage-job-status-evidence.py')

class StorageJobStatusEvidenceTests(unittest.TestCase):
    def setUp(self):
        self.module=importlib.util.module_from_spec(SPEC)
        SPEC.loader.exec_module(self.module)
        self.log='--- PASS: TestNativeStorageJobStatus (1.0s)\nSTORAGE_JOB_STATUS_RESULT '+' '.join(c+'=pass' for c in sorted(self.module.CASES))+'\nSTORAGE_JOB_STATUS_BACKING filesystem=zfs softupdates=off teardown=pass\n'

    def test_complete_fixed_inventory(self):
        self.assertEqual(self.module.CASES,{'built_process','empty','mixed_complete','queued_held','applying_held','observing_held','terminal_retained','private_baseline_hidden','repeat_preserved','lease_released','active_writer_denied','kernel_uid_denied','unknown_denied','newer_denied','unsafe_file_denied','orphan_denied','cancelled_denied','output_cancelled','native_unavailable_not_consulted','lock_preserved','fixtures_teardown'})
        self.module.validate(self.log)

    def test_missing_duplicate_failed_or_extra_assertions(self):
        for case in self.module.CASES:
            for bad in [self.log.replace(case+'=pass',''),self.log.replace(case+'=pass',case+'=fail'),self.log.replace(case+'=pass',case+'=pass '+case+'=pass')]:
                with self.assertRaises(ValueError):self.module.validate(bad)
        with self.assertRaises(ValueError):self.module.validate(self.log.replace('fixtures_teardown=pass','fixtures_teardown=pass extra=pass'))

    def test_actual_native_parent_and_unique_result_required(self):
        for bad in ['',self.log.replace('--- PASS:','--- SKIP:'),self.log.replace('--- PASS:','--- FAIL:'),self.log+self.log,'x'*(1<<20)+self.log,self.log.replace('TestNativeStorageJobStatus','TestStorageJobStatusCommandProcess')]:
            with self.assertRaises(ValueError):self.module.validate(bad)

    def test_backing_record_is_required(self):
        for bad in [self.log.replace('filesystem=zfs','filesystem=unknown'),self.log.replace('teardown=pass','teardown=fail'),self.log+'STORAGE_JOB_STATUS_BACKING filesystem=zfs softupdates=off teardown=pass\n']:
            with self.assertRaises(ValueError):self.module.validate(bad)

    def test_native_emitted_cases_match(self):
        source=(ROOT/'cmd/horvium/storage_jobs_freebsd_test.go').read_text()
        cases=re.findall(r'"([a-z_]+)"',re.search(r'cases := \[\]string\{([^}]+)\}',source).group(1))
        self.assertEqual(set(cases),self.module.CASES)
        self.assertEqual(len(cases),len(self.module.CASES))

    def test_script_selects_only_native_parent(self):
        script=(ROOT/'tools/bsd-storage-job-status-prebuilt.sh').read_text()
        self.assertEqual(re.findall(r"-test.run='([^']+)'",script),['^TestNativeStorageJobStatus$'])
        self.assertIn('func TestNativeStorageJobStatus(', (ROOT/'cmd/horvium/storage_jobs_freebsd_test.go').read_text())
