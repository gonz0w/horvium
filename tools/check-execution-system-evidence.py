#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Actual joint/system assertions and every retained original lease gate."""
import importlib.util
from pathlib import Path
import re
import sys
CASES=set('pair_open actual_system_pair repeat ordinary_observation_separate actual_disjoint_backing nil_system nil_context cancelled_before copied_owner unresolved_owner_denied unresolved_retired reopen_observation_only system_error error_retired zero_system_denied final_system_error system_phase_ledger_drift system_use_drift close_excluded_system system_observation_finishes close_finishes ledger_preserved data_sentinel_preserved domain_teardown mounts_restored'.split())
ZFS_CASES={'owned_metadata_pool_denied','journal_data_mount_denied','ledger_data_mount_denied'}
def validate(text):
 if len(text)>2<<20 or re.search(r'(?m)^(?:FAIL|\s*--- (?:FAIL|SKIP):)',text):raise ValueError('failed/skipped/oversized system evidence')
 chunks=re.findall(r'(?ms)^=== RUN   (TestNative(?:OriginalBacking|JointExecutionLeases|ExecutionSystem)(?:UFS|ZFS))\n(.*?)^--- PASS: (Test\w+) \(([0-9.]+)s\)$',text)
 wanted={'TestNative'+family+kind for family in ('OriginalBacking','JointExecutionLeases','ExecutionSystem') for kind in ('UFS','ZFS')}
 if len(chunks)!=6 or {n for n,_,_,_ in chunks}!=wanted or any(n!=end for n,_,end,_ in chunks):raise ValueError('six distinct actual parents required')
 if re.findall(r'(?m)^--- PASS: (Test\w+) \([0-9.]+s\)$',text)!=[n for n,_,_,_ in chunks]:raise ValueError('unknown/repeated parent')
 retained=''.join('=== RUN   '+n+'\n'+b+'--- PASS: '+n+' ('+secs+'s)\n' for n,b,_,secs in chunks if 'ExecutionSystem' not in n)
 spec=importlib.util.spec_from_file_location('joint',Path(__file__).with_name('check-joint-leases-evidence.py'));m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);m.validate(retained)
 for n,b,_,_ in chunks:
  if 'ExecutionSystem' not in n:continue
  kind='zfs' if n.endswith('ZFS') else 'ufs'
  cases=CASES | (ZFS_CASES if kind=='zfs' else set())
  rows=re.findall(r'EXECUTION_SYSTEM_'+kind.upper()+r'_RESULT ([^\n]+)',b)
  if len(rows)!=1 or len(rows[0].split())!=len(cases) or set(rows[0].split())!={c+'=pass' for c in cases}:raise ValueError('missing/duplicate/failed assertion')
  for marker,expected in [('DOMAIN','filesystems=pass directories=pass'),('DATA','restored=pass')]:
   if re.findall(r'EXECUTION_SYSTEM_'+marker+r'_TEARDOWN ([^\n]+)',b)!=[kind+' '+expected]:raise ValueError('owned fixture teardown required')
 for marker,count in [('EXECUTION_SYSTEM_DOMAIN_TEARDOWN ',2),('EXECUTION_SYSTEM_DATA_TEARDOWN ',2),('PERSISTENT_BACKING_TEARDOWN ',4),('ORIGINAL_BACKING_(?:UFS|ZFS)_RESULT ',2),('JOINT_EXECUTION_(?:UFS|ZFS)_RESULT ',2),('EXECUTION_SYSTEM_(?:UFS|ZFS)_RESULT ',2)]:
  if len(re.findall(marker,text))!=count:raise ValueError('extra/missing native marker')
if __name__=='__main__':validate(Path(sys.argv[1]).read_text())
