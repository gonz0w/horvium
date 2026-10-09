#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Require joint native ownership plus retained original-lease safety cases."""
from pathlib import Path
import importlib.util
import re
import sys

CASES = {'kernel_uid_denial', 'nil_reader', 'journal_lock_not_created', 'journal_namespace_drift', 'native_pair', 'cancelled', 'child_reaped', 'cross_domain_drift', 'in_call_cancel', 'journal_busy', 'journal_first_acquisition', 'ledger_lock_not_created', 'partial_open_cleanup', 'aliased_domains_denied', 'ledger_contention', 'original_survives_copy', 'bounded_pair_context', 'exact_ledger_census_drift', 'close_excluded', 'copied_close', 'close_finishes', 'reader_error', 'pair_open', 'repeat', 'reopen_new_origin', 'reader_error_retirement', 'invalid_installation', 'observation_finishes', 'nil_owner', 'closed_denial', 'close_idempotent', 'copied_owner', 'drift_retirement', 'actual_backing_pair', 'both_locks_released', 'ledger_busy', 'nil_context', 'journal_first_probes', 'inert_owner', 'first_close'}

def validate(text):
    if len(text)>1<<20 or re.search(r'(?m)^(?:FAIL|\s*--- (?:FAIL|SKIP):)',text):
        raise ValueError('failed/skipped/oversized joint evidence')
    chunks=re.findall(r'(?ms)^=== RUN   (TestNative(?:OriginalBacking|JointExecutionLeases)(?:UFS|ZFS))\n(.*?)^--- PASS: (Test\w+) \([0-9.]+s\)$',text)
    wanted={'TestNative'+family+kind for family in ('OriginalBacking','JointExecutionLeases') for kind in ('UFS','ZFS')}
    if len(chunks)!=4 or {name for name,_,_ in chunks}!=wanted or any(name!=end for name,_,end in chunks):
        raise ValueError('four distinct actual native parents required')
    parents=re.findall(r'(?m)^--- PASS: (Test\w+) \([0-9.]+s\)$',text)
    if len(parents)!=4 or set(parents)!=wanted:
        raise ValueError('unknown or repeated native parent')
    original=''.join('=== RUN   '+name+'\n'+body+'--- PASS: '+name+' (0.1s)\n' for name,body,_ in chunks if 'OriginalBacking' in name)
    spec=importlib.util.spec_from_file_location('retained_original',Path(__file__).with_name('check-original-backing-evidence.py'))
    module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module);module.validate(original)
    for name,body,_ in chunks:
        if 'JointExecutionLeases' not in name:continue
        kind='ZFS' if name.endswith('ZFS') else 'UFS'
        cases=CASES | ({'same_dataset_denial'} if kind=='ZFS' else set())
        rows=re.findall(r'JOINT_EXECUTION_'+kind+r'_RESULT ([^\n]+)',body)
        if len(rows)!=1 or len(rows[0].split())!=len(cases) or set(rows[0].split())!={c+'=pass' for c in cases}:
            raise ValueError('missing/duplicate/unknown/failed joint assertion')
        teardown=re.findall(r'PERSISTENT_BACKING_TEARDOWN ([^\n]+)',body)
        if teardown!=[kind.lower()+' filesystem=pass device=pass directory=pass']:
            raise ValueError('actual owned joint fixture teardown required')
    if len(re.findall(r'JOINT_EXECUTION_(?:UFS|ZFS)_RESULT ',text))!=2 or len(re.findall(r'PERSISTENT_BACKING_TEARDOWN ',text))!=4:
        raise ValueError('extra or incomplete native evidence')

if __name__=='__main__':validate(Path(sys.argv[1]).read_text())
