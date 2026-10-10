# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
from pathlib import Path
import unittest
ROOT=Path(__file__).resolve().parents[2]
def module(name):
 spec=importlib.util.spec_from_file_location(name,ROOT/'tools'/f'{name}.py');m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
class ExecutionSystemEvidenceTests(unittest.TestCase):
 def setUp(self):
  self.m=module('check-execution-system-evidence');old=module('check-original-backing-evidence');joint=module('check-joint-leases-evidence');self.log=''
  for family in ('ExecutionSystem','OriginalBacking','JointExecutionLeases'):
   for kind in ('UFS','ZFS'):
    cases=(old.CASES if family=='OriginalBacking' else joint.CASES | ({'same_dataset_denial'} if kind=='ZFS' else set())) if family!='ExecutionSystem' else self.m.CASES | (self.m.ZFS_CASES if kind=='ZFS' else set())
    n='TestNative'+family+kind;marker=('EXECUTION_SYSTEM_' if family=='ExecutionSystem' else 'ORIGINAL_BACKING_' if family=='OriginalBacking' else 'JOINT_EXECUTION_')+kind+'_RESULT '
    self.log+='=== RUN   '+n+'\n'+marker+' '.join(c+'=pass' for c in sorted(cases))+'\n'
    if family=='ExecutionSystem':self.log+='EXECUTION_SYSTEM_DOMAIN_TEARDOWN '+kind.lower()+' filesystems=pass directories=pass\nEXECUTION_SYSTEM_DATA_TEARDOWN '+kind.lower()+' restored=pass\n'
    else:self.log+='PERSISTENT_BACKING_TEARDOWN '+kind.lower()+' filesystem=pass device=pass directory=pass\n'
    self.log+='--- PASS: '+n+' (0.1s)\n'
 def test_required_system_and_retained_native_scope(self):
  self.m.validate(self.log)
  for c in self.m.CASES | self.m.ZFS_CASES:
   for replacement in ['',c+'=fail',c+'=pass '+c+'=pass']:
    with self.subTest(c=c),self.assertRaises(ValueError):self.m.validate(self.log.replace(c+'=pass',replacement))
  for bad in ['',self.log*2,self.log.replace('PASS','SKIP'),self.log.replace('PASS','FAIL'),self.log.replace('DOMAIN_TEARDOWN','missing'),self.log.replace('DATA_TEARDOWN','missing')]:
   with self.assertRaises(ValueError):self.m.validate(bad)
 def test_bound_binary_and_device_teardown_required(self):
  m=module('local-qualification');self.assertTrue(m.uses_device_fixture('execution-system'))
  m.validate_result('execution-system',self.log)
  with self.assertRaises(ValueError):m.validate_result('execution-system',self.log,complete=True)
  m.validate_result('execution-system',self.log+'DEVICE_FIXTURE_CLEANUP pass\n',complete=True)
  m.validate_prebuilt('EXECUTION_SYSTEM_PREBUILT_SHA256 '+'a'*64+'\n','a'*64,'execution-system')
  with self.assertRaises(ValueError):m.validate_prebuilt('', 'a'*64,'execution-system')
  self.assertIn('bsd-execution-system-prebuilt.sh',m.suite_command('execution-system',False,'a'*64)[1])
