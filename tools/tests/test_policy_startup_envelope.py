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
    return {'return_ns':180010000000,'caller_deadline_ns':180000000000,'caller_error':'deadline','startup_error':'deadline','owner_present':False,'samples':[{'id':i,'offset_ns':t*1000000000,'begin_ns':t*1000000000+1,'end_ns':t*1000000000+1000,'bytes':100,'sha256':'a'*64,'status':'captured'} for i,t in enumerate((30,75,110),1)]}

def fixture(s=None):
    s=s or summary()
    frames=[{'id':r['id'],'status':'located' if r['status']=='captured' else r['status'],'frames':[{'file':'internal/host/freebsd/execution_restart_freebsd.go','line':26,'symbol':'github.com/gonz0w/horvium/internal/host/freebsd.activateNativeClosedExecutionOwner'}] if r['status']=='captured' else []} for r in s['samples']]
    journal={'before_sha256':'b'*64,'after_sha256':'b'*64,'unchanged':True,'decoded':True,'state':'closed','generation_before':2,'generation_after':2}
    return ('POLICY_STARTUP_ENVELOPE_PROFILE cpu_count=1 child_seconds=210 parent_seconds=480 phase_minutes=18\n'
            '=== RUN   TestNativeMultiplePolicyStartupEnvelopeUFS\n'
            'POLICY_FACTS_CHILD_RESULT '+' '.join(x+'=pass' for x in sorted(FACT_CHILD))+'\n'
            'POLICY_STARTUP_ENVELOPE_SUMMARY '+json.dumps(s)+'\n'+
            ''.join('POLICY_STARTUP_ENVELOPE_STACK '+json.dumps(x)+'\n' for x in frames)+
            'POLICY_STARTUP_ENVELOPE_JOURNAL '+json.dumps(journal)+'\n'
            'POLICY_STARTUP_ENVELOPE_DOMAIN_TEARDOWN ufs filesystems=pass directories=pass mounts=pass\n'
            'POLICY_FACTS_PARENT_RESULT '+' '.join(x+'=pass' for x in sorted(FACT_PARENT))+'\n'
            'POLICY_STARTUP_ENVELOPE_UFS_RESULT '+' '.join(x+'=pass' for x in sorted(ROOT))+'\n'
            '--- PASS: TestNativeMultiplePolicyStartupEnvelopeUFS (200.0s)\nPASS\n')

