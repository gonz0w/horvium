# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
from pathlib import Path
import unittest
ROOT=Path(__file__).resolve().parents[2]
def module(name):
 spec=importlib.util.spec_from_file_location(name,ROOT/'tools'/f'{name}.py');m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
class ClosedStartupEvidenceTests(unittest.TestCase):
 def setUp(self):
  self.m=module('check-closed-startup-evidence');self.original=module('check-execution-startup-evidence')
 def publication(self):
  text=''
  for kind in ('UFS','ZFS','Cuts'):
   name='TestNativeClosedStartupPublication'+kind;prefix='CLOSED_STARTUP_CUT_RESULT' if kind=='Cuts' else 'CLOSED_STARTUP_PUBLICATION_'+kind+'_RESULT';cases=self.m.CUTS if kind=='Cuts' else self.m.STORAGE
   text+='=== RUN   '+name+'\n'+prefix+' '+' '.join(c+'=pass' for c in sorted(cases))+'\nSTATE_BACKING_RESULT '+name+' filesystem='+('zfs' if kind=='ZFS' else 'ufs')+' softupdates=off teardown=pass\n--- PASS: '+name+' (1.0s)\n'
  return text
 def owner(self):
  text=''
  for old in (True,False):
   for kind in ('UFS','ZFS'):
    name=('TestNativeExecutionStartup' if old else 'TestNativeClosedStartupOwner')+kind;prefix='EXECUTION_STARTUP' if old else 'CLOSED_STARTUP_OWNER';cases=self.original.CASES if old else self.m.OWNER
    text+='=== RUN   '+name+'\n'+prefix+'_'+kind+'_RESULT '+' '.join(c+'=pass' for c in sorted(cases))+'\n'+prefix+'_DOMAIN_TEARDOWN '+kind.lower()+' filesystems=pass directories=pass\n'+prefix+'_DATA_TEARDOWN '+kind.lower()+' restored=pass\n--- PASS: '+name+' (1.0s)\n'
  return text
 def test_independent_inventory(self):
  self.assertEqual(self.m.STORAGE,set('original_closed_review original_active_receipt real_sync_order exact_new_incarnation ledger_unchanged old_review_consumed both_leases_retained original_observation repeated_observation copied_denied decoded_denied cancelled_no_effect cancelled_original_survives release_once released_leases reopen_observation_only ordinary_successor_denied ordinary_reopen_release active_no_takeover active_preserved entropy-short_denied entropy-short_retired entropy-short_retry_preserves entropy-short_release old-identity_denied old-identity_retired old-identity_retry_preserves old-identity_release installation-identity_denied installation-identity_retired installation-identity_retry_preserves installation-identity_release boot-before_denied boot-before_retired boot-before_retry_preserves boot-before_release system-before_denied system-before_retired system-before_retry_preserves system-before_release ledger-before_denied ledger-before_retired ledger-before_retry_preserves ledger-before_release journal-before_denied journal-before_retired journal-before_retry_preserves journal-before_release write_denied write_retired write_retry_preserves write_release syncFile_denied syncFile_retired syncFile_retry_preserves syncFile_release closeFile_denied closeFile_retired closeFile_retry_preserves closeFile_release verify_denied verify_retired verify_retry_preserves verify_release rename_denied rename_retired rename_retry_preserves rename_release syncDirectory_denied syncDirectory_retired syncDirectory_retry_preserves syncDirectory_release after-sync-drift_denied after-sync-drift_retired after-sync-drift_retry_preserves after-sync-drift_release boot-after_denied boot-after_retired boot-after_retry_preserves boot-after_release system-after_denied system-after_retired system-after_retry_preserves system-after_release journal-after_denied journal-after_retired journal-after_retry_preserves journal-after_release ledger-after_denied ledger-after_retired ledger-after_retry_preserves ledger-after_release cancel-after_denied cancel-after_retired cancel-after_retry_preserves cancel-after_release lost-ack_denied lost-ack_retired lost-ack_retry_preserves lost-ack_release after-sync-directory_denied after-sync-directory_retired after-sync-directory_retry_preserves after-sync-directory_release after-sync-ledger_denied after-sync-ledger_retired after-sync-ledger_retry_preserves after-sync-ledger_release after-sync-lock_denied after-sync-lock_retired after-sync-lock_retry_preserves after-sync-lock_release nonempty_review_metadata_only nonempty_activation_denied nonempty_retired nonempty_history_preserved nonempty_release no-close-capacity_review_metadata_only no-close-capacity_activation_denied no-close-capacity_retired no-close-capacity_history_preserved no-close-capacity_release'.split()));self.assertEqual(self.m.CUTS,set('file-sync_barrier file-sync_killed_reaped file-sync_visible_history file-sync_no_startup_takeover file-sync_history_preserved file-sync_leases_released directory-sync_barrier directory-sync_killed_reaped directory-sync_visible_history directory-sync_no_startup_takeover directory-sync_history_preserved directory-sync_leases_released acknowledged_barrier acknowledged_killed_reaped acknowledged_visible_history acknowledged_no_startup_takeover acknowledged_history_preserved acknowledged_leases_released uid_barrier uid_killed_reaped uid_visible_history uid_closed_review_only uid_history_preserved uid_leases_released'.split()));self.assertEqual(self.m.OWNER,set('seed_closed_receipt seed_active_receipt journal_ledger_independent declared_disk_disjoint closed_observation_metadata_only closed_history_final active_history_final data_sentinel_preserved domain_teardown mounts_restored clean_child_waited clean_nil_no_creation clean_cancelled_no_creation clean_original_registered clean_both_locks_held clean_fresh_active_identity clean_binding_census_preserved clean_original_observation clean_copied_denied clean_decoded_denied clean_cancelled_no_effect clean_other_startup_denied clean_other_commission_denied clean_finish_denied clean_finish_retires clean_release_once clean_active_preserved clean_registration_not_reset clean_parent_locks_released clean_parent_history touch_child_waited touch_original_registered touch_directory_drift_denied touch_observation_retired touch_release_once touch_active_preserved touch_registration_not_reset touch_parent_locks_released touch_parent_history uid_child_waited uid_original_registered uid_actual_nonroot uid_inherited_root_descriptors uid_inherited_operations_denied uid_active_preserved uid_nonroot_release_denied uid_parent_locks_released uid_parent_history active_child_waited active_active_denied active_active_preserved active_both_locks_released active_registration_not_reset active_parent_locks_released active_parent_history missing_child_waited missing_missing_denied missing_no_creation missing_registration_not_reset missing_retry_denied boot_child_waited boot_fault_denied boot_closed_preserved boot_both_locks_released boot_registration_not_reset boot_parent_locks_released boot_parent_history system_child_waited system_fault_denied system_closed_preserved system_both_locks_released system_registration_not_reset system_parent_locks_released system_parent_history post-system_child_waited post-system_fault_denied post-system_fresh_active_preserved post-system_both_locks_released post-system_registration_not_reset post-system_parent_locks_released post-system_parent_history post-touch_child_waited post-touch_fault_denied post-touch_fresh_active_preserved post-touch_both_locks_released post-touch_registration_not_reset post-touch_parent_locks_released post-touch_parent_history'.split()))
 def test_publication_and_actual_teardown(self):
  log=self.publication();self.m.validate(log,'publication')
  for bad in ['',log*2,log.replace('PASS','SKIP'),log.replace('PASS','FAIL'),log.replace('STATE_BACKING_RESULT','missing'),log+'CLOSED_STARTUP_CUT_RESULT extra=pass\n',log+'--- PASS: TestUnknown (1.0s)\n']:
   with self.assertRaises(ValueError):self.m.validate(bad,'publication')
  for c in self.m.STORAGE|self.m.CUTS:
   for value in ['',c+'=fail',c+'=pass '+c+'=pass']:
    with self.subTest(c=c),self.assertRaises(ValueError):self.m.validate(log.replace(c+'=pass',value),'publication')
 def test_owner_retains_every_old_startup_case(self):
  log=self.owner();self.m.validate(log,'owner')
  for bad in ['',log*2,log.replace('PASS','SKIP'),log.replace('PASS','FAIL'),log.replace('DOMAIN_TEARDOWN','missing'),log.replace('DATA_TEARDOWN','missing'),log+'CLOSED_STARTUP_OWNER_UFS_RESULT extra=pass\n']:
   with self.assertRaises(ValueError):self.m.validate(bad,'owner')
  for c in self.m.OWNER|self.original.CASES:
   for value in ['',c+'=fail',c+'=pass '+c+'=pass']:
    with self.subTest(c=c),self.assertRaises(ValueError):self.m.validate(log.replace(c+'=pass',value),'owner')
 def test_registered_artifacts_and_cleanup(self):
  m=module('local-qualification')
  for scope,package,device,log in [('publication','./internal/storage',False,self.publication()),('owner','./internal/host/freebsd',True,self.owner())]:
   suite='closed-startup-'+scope;self.assertEqual(m.PREBUILT[suite][1],package);self.assertEqual(m.uses_device_fixture(suite),device);m.validate_result(suite,log)
   if device:
    for bad in [log,log+'DEVICE_FIXTURE_CLEANUP fail\n',log+'DEVICE_FIXTURE_CLEANUP pass\n'*2]:
     with self.assertRaises(ValueError):m.validate_result(suite,bad,complete=True)
    log+='DEVICE_FIXTURE_CLEANUP pass\n'
   m.validate_result(suite,log,complete=True);m.validate_prebuilt('CLOSED_STARTUP_'+scope.upper()+'_PREBUILT_SHA256 '+'a'*64+'\n','a'*64,suite)
   self.assertIn('bsd-'+suite+'-prebuilt.sh',m.suite_command(suite,False,'a'*64)[1])
   with self.assertRaises(ValueError):m.validate_prebuilt('', 'a'*64,suite)
