# SPDX-License-Identifier: BSD-3-Clause
"""Synthetic parser coverage; never qualifies a native stack or lifecycle."""
import copy
import json
import unittest
from test_closed_startup_evidence import module
from test_policy_facts_evidence import CHILD as FACT_CHILD, PARENT as FACT_PARENT

CHILD = set('cpu_affinity child_budget caller_deadline original_call sampler_joined samples_bounded owner_release deadline_unchanged'.split())
ROOT = set('cpu_affinity parent_budget seed_closed child_waited child_inventory snapshots_verified final_journal_observed ledger_preserved nonjournal_preserved child_locks_released raw_removed domain_teardown mounts_restored'.split()) | {'child_' + x for x in CHILD}

def summary():
    return {'return_ns':120010000000,'caller_deadline_ns':120000000000,'caller_error':'deadline','startup_error':'deadline','owner_present':False,'samples':[{'id':i,'offset_ns':t*1000000000,'begin_ns':t*1000000000+1,'end_ns':t*1000000000+1000,'bytes':100,'sha256':'a'*64,'status':'captured'} for i,t in enumerate((30,75,110),1)]}

def fixture(s=None):
    s=s or summary()
    frames=[{'id':r['id'],'status':'located' if r['status']=='captured' else r['status'],'frames':[{'file':'internal/host/freebsd/execution_restart_freebsd.go','line':26,'symbol':'github.com/gonz0w/horvium/internal/host/freebsd.activateNativeClosedExecutionOwner'}] if r['status']=='captured' else []} for r in s['samples']]
    journal={'before_sha256':'b'*64,'after_sha256':'b'*64,'unchanged':True,'decoded':True,'state':'closed','generation_before':2,'generation_after':2}
    return ('POLICY_STARTUP_TRACE_PROFILE cpu_count=1 child_seconds=150 parent_seconds=480 phase_minutes=18\n'
            '=== RUN   TestNativeMultiplePolicyStartupTraceUFS\n'
            'POLICY_FACTS_CHILD_RESULT '+' '.join(x+'=pass' for x in sorted(FACT_CHILD))+'\n'
            'POLICY_STARTUP_TRACE_SUMMARY '+json.dumps(s)+'\n'+
            ''.join('POLICY_STARTUP_TRACE_STACK '+json.dumps(x)+'\n' for x in frames)+
            'POLICY_STARTUP_TRACE_JOURNAL '+json.dumps(journal)+'\n'
            'POLICY_STARTUP_TRACE_DOMAIN_TEARDOWN ufs filesystems=pass directories=pass mounts=pass\n'
            'POLICY_FACTS_PARENT_RESULT '+' '.join(x+'=pass' for x in sorted(FACT_PARENT))+'\n'
            'POLICY_STARTUP_TRACE_UFS_RESULT '+' '.join(x+'=pass' for x in sorted(ROOT))+'\n'
            '--- PASS: TestNativeMultiplePolicyStartupTraceUFS (200.0s)\nPASS\n')

class PolicyStartupTrace(unittest.TestCase):
    def test_complete_failure_is_diagnostic_only(self):
        m=module('check-policy-startup-trace');self.assertEqual(m.CHILD,CHILD);self.assertEqual(m.ROOT,ROOT)
        result=m.validate(fixture());self.assertFalse(result['native_acceptance']);self.assertEqual(result['summary']['startup_error'],'deadline')

    def test_all_controls_and_whole_parent_required(self):
        m=module('check-policy-startup-trace');log=fixture()
        for bad in ('',log*2,log.replace('--- PASS:','--- FAIL:'),log.replace('PASS\n','SKIP\n'),log.replace('phase_minutes=18','phase_minutes=40'),log.replace('cpu_count=1','cpu_count=2'),log.replace('POLICY_STARTUP_TRACE_DOMAIN_TEARDOWN','MISSING'),log+'=== RUN   TestOther\n'):
            with self.assertRaises(ValueError):m.validate(bad)
        for marker,controls in [('POLICY_STARTUP_TRACE_UFS_RESULT',ROOT),('POLICY_FACTS_CHILD_RESULT',FACT_CHILD),('POLICY_FACTS_PARENT_RESULT',FACT_PARENT)]:
            row=next(x for x in log.splitlines() if x.startswith(marker+' '))
            for c in controls:
                for replacement in ('',c+'=fail',c+'=pass '+c+'=pass'):
                    with self.subTest(marker=marker,c=c),self.assertRaises(ValueError):m.validate(log.replace(row,row.replace(c+'=pass',replacement)))

    def test_skips_require_actual_early_return(self):
        m=module('check-policy-startup-trace');s=summary();s.update(return_ns=1000000000,caller_error='none',startup_error='none',owner_present=True)
        for row in s['samples']:row.update(begin_ns=0,end_ns=0,bytes=0,sha256='',status='skipped')
        m.validate(fixture(s))
        bad=copy.deepcopy(s);bad['return_ns']=120010000000
        with self.assertRaises(ValueError):m.validate(fixture(bad))

    def test_snapshot_shapes_and_bounds(self):
        m=module('check-policy-startup-trace')
        for key,value in [('bytes',65536),('bytes',0),('sha256','private'),('offset_ns',1),('id',2),('status','unknown'),('end_ns',-1)]:
            s=summary();s['samples'][0][key]=value
            with self.subTest(key=key),self.assertRaises(ValueError):m.validate(fixture(s))
        s=summary();s['extra']='secret'
        with self.assertRaises(ValueError):m.validate(fixture(s))
        s=summary();s['owner_present']=True
        with self.assertRaises(ValueError):m.validate(fixture(s))

    def test_raw_or_unsafe_frames_rejected(self):
        m=module('check-policy-startup-trace');log=fixture()
        for bad in [log.replace('internal/host/freebsd/execution_restart_freebsd.go','/private/inventory'),log.replace('activateNativeClosedExecutionOwner"','activateNativeClosedExecutionOwner(0x123)"'),log.replace('"line": 26','"line": 0'),log.replace('"status": "located"','"status": "unknown"'),log.replace('POLICY_STARTUP_TRACE_STACK','MISSING'),log+'goroutine 9 [syscall]:\n']:
            with self.assertRaises(ValueError):m.validate(bad)

    def test_runner_profile_and_independent_teardown(self):
        q=module('local-qualification');self.assertEqual(q.cpu_count_for_suite('policy-startup-trace'),1);self.assertTrue(q.uses_device_fixture('policy-startup-trace'))
        self.assertIsNone(q.lifecycle_budget_for_suite('policy-startup-trace'))
        _,command=q.suite_command('policy-startup-trace',artifact_digest='a'*64);self.assertIn('HORVIUM_POLICY_STARTUP_TRACE_VM=1',command)
        for digest in (None, 'a'*64):
            _,command=q.suite_command('policy-startup-trace',source_build=digest is None,artifact_digest=digest)
            self.assertIn('HORVIUM_CLOSED_STARTUP_OWNER_VM=1',command)
        from pathlib import Path
        script=Path(q.__file__).with_name('bsd-policy-startup-trace-prebuilt.sh').read_text()
        self.assertIn('${HORVIUM_CLOSED_STARTUP_OWNER_VM:-}',script)
        q.validate_result('policy-startup-trace',fixture())
        with self.assertRaises(ValueError):q.validate_result('policy-startup-trace',fixture(),complete=True)
        q.validate_result('policy-startup-trace',fixture()+'DEVICE_FIXTURE_CLEANUP pass\n',complete=True)