class PolicyStartupEnvelope(unittest.TestCase):
    def test_complete_failure_is_diagnostic_only(self):
        m=module('check-policy-startup-envelope');self.assertEqual(m.CHILD,CHILD);self.assertEqual(m.ROOT,ROOT)
        result=m.validate(fixture());self.assertFalse(result['native_acceptance']);self.assertEqual(result['summary']['startup_error'],'deadline')

    def test_all_controls_and_whole_parent_required(self):
        m=module('check-policy-startup-envelope');log=fixture()
        for bad in ('',log*2,log.replace('--- PASS:','--- FAIL:'),log.replace('PASS\n','SKIP\n'),log.replace('phase_minutes=18','phase_minutes=40'),log.replace('cpu_count=1','cpu_count=2'),log.replace('POLICY_STARTUP_ENVELOPE_DOMAIN_TEARDOWN','MISSING'),log+'=== RUN   TestOther\n'):
            with self.assertRaises(ValueError):m.validate(bad)
        for marker,controls in [('POLICY_STARTUP_ENVELOPE_UFS_RESULT',ROOT),('POLICY_FACTS_CHILD_RESULT',FACT_CHILD),('POLICY_FACTS_PARENT_RESULT',FACT_PARENT)]:
            row=next(x for x in log.splitlines() if x.startswith(marker+' '))
            for c in controls:
                for replacement in ('',c+'=fail',c+'=pass '+c+'=pass'):
                    with self.subTest(marker=marker,c=c),self.assertRaises(ValueError):m.validate(log.replace(row,row.replace(c+'=pass',replacement)))

    def test_skips_require_actual_early_return(self):
        m=module('check-policy-startup-envelope');s=summary();s.update(return_ns=1000000000,caller_error='none',startup_error='none',owner_present=True)
        for row in s['samples']:row.update(begin_ns=0,end_ns=0,bytes=0,sha256='',status='skipped')
        m.validate(fixture(s))
        bad=copy.deepcopy(s);bad['return_ns']=180010000000
        with self.assertRaises(ValueError):m.validate(fixture(bad))

    def test_snapshot_shapes_and_bounds(self):
        m=module('check-policy-startup-envelope')
        for key,value in [('bytes',65536),('bytes',0),('sha256','private'),('offset_ns',1),('id',2),('status','unknown'),('end_ns',-1)]:
            s=summary();s['samples'][0][key]=value
            with self.subTest(key=key),self.assertRaises(ValueError):m.validate(fixture(s))
        s=summary();s['extra']='secret'
        with self.assertRaises(ValueError):m.validate(fixture(s))
        s=summary();s['owner_present']=True
        with self.assertRaises(ValueError):m.validate(fixture(s))

    def test_raw_or_unsafe_frames_rejected(self):
        m=module('check-policy-startup-envelope');log=fixture()
        for bad in [log.replace('internal/host/freebsd/execution_restart_freebsd.go','/private/inventory'),log.replace('activateNativeClosedExecutionOwner"','activateNativeClosedExecutionOwner(0x123)"'),log.replace('"line": 26','"line": 0'),log.replace('"status": "located"','"status": "unknown"'),log.replace('POLICY_STARTUP_ENVELOPE_STACK','MISSING'),log+'goroutine 9 [syscall]:\n']:
            with self.assertRaises(ValueError):m.validate(bad)

    def test_runner_profile_and_independent_teardown(self):
        q=module('local-qualification');self.assertEqual(q.cpu_count_for_suite('policy-startup-envelope'),1);self.assertTrue(q.uses_device_fixture('policy-startup-envelope'))
        self.assertIsNone(q.lifecycle_budget_for_suite('policy-startup-envelope'))
        _,command=q.suite_command('policy-startup-envelope',artifact_digest='a'*64);self.assertIn('HORVIUM_POLICY_STARTUP_ENVELOPE_VM=1',command)
        for digest in (None, 'a'*64):
            _,command=q.suite_command('policy-startup-envelope',source_build=digest is None,artifact_digest=digest)
            self.assertIn('HORVIUM_CLOSED_STARTUP_OWNER_VM=1',command)
        from pathlib import Path
        script=Path(q.__file__).with_name('bsd-policy-startup-envelope-prebuilt.sh').read_text()
        self.assertIn('${HORVIUM_CLOSED_STARTUP_OWNER_VM:-}',script)
        q.validate_result('policy-startup-envelope',fixture())
        with self.assertRaises(ValueError):q.validate_result('policy-startup-envelope',fixture(),complete=True)
        q.validate_result('policy-startup-envelope',fixture()+'DEVICE_FIXTURE_CLEANUP pass\n',complete=True)

    def test_original_and_envelope_profiles_are_not_interchangeable(self):
        from test_policy_startup_trace import fixture as old_fixture
        old=module('check-policy-startup-trace');new=module('check-policy-startup-envelope')
        old.validate(old_fixture());new.validate(fixture())
        for checker,log in [(old,fixture()),(new,old_fixture()),(new,fixture().replace('child_seconds=210','child_seconds=150')),(new,fixture().replace('180000000000','120000000000'))]:
            with self.assertRaises(ValueError):checker.validate(log)
        s=summary();s['native_acceptance']=True
        with self.assertRaises(ValueError):new.validate(fixture(s))

    def test_profile_artifact_and_cleanup_are_independent(self):
        import tempfile
        from pathlib import Path
        q=module('local-qualification')
        expected={'child_seconds':210,'parent_seconds':480,'phase_minutes':18,'caller_seconds':180,'product_seconds':90,'sample_seconds':[30,75,110],'snapshot_bytes':65536,'join_seconds':5}
        self.assertEqual(q.startup_envelope_budget_for_suite('policy-startup-envelope'),expected)
        self.assertIsNone(q.startup_envelope_budget_for_suite('policy-startup-trace'))
        marker='POLICY_STARTUP_ENVELOPE_PREBUILT_SHA256 '
        q.validate_prebuilt(marker+'a'*64+'\n','a'*64,'policy-startup-envelope')
        for log in ['',marker+'b'*64+'\n',(marker+'a'*64+'\n')*2,'POLICY_STARTUP_TRACE_PREBUILT_SHA256 '+'a'*64+'\n']:
            with self.assertRaises(ValueError):q.validate_prebuilt(log,'a'*64,'policy-startup-envelope')
        with tempfile.TemporaryDirectory() as name:
            directory=Path(name);binary=directory/'policy-startup-envelope-native.test';binary.touch()
            unrelated=directory/'operator-file';unrelated.touch()
            q.cleanup_prebuilt(directory)
            self.assertFalse(binary.exists());self.assertTrue(unrelated.exists())

    def test_source_identity_does_not_accept_changed_or_dirty_candidate(self):
        from unittest.mock import patch
        q=module('local-qualification')
        with patch.object(q.subprocess,'check_output',side_effect=['a'*40+'\n',b'']):q.source_identity('a'*40)
        for outputs in [['b'*40+'\n'],['a'*40+'\n',b' M changed.go\n']]:
            with patch.object(q.subprocess,'check_output',side_effect=outputs),self.assertRaises(ValueError):q.source_identity('a'*40)

    def test_supervisor_fallback_removes_envelope_artifact_on_interruption(self):
        import tempfile
        from pathlib import Path
        from unittest.mock import patch
        owner=module('owned-process')
        with tempfile.TemporaryDirectory() as name:
            root=Path(name).resolve();lab=root/'.tools/lab-runs/envelope';lab.mkdir(parents=True,mode=0o700)
            binary=lab/'policy-startup-envelope-native.test';binary.write_text('synthetic executable')
            unrelated=lab/'unrelated';unrelated.write_text('preserve')
            (lab/'result.json').write_text(json.dumps({'qualified':True}))
            with patch.object(owner,'ROOT',root):owner.native_cleanup(lab,interrupted=True)
            self.assertFalse(binary.exists());self.assertEqual(unrelated.read_text(),'preserve')
            result=json.loads((lab/'result.json').read_text());self.assertFalse(result['qualified']);self.assertTrue(result['supervisor_fixture_cleanup'])
