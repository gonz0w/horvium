# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
from pathlib import Path
import unittest
ROOT=Path(__file__).resolve().parents[2]
def module(name):
 spec=importlib.util.spec_from_file_location(name,ROOT/'tools'/f'{name}.py');m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
class ExecutionStartupEvidenceTests(unittest.TestCase):
 def setUp(self):
  self.m=module('check-execution-startup-evidence');self.log=''
  for kind in ('ZFS','UFS'):
   n='TestNativeExecutionStartup'+kind
   self.log+='=== RUN   '+n+'\nEXECUTION_STARTUP_'+kind+'_RESULT '+' '.join(c+'=pass' for c in sorted(self.m.CASES))+'\nEXECUTION_STARTUP_DOMAIN_TEARDOWN '+kind.lower()+' filesystems=pass directories=pass\nEXECUTION_STARTUP_DATA_TEARDOWN '+kind.lower()+' restored=pass\n--- PASS: '+n+' (1.0s)\n'
 def test_exact_independent_inventory(self):
  expected=set('seed_closed_receipt seed_active_receipt journal_ledger_independent declared_disk_disjoint closed_observation_metadata_only closed_history_final active_history_final data_sentinel_preserved mounts_restored domain_teardown clean_child_waited clean_nil_no_creation clean_cancelled_no_creation clean_closed_registered clean_both_locks_held clean_original_observation clean_repeated_observation clean_copied_denied clean_decoded_denied clean_cancelled_observe_no_effect clean_other_startup_no_creation clean_other_commission_no_creation clean_finish_denied clean_finished_observe_denied clean_closed_both_locks_released clean_closed_registration_not_reset clean_closed_history_preserved clean_parent_locks_released clean_parent_history touch_child_waited touch_closed_registered touch_ledger_touch_denied touch_drift_retired touch_finish_denied touch_release touch_closed_history_preserved touch_both_locks_released touch_no_commission_after_failure touch_parent_locks_released touch_parent_history active_child_waited active_active_denied active_active_history_preserved active_both_locks_released active_no_commission_after_failure active_parent_locks_released active_parent_history missing_child_waited missing_missing_denied missing_no_creation missing_no_commission_after_failure missing_no_retry_creation drift_child_waited drift_closed_registered drift_ledger_drift_denied drift_drift_retired drift_finish_denied drift_release drift_closed_history_preserved drift_both_locks_released drift_no_commission_after_failure drift_parent_locks_released drift_parent_history uid_child_waited uid_closed_registered uid_actual_nonroot uid_inherited_root_descriptors uid_inherited_operations_denied uid_closed_history_preserved uid_nonroot_release_denied uid_parent_locks_released uid_parent_history boot_child_waited boot_injected_error_denied boot_closed_history_preserved boot_both_locks_released boot_no_commission_after_failure boot_parent_locks_released boot_parent_history system_child_waited system_injected_error_denied system_closed_history_preserved system_both_locks_released system_no_commission_after_failure system_parent_locks_released system_parent_history late_child_waited late_injected_error_denied late_closed_history_preserved late_both_locks_released late_no_commission_after_failure late_parent_locks_released late_parent_history'.split());self.assertEqual(self.m.CASES,expected);self.assertEqual(len(expected),93)
 def test_actual_both_parents_and_teardown(self):
  self.m.validate(self.log)
  for bad in ['',self.log*2,self.log.replace('PASS','SKIP'),self.log.replace('PASS','FAIL'),self.log.replace('DATA_TEARDOWN','missing'),self.log.replace('DOMAIN_TEARDOWN','missing'),self.log+'EXECUTION_STARTUP_UFS_RESULT extra=pass\n',self.log+'--- PASS: TestUnknown (1.0s)\n']:
   with self.assertRaises(ValueError):self.m.validate(bad)
  for c in self.m.CASES:
   for replace in ['',c+'=fail',c+'=pass '+c+'=pass']:
    with self.subTest(c=c),self.assertRaises(ValueError):self.m.validate(self.log.replace(c+'=pass',replace))
 def test_bound_artifact_devices_and_completion(self):
  self.m.validate(self.log)
  m=module('local-qualification');self.assertTrue(m.uses_device_fixture('execution-startup'));self.assertEqual(m.PREBUILT['execution-startup'][1],'./internal/host/freebsd')
  m.validate_result('execution-startup',self.log)
  for log in [self.log,self.log+'DEVICE_FIXTURE_CLEANUP fail\n',self.log+'DEVICE_FIXTURE_CLEANUP pass\n'*2]:
   with self.assertRaises(ValueError):m.validate_result('execution-startup',log,complete=True)
  m.validate_result('execution-startup',self.log+'DEVICE_FIXTURE_CLEANUP pass\n',complete=True);m.validate_prebuilt('EXECUTION_STARTUP_PREBUILT_SHA256 '+'a'*64+'\n','a'*64,'execution-startup')
  with self.assertRaises(ValueError):m.validate_prebuilt('', 'a'*64,'execution-startup')
  self.assertIn('bsd-execution-startup-prebuilt.sh',m.suite_command('execution-startup',False,'a'*64)[1])
 def test_old_host_evidence_cannot_qualify_startup(self):
  with self.assertRaises(ValueError):self.m.validate('--- PASS: TestNativeExecutionSystemUFS (1.0s)\n')
