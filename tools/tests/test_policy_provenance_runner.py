# SPDX-License-Identifier: BSD-3-Clause
"""Portable runner coverage with synthetic build/transcript evidence only."""
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
from test_closed_startup_evidence import module
from test_policy_startup_provenance import fixture, SOURCE, COMPILER, ARTIFACT

BEGIN = 'POLICY_STARTUP_PROVENANCE_TRANSCRIPT_BEGIN\n'
END = 'POLICY_STARTUP_PROVENANCE_TRANSCRIPT_END\n'
SUITE = 'policy-startup-provenance'
BINARY = 'policy-startup-provenance-native.test'
NORMAL = 'policy-startup-provenance-normal.test'


def build():
    return dict(source_commit=SOURCE, compiler_sha256=COMPILER, sha256=ARTIFACT,
                target='freebsd/amd64', goamd64='v1', cgo_enabled=False,
                build_arguments=['go', 'test', '-c', '-trimpath', '-tags=horvium_native_fixture', '-o', '<owned-artifact>', './internal/host/freebsd'],
                source_isolation=dict(schema=1, normal_binary_sha256='d'*64,
                    normal={'storage': {'go_files':['deadline_return_plain.go'], 'test_files':[]},
                            'host': {'go_files':[], 'test_files':[]}},
                    tagged={'storage': {'go_files':['deadline_return_fixture.go','deadline_semantics_fixture.go'], 'test_files':[]},
                            'host': {'go_files':[], 'test_files':['execution_policy_startup_provenance_fixture_test.go','execution_policy_provenance_support_test.go','execution_policy_provenance_support_fixture_test.go']}}))


def log(record=None):
    return 'private full-log metadata retained outside canonical frame\n' + BEGIN + fixture(record) + END


