# SPDX-License-Identifier: BSD-3-Clause
"""Disposable portable build inputs; synthetic executables are never executed."""
import copy
import hashlib
import importlib.util
import io
import json
import subprocess
import tempfile
import tarfile
import unittest
from pathlib import Path
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
SOURCE, BASE = '1' * 40, '2' * 40
TITLE = 'test(dashboards): bind reviewed candidate artifacts'


def digest(data):
    return hashlib.sha256(data).hexdigest()


class DashboardCandidateBuildTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        spec = importlib.util.spec_from_file_location('dashboard_candidate', ROOT / 'tools/dashboard-candidate-build.py')
        cls.m = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(cls.m)

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name).resolve()
        self.review = self.root / '.tools/review-runs/original'
        self.review.mkdir(parents=True, mode=0o700)
        self.output = self.root / '.tools/dashboard-candidates/one'
        self.compiler = self.root / '.tools/go/bin/go'
        self.compiler.parent.mkdir(parents=True)
        self.compiler.write_bytes(b'synthetic compiler, never executed')
        self.compiler.chmod(0o700)
        self.identity = dict(source_commit=SOURCE, base_commit=BASE, source_tree='a' * 40,
                             title=TITLE, signed_commits=[SOURCE])
        self.key = dict(mode='full', packages=[], inputs_digest='b' * 64, environment_digest='c' * 64)
        self.supervisor = dict(completed=True, group_cleaned=True, worker_returncode=0)
        self.result = dict(self.identity, schema=2, qualified=True,
                           command=['make', 'check', 'lint', 'test', 'secrets'],
                           plan=dict(mode='full', packages=[], reason='shared tooling, policy, module or packaging change', native_review_required=True), changed_paths=['tools/tool.py'], verification_inputs={'tools/tool.py': 'blob'},
                           deep_key=self.key, full_requested=False, supervisors=[self.supervisor],
                           deep_evidence=dict(executed=True, supervisor=self.supervisor,
                                              log_sha256=digest(b'deep evidence\n')),
                           checks_log_sha256=digest(b'actual checks\n'))
        (self.review / 'checks.log').write_bytes(b'actual checks\n')
        (self.review / 'deep.log').write_bytes(b'deep evidence\n')
        self.save_review()
        archive=io.BytesIO()
        with tarfile.open(fileobj=archive,mode='w') as tar:
            info=tarfile.TarInfo('go.mod');data=b'module example.test/fixture\n';info.size=len(data);tar.addfile(info,io.BytesIO(data))
        self.archive=archive.getvalue()

    def save_review(self):
        (self.review / 'result.json').write_text(json.dumps(self.result))

    def review_patches(self):
        return [patch.object(self.m, 'source_identity', return_value=self.identity),
                patch.object(self.m.review.PLAN, 'input_fingerprint', return_value=('b' * 64, {'tools/tool.py': 'blob'})),
                patch.object(self.m.review.PLAN, 'environment_fingerprint', return_value='c' * 64),
                patch.object(self.m.review.PLAN, 'changed_paths', return_value=['tools/tool.py'])]

    def verify(self):
        patches = self.review_patches()
        for p in patches:p.start();self.addCleanup(p.stop)
        return self.m.verify_review(self.root, self.review, source_commit=SOURCE, base_commit=BASE, pr_title=TITLE)

    def test_review_requires_matching_actual_source_logs_inputs_and_cleanup(self):
        accepted = self.verify()
        self.assertEqual(accepted['review_sha256'], digest((self.review / 'result.json').read_bytes()))
        original = copy.deepcopy(self.result)
        for mutate in [lambda r:r.update(qualified=False), lambda r:r.update(schema=True),
                       lambda r:r.update(source_commit='3' * 40), lambda r:r.update(base_commit='3' * 40),
                       lambda r:r.update(title='test: other title'), lambda r:r.update(checks_log_sha256='0' * 64),
                       lambda r:r.update(command=['true']), lambda r:r.update(verification_inputs={}),
                       lambda r:r['deep_key'].update(environment_digest='0' * 64),
                       lambda r:r['supervisors'][0].update(group_cleaned=False),
                       lambda r:r['supervisors'][0].update(worker_returncode=False),
                       lambda r:r['deep_evidence'].update(log_sha256='0' * 64)]:
            self.result=copy.deepcopy(original);mutate(self.result);self.save_review()
            with self.subTest(result=self.result), self.assertRaises(ValueError):self.verify()
        self.result=original;self.save_review()
        (self.review / 'checks.log').write_bytes(b'altered log')
        with self.assertRaises(ValueError):self.verify()

    def test_original_deep_reuse_requires_safe_ancestor_original_report_and_log(self):
        self.verify()
        original_hash=digest((self.review / 'result.json').read_bytes())
        final=self.root / '.tools/review-runs/followup';final.mkdir(mode=0o700)
        (final / 'checks.log').write_bytes(b'actual checks\n')
        original=copy.deepcopy(self.result)
        reuse=dict(record='.tools/review-runs/original/result.json',record_sha256=original_hash,
                   source_commit=SOURCE,log_sha256=original['deep_evidence']['log_sha256'],result={})
        followup=dict(original,deep_evidence=dict(executed=False,reused=reuse))
        (final / 'result.json').write_text(json.dumps(followup))
        self.result['deep_evidence']['result']={};self.save_review()
        reuse['record_sha256']=digest((self.review / 'result.json').read_bytes())
        (final / 'result.json').write_text(json.dumps(followup))
        with patch.object(self.m, 'ancestor', return_value=True):
            self.m.verify_review(self.root,final,source_commit=SOURCE,base_commit=BASE,pr_title=TITLE)
        for key,value in [('record','../../external/result.json'),('record_sha256','0'*64),('source_commit','3'*40),('log_sha256','0'*64)]:
            changed=copy.deepcopy(followup);changed['deep_evidence']['reused'][key]=value
            (final / 'result.json').write_text(json.dumps(changed))
            with self.subTest(key=key),patch.object(self.m,'ancestor',return_value=True),self.assertRaises(ValueError):
                self.m.verify_review(self.root,final,source_commit=SOURCE,base_commit=BASE,pr_title=TITLE)
        (final / 'result.json').write_text(json.dumps(followup))
        with patch.object(self.m,'ancestor',return_value=False),self.assertRaises(ValueError):
            self.m.verify_review(self.root,final,source_commit=SOURCE,base_commit=BASE,pr_title=TITLE)

    def test_review_rejects_link_duplicate_fields_and_changed_record(self):
        self.verify()
        payload=(self.review/'result.json').read_bytes();(self.review/'result.json').unlink()
        elsewhere=self.root/'outside.json';elsewhere.write_bytes(payload)
        (self.review/'result.json').symlink_to(elsewhere)
        with self.assertRaises(ValueError):self.verify()
        (self.review/'result.json').unlink();(self.review/'result.json').write_text('{"schema":2,"schema":2}')
        with self.assertRaises(ValueError):self.verify()

    def test_compiler_requires_actual_pin_marker_version_and_stable_file(self):
        (self.root/'tools').mkdir()
        pin=dict(version='go1.27.1',artifacts={'darwin_arm64':dict(sha256='d'*64)})
        (self.root/'tools/go-pins.json').write_text(json.dumps(pin))
        marker=dict(artifact='d'*64,binary=digest(self.compiler.read_bytes()))
        (self.root/'.tools/go.sha256.json').write_text(json.dumps(marker))
        with patch.object(self.m.platform,'system',return_value='Darwin'),patch.object(self.m.platform,'machine',return_value='arm64'),patch.object(self.m,'compiler_version',return_value='go1.27.1'):
            result=self.m.verify_compiler(self.root)
            self.assertEqual(result['compiler_sha256'],marker['binary'])
            for key in ['artifact','binary']:
                changed=dict(marker);changed[key]='0'*64
                (self.root/'.tools/go.sha256.json').write_text(json.dumps(changed))
                with self.subTest(key=key),self.assertRaises(ValueError):self.m.verify_compiler(self.root)
            (self.root/'.tools/go.sha256.json').write_text(json.dumps(marker))
            with patch.object(self.m,'compiler_version',return_value='go1.26.0'),self.assertRaises(ValueError):self.m.verify_compiler(self.root)
            self.compiler.chmod(0o600)
            with self.assertRaises(ValueError):self.m.verify_compiler(self.root)

    def fake_build(self, command, log, root, env):
        self.commands.append(command)
        self.assertEqual(env['GOOS'],'freebsd');self.assertEqual(env['GOARCH'],'amd64')
        self.assertEqual(env['CGO_ENABLED'],'0');self.assertEqual(env['GOAMD64'],'v1')
        self.assertEqual(env['GOTOOLCHAIN'],'local');self.assertEqual(env['GOFLAGS'],'')
        self.assertEqual(env['GOPROXY'],'off')
        Path(log).write_bytes(b'synthetic build log\n')
        if '-o' in command:
            destination=Path(command[command.index('-o')+1]);destination.write_bytes(('synthetic '+destination.name).encode());destination.chmod(0o700)
        else:
            # Archive stdout is the actual opened destination, not text re-encoded by the caller.
            Path(log).write_bytes(self.archive)
        return dict(self.supervisor,command_sha256='e'*64)

    def build(self, change=None):
        self.commands=[]
        binding={'review_sha256':'f'*64,'checks_log_sha256':'a'*64,'deep_source_commit':SOURCE}
        compiler={'go_version':'go1.27.1','compiler_sha256':digest(self.compiler.read_bytes()),'pins_sha256':'b'*64,'marker_sha256':'c'*64}
        with patch.object(self.m,'source_identity',return_value=self.identity),patch.object(self.m,'verify_review',return_value=binding),patch.object(self.m,'verify_compiler',return_value=compiler),patch.object(self.m,'run_build',side_effect=change or self.fake_build):
            return self.m.build_candidate(self.root,self.review,self.output,source_commit=SOURCE,base_commit=BASE,pr_title=TITLE)

    def test_controlled_build_hashes_actual_bytes_and_shared_release_artifacts(self):
        record=self.build();candidate=record['candidate']
        self.m.contract.candidate_identity(candidate)
        self.assertEqual(candidate['archive_sha256'],digest(self.archive))
        self.assertEqual(candidate['artifacts']['15.1'],candidate['artifacts']['14.5'])
        self.assertFalse(record['native_execution']);self.assertTrue(record['build_complete'])
        self.assertEqual([c[1:4] for c in self.commands[1:]],
                         [['build','-trimpath','-buildvcs=false'],['test','-c','-trimpath'],['test','-c','-trimpath']])
        self.assertTrue(all('-buildvcs=false' in c for c in self.commands[1:]))
        self.assertTrue(record['isolated_source_removed'])
        self.assertEqual(list(self.output.glob('source-*')),[])
        self.assertEqual(len(record['builds']),3)
        for item in record['builds']:
            self.assertEqual(item['sha256'],digest((self.output/item['filename']).read_bytes()))
        with self.assertRaises(ValueError):self.build()
        self.m.remove_artifacts(self.root,self.output)
        self.assertTrue((self.output/'result.json').exists());self.assertTrue((self.output/'management.log').exists())
        self.assertFalse((self.output/'source.tar').exists())
        self.assertTrue(json.loads((self.output/'result.json').read_text())['artifacts_removed'])

    def test_failure_retains_record_and_logs_but_removes_owned_executables(self):
        def fail(command,log,root,env):
            result=self.fake_build(command,log,root,env)
            if command[-1]=='./internal/host/freebsd':raise ValueError('private raw input')
            return result
        with self.assertRaises(ValueError) as error:self.build(fail)
        self.assertNotIn('private raw input',str(error.exception))
        record=json.loads((self.output/'result.json').read_text())
        self.assertFalse(record['build_complete']);self.assertNotIn('candidate',record)
        self.assertTrue(record['artifacts_removed']);self.assertEqual(record['failure'],'ValueError')
        self.assertTrue((self.output/'helper.log').exists())
        for name in self.m.ARTIFACT_FILES.values():self.assertFalse((self.output/name).exists())

    def test_post_build_compiler_or_artifact_mutation_and_missing_cleanup_reject(self):
        def mutate(command,log,root,env):
            result=self.fake_build(command,log,root,env)
            if command[-1]=='./cmd/horvium' and command[1]=='test':self.compiler.write_bytes(b'changed compiler')
            return result
        with self.assertRaises(ValueError):self.build(mutate)
        record=json.loads((self.output/'result.json').read_text());self.assertFalse(record['build_complete'])
        self.assertTrue(record['artifacts_removed'])

    def test_build_step_requires_owned_successful_stopped_group(self):
        for index,state in enumerate([self.supervisor,dict(self.supervisor,group_cleaned=False),dict(self.supervisor,worker_returncode=1)]):
            with self.subTest(state=state),patch.object(self.m.review.OWNED,'OwnedProcess') as cls:
                process=cls.return_value;process.wait.return_value=0;process.record.return_value=state
                if state==self.supervisor:self.m.run_build(['go','build'],self.root/('log-'+str(index)),self.root,{})
                else:
                    with self.assertRaises(ValueError):self.m.run_build(['go','build'],self.root/('log-'+str(index)),self.root,{})
                process.stop.assert_called_once()
                self.assertEqual(cls.call_args.kwargs['runs'], self.m.review.OWNED.RUNS)

    def test_source_requires_original_clean_identity_and_every_contributor_dco(self):
        with patch.object(self.m.files,'verify_source') as source,patch.object(self.m.review,'identity',return_value=self.identity),patch.object(self.m.subprocess,'run') as run:
            self.assertEqual(self.m.source_identity(self.root,SOURCE,BASE,TITLE),self.identity)
            source.assert_called_once_with(self.root,SOURCE)
            self.assertEqual(run.call_args.args[0][-2:],[BASE,SOURCE])
            run.side_effect=subprocess.CalledProcessError(1,['check-dco'])
            with self.assertRaises(ValueError):self.m.source_identity(self.root,SOURCE,BASE,TITLE)
        with patch.object(self.m.files,'verify_source',side_effect=ValueError('dirty source')),patch.object(self.m.review,'identity') as identity:
            with self.assertRaises(ValueError):self.m.source_identity(self.root,SOURCE,BASE,TITLE)
            identity.assert_not_called()

    def test_archive_rejects_changed_bytes_paths_links_and_duplicate_members(self):
        target=self.root/'extracted';target.mkdir()
        archive=self.root/'source.tar'
        for mode in ['changed','traversal','absolute','symlink','duplicate']:
            stream=io.BytesIO()
            with tarfile.open(fileobj=stream,mode='w') as tar:
                info=tarfile.TarInfo('../escape' if mode=='traversal' else '/escape' if mode=='absolute' else 'file')
                if mode=='symlink':info.type=tarfile.SYMTYPE;info.linkname='../outside'
                tar.addfile(info)
                if mode=='duplicate':tar.addfile(info)
            archive.write_bytes(stream.getvalue())
            with self.subTest(mode=mode),self.assertRaises(ValueError):
                self.m.extract_source(archive,target,'0'*64 if mode=='changed' else digest(stream.getvalue()))
            self.assertEqual(list(target.iterdir()),[])

    def test_source_review_and_earlier_artifact_mutations_abort_publication(self):
        for mode in ['source','review','artifact']:
            self.output=self.root/('.tools/dashboard-candidates/'+mode)
            binding={'review_sha256':'f'*64,'checks_log_sha256':'a'*64,'deep_source_commit':SOURCE}
            compiler={'go_version':'go1.27.1','compiler_sha256':digest(self.compiler.read_bytes())}
            self.commands=[]
            def change(command,log,root,env):
                state=self.fake_build(command,log,root,env)
                if command[-1]=='./cmd/horvium' and command[1]=='test' and mode=='artifact':
                    (self.output/'horvium').write_bytes(b'replaced earlier artifact')
                return state
            with self.subTest(mode=mode),patch.object(self.m,'source_identity',side_effect=[self.identity,ValueError('source changed')] if mode=='source' else None,return_value=self.identity),patch.object(self.m,'verify_review',side_effect=[binding,ValueError('review changed')] if mode=='review' else None,return_value=binding),patch.object(self.m,'verify_compiler',return_value=compiler),patch.object(self.m,'run_build',side_effect=change),self.assertRaises(ValueError):
                self.m.build_candidate(self.root,self.review,self.output,source_commit=SOURCE,base_commit=BASE,pr_title=TITLE)
            record=json.loads((self.output/'result.json').read_text())
            self.assertFalse(record['build_complete']);self.assertNotIn('candidate',record)
            self.assertTrue(record['artifacts_removed']);self.assertEqual(list(self.output.glob('source-*')),[])


if __name__ == '__main__':
    unittest.main()
