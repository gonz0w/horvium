# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]

class PersistentBackingEvidenceTests(unittest.TestCase):
    def setUp(self):
        spec = importlib.util.spec_from_file_location('persistent_evidence', ROOT/'tools/check-persistent-backing-evidence.py')
        self.module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.module)
        self.log = ''.join('--- PASS: TestNativePersistentBacking'+kind+' (0.1s)\nPERSISTENT_BACKING_'+kind+'_RESULT '+' '.join(c+'=pass' for c in sorted(cases))+'\nPERSISTENT_BACKING_TEARDOWN '+kind.lower()+' filesystem=pass device=pass directory=pass\n' for kind, cases in self.module.CASES.items())

    def test_complete_actual_native_scope_required(self):
        self.assertEqual([len(self.module.CASES[k]) for k in ('UFS','ZFS')], [16,12])
        self.module.validate(self.log)
        for bad in ['',self.log*2,self.log.replace('PASS','SKIP'),self.log.replace('PASS','FAIL'),self.log.replace('PERSISTENT_BACKING_TEARDOWN','MISSING_TEARDOWN')]:
            with self.assertRaises(ValueError): self.module.validate(bad)
        for kind, cases in self.module.CASES.items():
            marker = 'PERSISTENT_BACKING_'+kind+'_RESULT '
            original = marker+' '.join(c+'=pass' for c in sorted(cases))
            for case in cases:
                for value in ['',case+'=fail',case+'=pass '+case+'=pass']:
                    changed = marker+' '.join(value if c==case else c+'=pass' for c in sorted(cases))
                    with self.subTest(kind=kind,case=case,value=value), self.assertRaises(ValueError):
                        self.module.validate(self.log.replace(original,changed))

    def test_runner_uses_pinned_artifact_and_focused_inventory(self):
        spec = importlib.util.spec_from_file_location('local_qualification', ROOT/'tools/local-qualification.py')
        module = importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
        packages, command = module.suite_command('persistent-backing',False,'a'*64)
        self.assertEqual(packages,'')
        self.assertIn('HORVIUM_BACKING_VM=1',command)
        self.assertIn('bsd-persistent-backing-prebuilt.sh',command)
        self.assertFalse(module.uses_device_fixture('persistent-backing'))
        module.validate_result('persistent-backing',self.log)
        with self.assertRaises(ValueError):module.validate_result('persistent-backing',self.log.replace('native_id=pass',''))
        module.validate_prebuilt('PERSISTENT_BACKING_PREBUILT_SHA256 '+'a'*64+'\n','a'*64,'persistent-backing')
        with self.assertRaises(ValueError):module.validate_prebuilt('', 'a'*64,'persistent-backing')
        script=(ROOT/'tools/bsd-persistent-backing-prebuilt.sh').read_text()
        self.assertIn("-test.run='^TestNativePersistentBacking(UFS|ZFS)$'",script)