class PolicyProvenanceRunner(unittest.TestCase):
    def setUp(self):
        self.m = module('policy-provenance-runner')
        self.q = module('local-qualification')

    def test_fixed_suite_cpu_devices_budget_and_external_command_binding(self):
        self.assertEqual(self.q.cpu_count_for_suite(SUITE), 1)
        self.assertTrue(self.q.uses_device_fixture(SUITE))
        self.assertIsNone(self.q.lifecycle_budget_for_suite(SUITE))
        self.assertEqual(self.q.startup_envelope_budget_for_suite(SUITE), self.q.startup_envelope_budget_for_suite('policy-startup-envelope'))
        packages, command = self.q.suite_command(SUITE, artifact_digest=ARTIFACT, candidate=build())
        self.assertEqual(packages, '')
        for value in (SOURCE, COMPILER, ARTIFACT, 'HORVIUM_POLICY_STARTUP_PROVENANCE_VM=1', 'HORVIUM_CLOSED_STARTUP_OWNER_VM=1', 'HORVIUM_POLICY_PROVENANCE_SOURCE_ISOLATED=1'):
            self.assertIn(value, command)
        for args in [dict(artifact_digest=ARTIFACT), dict(source_build=True, candidate=build()), dict(artifact_digest='d'*64, candidate=build())]:
            with self.subTest(args=args), self.assertRaises(ValueError):
                self.q.suite_command(SUITE, **args)

    def test_exact_framed_parent_retains_unknown_lines_for_rejection(self):
        result = self.m.validate(log(), build())
        self.assertTrue(result['diagnostic_complete'])
        self.assertFalse(result['native_acceptance'])
        for bad in ['', log()*2, log().replace(END,''), log().replace(BEGIN,''), log().replace(END,'raw parent inventory\n'+END), log()+'POLICY_STARTUP_PROVENANCE_TRACE {}\n', log().replace(END, '=== RUN   TestUnknown\n'+END), log().replace('--- PASS:', '--- FAIL:')]:
            with self.subTest(case=bad[:40]), self.assertRaises(ValueError):
                self.m.validate(bad, build())

    def test_incomplete_trace_retained_and_cleanup_is_independent(self):
        for status in ('unattributed','ambiguous','invalid','overflow'):
            with self.subTest(status=status), self.assertRaises(ValueError) as error:
                self.m.validate(log(dict(status=status,frames=[])), build())
            self.assertEqual(error.exception.evidence['trace']['status'],status)
            self.assertFalse(error.exception.evidence['native_acceptance'])
        for bad in [log(), log()+'DEVICE_FIXTURE_CLEANUP fail\n', log()+'DEVICE_FIXTURE_CLEANUP pass\n'*2, 'DEVICE_FIXTURE_CLEANUP pass\n'+log()]:
            with self.assertRaises(ValueError):self.m.validate(bad, build(), complete=True)
        self.m.validate(log()+'DEVICE_FIXTURE_CLEANUP pass\n', build(), complete=True)
        self.q.validate_result(SUITE, log()+'DEVICE_FIXTURE_CLEANUP pass\n', complete=True, candidate=build())

    def test_build_isolation_profile_and_source_bindings_are_required(self):
        self.m.check_build(build())
        mutations = [lambda b:b.pop('source_isolation'), lambda b:b.update(target='linux/amd64'),
                     lambda b:b.update(cgo_enabled=True), lambda b:b.update(source_commit='private'),
                     lambda b:b['source_isolation'].update(schema=True), lambda b:b['source_isolation'].update(normal_binary_sha256='invalid'),
                     lambda b:b['source_isolation']['normal']['storage']['go_files'].append('deadline_return_fixture.go'),
                     lambda b:b['source_isolation']['normal']['host']['test_files'].append('execution_policy_startup_provenance_fixture_test.go'),
                     lambda b:b['source_isolation']['tagged']['storage']['go_files'].remove('deadline_semantics_fixture.go'),
                     lambda b:b['source_isolation']['tagged']['host']['test_files'].clear(),
                     lambda b:b.update(build_arguments=['go','test','-c','-tags=changed'])]
        for mutate in mutations:
            candidate=build(); mutate(candidate)
            with self.subTest(candidate=candidate), self.assertRaises(ValueError):self.m.check_build(candidate)
        for key,value in [('source_commit','d'*40),('compiler_sha256','d'*64),('sha256','d'*64)]:
            candidate=build();candidate[key]=value
            with self.assertRaises(ValueError):self.m.validate(log(),candidate)
        candidate=build();candidate['validated_transcript_sha256']='d'*64
        with self.assertRaisesRegex(ValueError,'evidence changed'):self.m.validate(log()+'DEVICE_FIXTURE_CLEANUP pass\n',candidate,complete=True)

    def test_actual_file_binding_retains_complete_canonical_frame(self):
        with tempfile.TemporaryDirectory() as name:
            directory=Path(name)
            expected=dict(diagnostic_complete=True,native_acceptance=False)
            with patch.object(self.m.candidate,'verify_candidate',return_value=expected) as verify:
                self.assertEqual(self.m.verify_run(directory,build(),log()),expected)
            path=directory/'provenance-transcript.txt'
            self.assertEqual(path.read_text(),fixture())
            self.assertEqual(path.stat().st_mode & 0o777,0o600)
            args,kwargs=verify.call_args
            self.assertEqual(args[1],path);self.assertEqual(args[3],directory/BINARY)
            self.assertEqual(kwargs,dict(source_commit=SOURCE,compiler_sha256=COMPILER,artifact_sha256=ARTIFACT))
            path.unlink()
            with patch.object(self.m.candidate,'verify_candidate',side_effect=ValueError('unknown canonical line')) as verify:
                with self.assertRaises(ValueError):self.m.verify_run(directory,build(),log().replace(END,'raw inventory\n'+END))
                # Framing preserves the raw line; the strict actual-file consumer
                # must reject it. Simulate that final boundary rather than filter it.
                self.assertTrue(verify.called)
                self.assertIn('raw inventory\n',path.read_text())
            path.unlink()
            other=directory/'unrelated';other.write_text('preserve original')
            path.symlink_to(other)
            with patch.object(self.m.candidate,'verify_candidate') as verify:
                with self.assertRaises(ValueError):self.m.verify_run(directory,build(),log())
                self.assertFalse(verify.called)
            self.assertEqual(other.read_text(),'preserve original')

    def test_original_profiles_cannot_satisfy_new_suite(self):
        from test_policy_startup_envelope import fixture as old
        for text in [old(), BEGIN+old()+END, log().replace('cpu_count=1','cpu_count=2')]:
            with self.assertRaises(ValueError):self.q.validate_result(SUITE,text,candidate=build())

    def test_both_new_artifacts_are_removed_without_unrelated_files(self):
        owner=module('owned-process')
        with tempfile.TemporaryDirectory() as name:
            root=Path(name).resolve();lab=root/'.tools/lab-runs/provenance';lab.mkdir(parents=True,mode=0o700)
            for filename in (BINARY,NORMAL): (lab/filename).write_text('synthetic')
            unrelated=lab/'operator-file'; unrelated.write_text('preserve')
            self.q.cleanup_prebuilt(lab)
            self.assertTrue(unrelated.exists())
            self.assertFalse((lab/BINARY).exists()); self.assertFalse((lab/NORMAL).exists())
            for filename in (BINARY,NORMAL): (lab/filename).write_text('synthetic')
            (lab/'result.json').write_text(json.dumps({'qualified':True,'failure':'retained historical failure'}))
            with patch.object(owner,'ROOT',root): owner.native_cleanup(lab,interrupted=True)
            result=json.loads((lab/'result.json').read_text())
            self.assertFalse(result['qualified']); self.assertTrue(result['supervisor_fixture_cleanup'])
            self.assertFalse((lab/BINARY).exists());self.assertFalse((lab/NORMAL).exists());self.assertEqual(unrelated.read_text(),'preserve')

    def test_release_and_source_build_are_refused_before_execution(self):
        import subprocess,sys
        tool=Path(self.q.__file__)
        for args in [(SUITE,'14.5'),(SUITE,'15.1','--source-build')]:
            result=subprocess.run([sys.executable,str(tool),*args],capture_output=True,text=True,timeout=10)
            self.assertNotEqual(result.returncode,0)
        result=subprocess.run([sys.executable,str(tool),SUITE,'15.1'],capture_output=True,text=True,timeout=10)
        self.assertEqual(result.returncode,0)
        self.assertIn('1 CPUs',result.stdout)

    def test_wrapper_uses_exact_parent_and_preserves_profile(self):
        script=Path(self.q.__file__).with_name('bsd-policy-startup-provenance-prebuilt.sh').read_text()
        for value in ('-test.timeout=18m','^TestNativeMultiplePolicyStartupProvenanceUFS$', 'POLICY_STARTUP_PROVENANCE_TRANSCRIPT_BEGIN','POLICY_STARTUP_PROVENANCE_TRANSCRIPT_END','${HORVIUM_POLICY_PROVENANCE_SOURCE_ISOLATED:-}','${HORVIUM_SOURCE_COMMIT:-}','${HORVIUM_COMPILER_SHA256:-}'):
            self.assertIn(value,script)
        self.assertNotIn('-test.timeout=44m',script)
