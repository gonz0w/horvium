# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]

class OriginalBackingEvidenceTests(unittest.TestCase):
    def setUp(self):
        spec = importlib.util.spec_from_file_location('original_evidence', ROOT/'tools/check-original-backing-evidence.py')
        self.module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.module)
        self.log = ''.join('--- PASS: TestNativeOriginalBacking'+kind+' (0.1s)\nORIGINAL_BACKING_'+kind+'_RESULT '+' '.join(c+'=pass' for c in sorted(self.module.CASES))+'\nPERSISTENT_BACKING_TEARDOWN '+kind.lower()+' filesystem=pass device=pass directory=pass\n' for kind in ('UFS','ZFS'))

    def test_complete_actual_original_lease_scope_required(self):
        self.assertEqual(len(self.module.CASES),33)
        self.module.validate(self.log)
        for bad in ['',self.log*2,self.log.replace('PASS','SKIP'),self.log.replace('PASS','FAIL'),self.log.replace('PERSISTENT_BACKING_TEARDOWN','MISSING_TEARDOWN')]:
            with self.assertRaises(ValueError):self.module.validate(bad)
        for case in self.module.CASES:
            for replacement in ['',case+'=fail',case+'=pass '+case+'=pass']:
                with self.subTest(case=case,replacement=replacement), self.assertRaises(ValueError):
                    self.module.validate(self.log.replace(case+'=pass',replacement,1))

    def test_focused_runner_and_artifact_are_bound(self):
        spec = importlib.util.spec_from_file_location('original_runner', ROOT/'tools/local-qualification.py')
        module = importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        packages,command = module.suite_command('original-backing',False,'a'*64)
        self.assertEqual(packages,'');self.assertIn('HORVIUM_BACKING_VM=1',command)
        self.assertIn('bsd-original-backing-prebuilt.sh',command)
        self.assertFalse(module.uses_device_fixture('original-backing'))
        module.validate_result('original-backing',self.log)
        with self.assertRaises(ValueError):module.validate_result('original-backing',self.log.replace('native_domain=pass',''))
        module.validate_prebuilt('ORIGINAL_BACKING_PREBUILT_SHA256 '+'a'*64+'\n','a'*64,'original-backing')
        with self.assertRaises(ValueError):module.validate_prebuilt('', 'a'*64,'original-backing')
        script=(ROOT/'tools/bsd-original-backing-prebuilt.sh').read_text()
        self.assertIn("-test.run='^TestNativeOriginalBacking(UFS|ZFS)$'",script)
