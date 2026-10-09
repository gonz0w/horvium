# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
from pathlib import Path
import unittest

ROOT=Path(__file__).resolve().parents[2]
def module(name):
    spec=importlib.util.spec_from_file_location(name,ROOT/'tools'/f'{name}.py')
    m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m

class JointLeasesEvidenceTests(unittest.TestCase):
    def setUp(self):
        self.m=module('check-joint-leases-evidence');old=module('check-original-backing-evidence')
        self.log=''
        for family in ('OriginalBacking','JointExecutionLeases'):
            for kind in ('UFS','ZFS'):
                cases=old.CASES if family=='OriginalBacking' else self.m.CASES | ({'same_dataset_denial'} if kind=='ZFS' else set())
                name='TestNative'+family+kind
                marker=('ORIGINAL_BACKING_' if family=='OriginalBacking' else 'JOINT_EXECUTION_')+kind+'_RESULT '
                self.log+='=== RUN   '+name+'\n'+marker+' '.join(c+'=pass' for c in sorted(cases))+'\nPERSISTENT_BACKING_TEARDOWN '+kind.lower()+' filesystem=pass device=pass directory=pass\n--- PASS: '+name+' (0.1s)\n'

    def test_actual_joint_and_retained_original_scope_required(self):
        self.assertEqual(len(self.m.CASES),40)
        self.m.validate(self.log)
        for bad in ['',self.log*2,self.log.replace('PASS','SKIP'),self.log.replace('PASS','FAIL'),self.log.replace('PERSISTENT_BACKING_TEARDOWN','MISSING_TEARDOWN')]:
            with self.assertRaises(ValueError):self.m.validate(bad)
        for case in self.m.CASES | {'same_dataset_denial'}:
            for value in ['',case+'=fail',case+'=pass '+case+'=pass']:
                with self.subTest(case=case,value=value),self.assertRaises(ValueError):self.m.validate(self.log.replace(case+'=pass',value))

    def test_artifact_and_four_parent_selection_are_pinned(self):
        m=module('local-qualification')
        packages,command=m.suite_command('joint-leases',False,'a'*64)
        self.assertEqual(packages,'');self.assertIn('bsd-joint-leases-prebuilt.sh',command)
        self.assertIn('HORVIUM_BACKING_VM=1',command)
        m.validate_result('joint-leases',self.log)
        with self.assertRaises(ValueError):m.validate_result('joint-leases',self.log.replace('native_pair=pass',''))
        m.validate_prebuilt('JOINT_LEASES_PREBUILT_SHA256 '+'a'*64+'\n','a'*64,'joint-leases')
        with self.assertRaises(ValueError):m.validate_prebuilt('', 'a'*64,'joint-leases')
        self.assertIn("-test.run='^TestNative(OriginalBacking|JointExecutionLeases)(UFS|ZFS)$'",(ROOT/'tools/bsd-joint-leases-prebuilt.sh').read_text())
