# SPDX-License-Identifier: BSD-3-Clause
"""Synthetic guest/supervisor records; no SSH, native executable or VM runs."""
import copy
import hashlib
import importlib.util
import io
import tarfile
import json
import tempfile
import unittest
from contextlib import ExitStack
from pathlib import Path
from unittest.mock import patch

ROOT=Path(__file__).resolve().parents[2]
SOURCE,BASE='1'*40,'2'*40
TITLE='test(dashboards): bind original guest preflight receipts'
def digest(data):return hashlib.sha256(data).hexdigest()

class DashboardHostPreflightTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):
  spec=importlib.util.spec_from_file_location('dashboard_host_preflight',ROOT/'tools/dashboard-host-preflight.py')
  cls.m=importlib.util.module_from_spec(spec);spec.loader.exec_module(cls.m)
 def setUp(self):
  self.temp=tempfile.TemporaryDirectory();self.addCleanup(self.temp.cleanup);self.root=Path(self.temp.name).resolve()
  self.output=self.root/'.tools/dashboard-candidates/one';self.review=self.root/'.tools/review-runs/one';self.lab=self.root/'.tools/lab-runs/one';self.supervisor=self.root/'.tools/process-runs/one'
  for p in [self.output,self.review,self.lab,self.supervisor]:p.mkdir(parents=True,mode=0o700)
  compiler={'go_version':'go1.27.1','compiler_sha256':'a'*64,'pins_sha256':'b'*64,'marker_sha256':'c'*64}
  reviewed={'review_sha256':'d'*64,'checks_log_sha256':'e'*64,'deep_record_sha256':'d'*64,'deep_log_sha256':'f'*64,'deep_source_commit':SOURCE}
  self.compiler,self.reviewed=compiler,reviewed
  self.builds=[];hashes={}
  for role,name in self.m.build.ARTIFACT_FILES.items():
   p=self.output/name;p.write_bytes(('synthetic '+role).encode());p.chmod(0o700);hashes[role]=digest(p.read_bytes())
   log=self.output/(role+'.log');log.write_bytes(b'Owned candidate command output follows:\n')
   arguments=['go','build','-trimpath','-buildvcs=false'] if role=='management' else ['go','test','-c','-trimpath','-buildvcs=false']
   package='./internal/host/freebsd' if role=='helper' else './cmd/horvium'
   self.builds.append({'role':role,'filename':name,'sha256':hashes[role],'log_sha256':digest(log.read_bytes()),'target':'freebsd/amd64','goamd64':'v1','cgo_enabled':False,'arguments':arguments+['-o','<owned-artifact>',package]})
  archive=self.output/'source.tar';self.original_blob=b'module example.test/synthetic\n';archive.write_bytes(self.archive(self.original_blob))
  self.candidate={'source_commit':SOURCE,'base_commit':BASE,'pr_title':TITLE,'go_version':'go1.27.1','compiler_sha256':'a'*64,'archive_sha256':digest(archive.read_bytes()),'review_verified':True,'artifacts':{r:dict(hashes) for r in ['15.1','14.5']}}
  modules=[]
  for name in ['modules-before.log','modules-after.log']:
   p=self.output/name;p.write_bytes(b'Owned candidate command output follows:\nall modules verified\n');modules.append(digest(p.read_bytes()))
  stopped={'completed':True,'group_cleaned':True,'worker_returncode':0,'completed_at':10.0,'started_at':9.0,'group':123,'command_sha256':'a'*64,'reason':'exit','escalated':False}
  self.record={'schema':1,'source_commit':SOURCE,'base_commit':BASE,'pr_title':TITLE,'build_complete':True,'native_execution':False,'artifacts_removed':False,'builds':self.builds,'supervisors':[dict(stopped) for _ in range(6)],'original_archive_build':True,'modules_verified_before_after':True,'module_log_sha256':modules,'isolated_source_removed':True,'candidate':self.candidate,'compiler':compiler,'review':reviewed}
  self.observed={'artifacts':hashes,'go_version':'go1.27.1','kernel':'15.1-RELEASE-p1','userland':'15.1-RELEASE','zfs':'zfs-actual zfs-kmod-actual','physical_bytes':6442450944}
  command=self.m.preflight_command(self.root,self.lab,2222)
  self.state={'completed':True,'group_cleaned':True,'worker_returncode':0,'command_sha256':digest(json.dumps(command).encode()),'group':123,'started_at':20.0,'completed_at':30.0,'escalated':False,'reason':'exit'}
  self.save_build();self.save_state();self.save_log()
  stack=ExitStack();self.addCleanup(stack.close)
  stack.enter_context(patch.object(self.m,'tracked_tree',create=True,return_value={'go.mod':(hashlib.sha1(b'blob '+str(len(self.original_blob)).encode()+b'\x00'+self.original_blob).hexdigest(),False)}))
  stack.enter_context(patch.object(self.m.build,'source_identity',return_value={'source_commit':SOURCE}))
  stack.enter_context(patch.object(self.m.build,'verify_compiler',return_value=compiler))
  stack.enter_context(patch.object(self.m.build,'verify_review',return_value=reviewed))
 def archive(self,data):
  output=io.BytesIO()
  with tarfile.open(fileobj=output,mode='w',format=tarfile.PAX_FORMAT,pax_headers={'comment':SOURCE}) as tar:
   info=tarfile.TarInfo('go.mod');info.size=len(data);info.mode=0o644;tar.addfile(info,io.BytesIO(data))
  return output.getvalue()
 def save_build(self): (self.output/'result.json').write_text(json.dumps(self.record))
 def save_state(self): (self.supervisor/'state.json').write_text(json.dumps(self.state))
 def payload(self):
  return ('=== RUN   TestNativeDashboardArtifactProvenance\n    dashboard_artifacts_freebsd_test.go:88: DASHBOARD_GUEST_ARTIFACTS '+json.dumps(self.observed)+'\n--- PASS: TestNativeDashboardArtifactProvenance (0.10s)\nPASS\n').encode()
 def save_log(self): (self.lab/'dashboard-preflight.log').write_bytes(self.payload())
 def verify(self):return self.m.verify_preflight(self.root,self.review,self.output,self.lab,self.supervisor,'15.1',2222,source_commit=SOURCE,base_commit=BASE,pr_title=TITLE)
 def test_complete_original_binding_is_read_only_and_not_acceptance(self):
  paths=[self.output/'result.json',self.lab/'dashboard-preflight.log',self.supervisor/'state.json'];before=[p.read_bytes() for p in paths]
  result=self.verify();self.assertEqual(result['candidate'],self.candidate);self.assertEqual(result['observed'],self.observed)
  self.assertEqual(result['transcript_sha256'],digest(self.payload()));self.assertFalse(result['native_acceptance']);self.assertFalse(result['resource_execution'])
  self.assertEqual(before,[p.read_bytes() for p in paths])
 def test_candidate_build_records_files_and_context_are_independent(self):
  original=copy.deepcopy(self.record)
  for mutate in [lambda r:r.update(schema=True),lambda r:r.update(build_complete=False),lambda r:r.update(artifacts_removed=True),lambda r:r.update(native_execution=True),lambda r:r['candidate'].update(source_commit='3'*40),lambda r:r['compiler'].update(compiler_sha256='4'*64),lambda r:r['review'].update(review_sha256='4'*64),lambda r:r['builds'][0].update(cgo_enabled=True),lambda r:r['builds'][0].update(arguments=['go','build','wrong']),lambda r:r['supervisors'][0].update(worker_returncode=False),lambda r:r.update(modules_verified_before_after=False),lambda r:r.update(supervisors=r['supervisors'][:-1]),lambda r:r['candidate']['artifacts']['14.5'].update(management='4'*64)]:
   self.record=copy.deepcopy(original);mutate(self.record);self.save_build()
   with self.subTest(record=self.record),self.assertRaises(ValueError):self.verify()
  self.record=original;self.save_build()
  for p in [self.output/'horvium',self.output/'source.tar',self.output/'helper.log',self.output/'modules-before.log']:
   original=p.read_bytes();p.write_bytes(b'altered actual bytes')
   with self.subTest(path=p.name),self.assertRaises(ValueError):self.verify()
   p.write_bytes(original)
  for method in ['source_identity','verify_compiler','verify_review']:
   with patch.object(self.m.build,method,side_effect=ValueError('independent check failed')),self.assertRaises(ValueError):self.verify()
 def test_guest_facts_never_replace_expected_candidate_or_release(self):
  original=copy.deepcopy(self.observed)
  for mutate in [lambda o:o['artifacts'].update(management='4'*64),lambda o:o.update(go_version='go1.27.2'),lambda o:o.update(kernel='14.5-RELEASE'),lambda o:o.update(userland='15.1-STABLE'),lambda o:o.update(physical_bytes=True),lambda o:o.update(physical_bytes=0),lambda o:o.update(zfs=''),lambda o:o.update(zfs='zfs\x7f'),lambda o:o.update(candidate=self.candidate),lambda o:o.update(qualified=True)]:
   self.observed=copy.deepcopy(original);mutate(self.observed);self.save_log()
   with self.subTest(observed=self.observed),self.assertRaises(ValueError):self.verify()
  self.observed=original;self.observed['kernel']=self.observed['userland']='14.5-RELEASE-p1';self.save_log()
  result=self.m.verify_preflight(self.root,self.review,self.output,self.lab,self.supervisor,'14.5',2222,source_commit=SOURCE,base_commit=BASE,pr_title=TITLE);self.assertEqual(result['release'],'14.5')
 def test_incomplete_duplicate_failed_unknown_or_oversized_log_denied(self):
  original=self.payload()
  for payload in [b'',original.replace(b'--- PASS:',b'--- SKIP:'),original.replace(b'--- PASS:',b'--- FAIL:'),original.replace(b'PASS\n',b'',1),original+original,original+b'DASHBOARD_RESOURCE {}\n',original.replace(b'"go_version":',b'"go_version":"go1.27.1","go_version":'),original.replace(b'6442450944',b'NaN'),original[:-5],b'x'*65537,original.replace(b'0.10s',b'nans')]:
   (self.lab/'dashboard-preflight.log').write_bytes(payload)
   with self.subTest(payload=payload[:60]),self.assertRaises(ValueError):self.verify()
 def test_actual_supervisor_command_completion_and_order_required(self):
  original=copy.deepcopy(self.state)
  for mutate in [lambda s:s.update(completed=False),lambda s:s.update(group_cleaned=False),lambda s:s.update(worker_returncode=False),lambda s:s.update(reason='timeout'),lambda s:s.update(command_sha256='4'*64),lambda s:s.update(started_at=5.0),lambda s:s.update(started_at=True),lambda s:s.update(completed_at=19.0),lambda s:s.update(escalated=True),lambda s:s.update(group=0),lambda s:s.update(cleanup_failure='unverified')]:
   self.state=copy.deepcopy(original);mutate(self.state);self.save_state()
   with self.subTest(state=self.state),self.assertRaises(ValueError):self.verify()
  self.state=original;self.save_state()
  for port in [True,0,65536,'2222']:
   with self.subTest(port=port),self.assertRaises(ValueError):self.m.preflight_command(self.root,self.lab,port)
 def test_linked_outside_and_changed_original_inputs_denied(self):
  for p in [self.output/'result.json',self.lab/'dashboard-preflight.log',self.supervisor/'state.json']:
   data=p.read_bytes();p.unlink();target=self.root/'elsewhere';target.write_bytes(data);p.symlink_to(target)
   with self.subTest(path=p.name),self.assertRaises(ValueError):self.verify()
   p.unlink();p.write_bytes(data)
  with self.assertRaises(ValueError):self.m.verify_preflight(self.root,self.review,self.output,self.lab,self.root,'15.1',2222,source_commit=SOURCE,base_commit=BASE,pr_title=TITLE)
  original=self.m.verify_build;calls=0
  def changing(*args,**kwargs):
   nonlocal calls
   result=original(*args,**kwargs);calls+=1
   if calls==1:(self.lab/'dashboard-preflight.log').write_bytes(self.payload()+b'changed\n')
   return result
  with patch.object(self.m,'verify_build',side_effect=changing),self.assertRaises(ValueError):self.verify()

 def test_original_archive_content_must_match_signed_tree(self):
  original=self.m.verify_build(self.root,self.review,self.output,source_commit=SOURCE,base_commit=BASE,pr_title=TITLE)
  self.assertEqual(original['candidate'],self.candidate)
  path=self.output/'source.tar';path.write_bytes(self.archive(b'module example.test/forged\n'))
  self.record['candidate']['archive_sha256']=digest(path.read_bytes());self.save_build()
  with self.assertRaises(ValueError):self.verify()

 def test_archive_missing_duplicate_mode_link_header_and_unexpected_entries_denied(self):
  for kind in ['missing','duplicate','mode','link','header','unexpected']:
   output=io.BytesIO()
   with tarfile.open(fileobj=output,mode='w',format=tarfile.PAX_FORMAT,pax_headers={'comment':SOURCE if kind!='header' else '3'*40}) as tar:
    if kind!='missing':
     info=tarfile.TarInfo('go.mod' if kind!='unexpected' else '../go.mod');info.mode=0o755 if kind=='mode' else 0o644;info.size=len(self.original_blob)
     if kind=='link':info.type=tarfile.SYMTYPE;info.linkname='outside';info.size=0
     tar.addfile(info,io.BytesIO(self.original_blob))
     if kind=='duplicate':tar.addfile(info,io.BytesIO(self.original_blob))
   path=self.output/'source.tar';path.write_bytes(output.getvalue());self.record['candidate']['archive_sha256']=digest(path.read_bytes());self.save_build()
   with self.subTest(kind=kind),self.assertRaises(ValueError):self.verify()
 def test_earlier_build_files_and_supervisor_mutation_during_rechecks_denied(self):
  for target in ['artifact','state']:
   original_compiler=self.m.build.verify_compiler;calls=0
   def changing(*args,**kwargs):
    nonlocal calls
    result=original_compiler(*args,**kwargs);calls+=1
    if calls==2:
     if target=='artifact':(self.output/'horvium').write_bytes(b'changed after first opened hash')
     else:self.state['reason']='timeout';self.save_state()
    return result
   with self.subTest(target=target),patch.object(self.m.build,'verify_compiler',side_effect=changing),self.assertRaises(ValueError):self.verify()
   (self.output/'horvium').write_bytes(b'synthetic management');self.state['reason']='exit';self.save_state()

 def test_original_build_supervisor_cannot_hide_timeout_or_cleanup_failure(self):
  original=copy.deepcopy(self.record)
  for mutate in [lambda s:s.update(reason='timeout'),lambda s:s.update(escalated=True),lambda s:s.update(cleanup_failure='unverified'),lambda s:s.update(group=True),lambda s:s.update(started_at=11.0),lambda s:s.update(command_sha256='malformed')]:
   self.record=copy.deepcopy(original);mutate(self.record['supervisors'][0]);self.save_build()
   with self.subTest(supervisor=self.record['supervisors'][0]),self.assertRaises(ValueError):self.verify()
