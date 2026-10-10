#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Require original closed startup review and independent owned teardown."""
from pathlib import Path
import re
import sys
CASES=set('seed_closed_receipt seed_active_receipt journal_ledger_independent declared_disk_disjoint closed_observation_metadata_only closed_history_final active_history_final data_sentinel_preserved mounts_restored domain_teardown clean_child_waited clean_nil_no_creation clean_cancelled_no_creation clean_closed_registered clean_both_locks_held clean_original_observation clean_repeated_observation clean_copied_denied clean_decoded_denied clean_cancelled_observe_no_effect clean_other_startup_no_creation clean_other_commission_no_creation clean_finish_denied clean_finished_observe_denied clean_closed_both_locks_released clean_closed_registration_not_reset clean_closed_history_preserved clean_parent_locks_released clean_parent_history touch_child_waited touch_closed_registered touch_ledger_touch_denied touch_drift_retired touch_finish_denied touch_release touch_closed_history_preserved touch_both_locks_released touch_no_commission_after_failure touch_parent_locks_released touch_parent_history active_child_waited active_active_denied active_active_history_preserved active_both_locks_released active_no_commission_after_failure active_parent_locks_released active_parent_history missing_child_waited missing_missing_denied missing_no_creation missing_no_commission_after_failure missing_no_retry_creation drift_child_waited drift_closed_registered drift_ledger_drift_denied drift_drift_retired drift_finish_denied drift_release drift_closed_history_preserved drift_both_locks_released drift_no_commission_after_failure drift_parent_locks_released drift_parent_history uid_child_waited uid_closed_registered uid_actual_nonroot uid_inherited_root_descriptors uid_inherited_operations_denied uid_closed_history_preserved uid_nonroot_release_denied uid_parent_locks_released uid_parent_history boot_child_waited boot_injected_error_denied boot_closed_history_preserved boot_both_locks_released boot_no_commission_after_failure boot_parent_locks_released boot_parent_history system_child_waited system_injected_error_denied system_closed_history_preserved system_both_locks_released system_no_commission_after_failure system_parent_locks_released system_parent_history late_child_waited late_injected_error_denied late_closed_history_preserved late_both_locks_released late_no_commission_after_failure late_parent_locks_released late_parent_history'.split())
def validate(text):
 if len(text)>1<<20 or re.search(r'(?m)^(?:FAIL|\s*--- (?:FAIL|SKIP):)',text):raise ValueError('failed/skipped/oversized startup evidence')
 chunks=re.findall(r'(?ms)^=== RUN   (TestNativeExecutionStartup(?:UFS|ZFS))\n(.*?)^--- PASS: (Test\w+) \(([0-9.]+)s\)$',text)
 wanted={'TestNativeExecutionStartup'+kind for kind in ('UFS','ZFS')}
 if len(chunks)!=2 or {name for name,_,_,_ in chunks}!=wanted or any(name!=end for name,_,end,_ in chunks):raise ValueError('two distinct actual startup parents required')
 if re.findall(r'(?m)^--- PASS: (Test\w+) \([0-9.]+s\)$',text)!=[name for name,_,_,_ in chunks]:raise ValueError('unknown/repeated startup parent')
 for name,body,_,_ in chunks:
  kind='zfs' if name.endswith('ZFS') else 'ufs'
  rows=re.findall('EXECUTION_STARTUP_'+kind.upper()+r'_RESULT ([^\n]+)',body)
  if len(rows)!=1 or len(rows[0].split())!=len(CASES) or set(rows[0].split())!={c+'=pass' for c in CASES}:raise ValueError('missing/duplicate/failed/unknown startup control')
  for marker,expected in [('DOMAIN','filesystems=pass directories=pass'),('DATA','restored=pass')]:
   if re.findall('EXECUTION_STARTUP_'+marker+r'_TEARDOWN ([^\n]+)',body)!=[kind+' '+expected]:raise ValueError('independent startup fixture teardown required')
 for marker in ['EXECUTION_STARTUP_(?:UFS|ZFS)_RESULT ','EXECUTION_STARTUP_DOMAIN_TEARDOWN ','EXECUTION_STARTUP_DATA_TEARDOWN ']:
  if len(re.findall(marker,text))!=2:raise ValueError('extra/missing startup evidence marker')
if __name__=='__main__':validate(Path(sys.argv[1]).read_text())
