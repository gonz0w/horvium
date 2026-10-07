# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
import re
import subprocess
import unittest
from pathlib import Path
from unittest.mock import Mock

ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location('local_qualification', ROOT / 'tools/local-qualification.py')


class LocalQualificationTests(unittest.TestCase):
    def setUp(self):
        self.module = importlib.util.module_from_spec(SPEC)
        SPEC.loader.exec_module(self.module)

    def test_focused_property_suite_uses_owned_snapshot_device_fixture(self):
        for suite in ['devices','pool-properties']:
            self.assertTrue(self.module.uses_device_fixture(suite))
        for suite in ['state','properties','foundation']:
            self.assertFalse(self.module.uses_device_fixture(suite))

    def test_focused_property_completion_requires_device_teardown(self):
        checker=self.module.load_tool('check-pool-properties-evidence')
        log='--- PASS: TestNativePoolPropertyCollection (1.0s)\nPOOL_PROPERTY_COLLECTION_RESULT '+' '.join(c+'=pass' for c in sorted(checker.CASES))+'\n'
        for bad in [log,log+'DEVICE_FIXTURE_CLEANUP fail\n',log+('DEVICE_FIXTURE_CLEANUP pass\n'*2)]:
            with self.assertRaises(ValueError):self.module.validate_result('pool-properties',bad,complete=True)
        self.module.validate_result('pool-properties',log+'DEVICE_FIXTURE_CLEANUP pass\n',complete=True)

    def test_prebuilt_state_avoids_guest_packages_and_checks_artifact_hash(self):
        packages,command=self.module.suite_command('state',False,'a'*64)
        self.assertEqual(packages,'')
        self.assertIn('bsd-state-prebuilt.sh',command)
        self.assertIn('a'*64,command)
        self.assertEqual(self.module.suite_command('state',True,None),self.module.SUITES['state'])
        self.assertEqual(self.module.suite_command('properties',False,None),self.module.SUITES['properties'])
        for digest in ['','oops','a'*63,'a'*64+';bad']:
            with self.assertRaises(ValueError):self.module.suite_command('state',False,digest)

    def test_pool_properties_use_focused_checksum_bound_prebuilt(self):
        packages,command=self.module.suite_command('pool-properties',False,'a'*64)
        self.assertEqual(packages,'')
        self.assertIn('bsd-pool-properties-prebuilt.sh',command)
        self.assertIn('HORVIUM_PROPERTY_VM=1',command)
        self.assertIn('a'*64,command)
        for digest in [None,'oops','a'*63,'a'*64+';bad']:
            with self.assertRaises(ValueError):self.module.suite_command('pool-properties',False,digest)
        self.module.validate_prebuilt('POOL_PROPERTIES_PREBUILT_SHA256 '+'a'*64+'\n','a'*64,'pool-properties')
        for log in ['',('POOL_PROPERTIES_PREBUILT_SHA256 '+'a'*64+'\n')*2,'STATE_PREBUILT_SHA256 '+'a'*64+'\n']:
            with self.assertRaises(ValueError):self.module.validate_prebuilt(log,'a'*64,'pool-properties')

    def test_pool_properties_require_native_assertions_and_teardown(self):
        checker=self.module.load_tool('check-pool-properties-evidence')
        log='--- PASS: TestNativePoolPropertyCollection (1.0s)\nPOOL_PROPERTY_COLLECTION_RESULT '+' '.join(c+'=pass' for c in sorted(checker.CASES))+'\n'
        self.module.validate_result('pool-properties',log)
        for bad in ['',log.replace('teardown=pass','teardown=fail'),log.replace('eligibility_denied=pass',''),log.replace('--- PASS:','--- SKIP:')]:
            with self.assertRaises(ValueError):self.module.validate_result('pool-properties',bad)

    def test_guest_has_snapshot_storage_and_only_loopback_ssh(self):
        command = self.module.qemu_command('qemu', Path('/fixture/base'), Path('/code'), Path('/fixture/vars'), 23456)
        self.assertIn('-snapshot', command)
        self.assertIn('q35,accel=tcg', command)
        self.assertIn('qemu64', command)
        network = command[command.index('-netdev') + 1]
        self.assertIn('hostfwd=tcp:127.0.0.1:23456-:22', network)
        self.assertNotIn('0.0.0.0', network)
        self.assertNotIn('-daemonize', command)

    def test_ssh_uses_fixture_key_and_config_without_host_credentials(self):
        command = self.module.ssh_command(Path('/fixture/key'), Path('/fixture/known_hosts'), 23456)
        self.assertEqual(command[1:3], ['-F', '/dev/null'])
        self.assertIn('IdentitiesOnly=yes', command)
        self.assertIn('IdentityAgent=none', command)
        self.assertIn('UserKnownHostsFile=/fixture/known_hosts', command)
        self.assertIn('root@127.0.0.1', command)

    def test_skipped_failed_or_incomplete_guest_cannot_be_qualified(self):
        for log in ('PASS: actual independent SMB/NFS fixture reads through helper and management outages',
                    '--- SKIP: TestNativeIndependentSMBNFSContinuity (0.1s)',
                    '--- FAIL: TestNativeIndependentSMBNFSContinuity (0.1s)'):
            with self.assertRaises(ValueError):
                self.module.validate_result('continuity', log)

    def test_state_suite_requires_actual_complete_native_lock_and_cleanup_cases(self):
        checker=self.module.load_tool('check-state-evidence')
        result='STATE_DIRECTORY_RESULT '+' '.join(c+'=pass' for c in sorted(checker.CASES))+'\nOWNERSHIP_READ_RESULT '+' '.join(c+'=pass' for c in sorted(checker.READ_CASES))+'\n'
        result+='STATE_ACL_RESULT '+' '.join(c+'=pass' for c in sorted(checker.ACL_CASES))+'\n'
        result+='OWNERSHIP_PUBLISH_RESULT '+' '.join(c+'=pass' for c in sorted(checker.PUBLISH_CASES))+'\nOWNERSHIP_PUBLISH_FS_RESULT '+' '.join(c+'=pass' for c in sorted(checker.PUBLISH_FS_CASES))+'\n'
        result+='OWNERSHIP_CATALOG_RESULT '+' '.join(c+'=pass' for c in sorted(checker.CATALOG_CASES))+'\n'
        result+='OWNERSHIP_CAPACITY_RESULT '+' '.join(c+'=pass' for c in sorted(checker.CAPACITY_CASES))+'\n'
        result+='JOB_READ_RESULT '+' '.join(c+'=pass' for c in sorted(checker.JOB_READ_CASES))+'\n'
        result+='STATE_CATALOG_RESULT '+' '.join(c+'=pass' for c in sorted(checker.STATE_CATALOG_CASES))+'\n'
        result+='JOB_PUBLICATION_RESULT '+' '.join(c+'=pass' for c in sorted(checker.JOB_PUBLICATION_CASES))+'\n'
        result+='PROPERTY_JOB_STORAGE_RESULT '+' '.join(c+'=pass' for c in sorted(checker.PROPERTY_JOB_STORAGE_CASES))+'\n'
        result+='JOB_RESERVATIONS_RESULT '+' '.join(c+'=pass' for c in sorted(checker.RESERVATION_CASES))+'\n'
        result+='PROPERTY_SUBMISSION_RESULT '+' '.join(c+'=pass' for c in sorted(checker.SUBMISSION_CASES))+'\n'
        result+='PROPERTY_PUBLICATION_CUTS_RESULT '+' '.join(c+'=pass' for c in sorted(checker.CUT_CASES))+'\n'
        result+='ENROLLMENT_PUBLICATION_RESULT '+' '.join(c+'=pass' for c in sorted(checker.ENROLLMENT_CASES))+'\n'
        result+='STATE_SOFTUPDATES_RESULT '+' '.join(c+'=pass' for c in sorted(checker.SOFTUPDATES_CASES))+'\n'
        result+=''.join('STATE_BACKING_RESULT '+name+(' filesystem=ufs softupdates=on' if name=='TestNativeStateSoftUpdates' else ' filesystem=zfs softupdates=off')+' teardown=pass\n' for name in sorted(checker.BACKING_TESTS))
        passed=''.join('--- PASS: '+name+' (1.0s)\n' for name in checker.TESTS)+''.join('    --- PASS: '+name+'/case_'+str(i)+' (0.0s)\n' for name,count in checker.CODEC_SUBTEST_COUNTS.items() for i in range(count))
        self.assertIn('HORVIUM_STATE_VM=1',self.module.SUITES['state'][1])
        self.module.validate_result('state',passed+result)
        for bad in ['',passed,result,passed+result.replace('process_reaped=pass',''),passed+result.replace('fixture_teardown=pass','fixture_teardown=fail'),passed+result.replace('PROPERTY_JOB_STORAGE_RESULT','MISSING_PROPERTY_RESULT'),passed+result.replace('JOB_RESERVATIONS_RESULT','MISSING_RESERVATIONS_RESULT'),passed+result.replace('PROPERTY_SUBMISSION_RESULT','MISSING_SUBMISSION_RESULT'),passed+result.replace('PROPERTY_PUBLICATION_CUTS_RESULT','MISSING_CUTS_RESULT'),passed+result.replace('ENROLLMENT_PUBLICATION_RESULT','MISSING_ENROLLMENT_RESULT')]:
            with self.assertRaises(ValueError):self.module.validate_result('state',bad)

    def test_state_selectors_match_the_required_freebsd_test_inventory(self):
        checker=self.module.load_tool('check-state-evidence')
        available=set()
        for path in (ROOT/'internal/storage').glob('*_test.go'):
            if path.name.endswith('_other_test.go'):
                continue
            available.update(re.findall(r'func (Test\w+)\(',path.read_text()))
        for name in ['bsd-state.sh','bsd-state-prebuilt.sh']:
            source=(ROOT/'tools'/name).read_text()
            selectors=re.findall(r"(?:-test.run=|-run )'([^']+)'",source)
            self.assertEqual(len(selectors),1)
            selected={test for test in available if re.search(selectors[0],test)}
            with self.subTest(script=name):self.assertEqual(selected,checker.TESTS)

    def test_prebuilt_verification_rejects_missing_duplicate_or_wrong_digest(self):
        expected = 'a' * 64
        self.module.validate_prebuilt('STATE_PREBUILT_SHA256 ' + expected + '\n', expected)
        for bad in ['', 'STATE_PREBUILT_SHA256 ' + 'b' * 64 + '\n',
                    ('STATE_PREBUILT_SHA256 ' + expected + '\n') * 2]:
            with self.assertRaises(ValueError):
                self.module.validate_prebuilt(bad, expected)

    def test_cleanup_waits_for_owned_child_and_does_not_signal_arbitrary_pids(self):
        process = Mock()
        process.poll.return_value = None
        self.module.stop_guest(process)
        process.terminate.assert_called_once_with()
        process.wait.assert_called_once_with(timeout=10)
        process.kill.assert_not_called()

    def test_unresponsive_owned_child_is_killed_and_waited_before_cleanup(self):
        process = Mock()
        process.poll.return_value = None
        process.wait.side_effect = [subprocess.TimeoutExpired('qemu', 10), 0]
        self.module.stop_guest(process)
        process.kill.assert_called_once_with()
        self.assertEqual(process.wait.call_count, 2)

    def test_boot_without_suite_assertions_is_not_discovery_qualification(self):
        with self.assertRaises(ValueError):
            self.module.validate_result('discovery', 'VM booted\nPASS\n')

    def test_device_suite_requires_guest_and_host_teardown_evidence(self):
        checker = self.module.load_tool('check-device-evidence')
        guest = '--- PASS: TestNativeDeviceFixtures (1.0s)\nDEVICE_OBSERVATION_RESULT ' + ' '.join(case + '=pass' for case in sorted(checker.CASES)) + '\n'
        with self.assertRaises(ValueError):
            self.module.validate_result('devices', guest)
        guest += 'DEVICE_CORRELATION_RESULT ' + ' '.join(case + '=pass' for case in sorted(checker.CORRELATION_CASES)) + '\n'
        with self.assertRaises(ValueError):
            self.module.validate_result('devices', guest)
        guest += '--- PASS: TestNativeDiskAttachments (1.0s)\nDEVICE_ATTACHMENT_RESULT ' + ' '.join(case + '=pass' for case in sorted(checker.ATTACHMENT_CASES)) + '\n'
        with self.assertRaises(ValueError):
            self.module.validate_result('devices', guest)
        guest += '--- PASS: TestNativeSystemDeviceUse (1.0s)\nDEVICE_SYSTEM_USE_RESULT ' + ' '.join(case + '=pass' for case in sorted(checker.SYSTEM_USE_CASES)) + '\n'
        with self.assertRaises(ValueError):
            self.module.validate_result('devices', guest)
        guest += '--- PASS: TestNativeImportedPoolUse (1.0s)\nDEVICE_IMPORTED_POOL_RESULT ' + ' '.join(case + '=pass' for case in sorted(checker.IMPORTED_POOL_CASES)) + '\n'
        with self.assertRaises(ValueError):
            self.module.validate_result('devices', guest)
        guest += 'DEVICE_SIGNATURE_UFS_RESULT ' + ' '.join(case + '=pass' for case in sorted(checker.SIGNATURE_UFS_CASES)) + '\n'
        with self.assertRaises(ValueError):
            self.module.validate_result('devices', guest)
        guest += 'DEVICE_SIGNATURE_ZFS_RESULT ' + ' '.join(case + '=pass' for case in sorted(checker.SIGNATURE_ZFS_CASES)) + '\n'
        with self.assertRaises(ValueError):
            self.module.validate_result('devices', guest)
        guest += 'DEVICE_USE_POOL_RESULT ' + ' '.join(case + '=pass' for case in sorted(checker.USE_POOL_CASES)) + '\n'
        with self.assertRaises(ValueError):
            self.module.validate_result('devices', guest)
        guest += 'DEVICE_USE_SYSTEM_RESULT ' + ' '.join(case + '=pass' for case in sorted(checker.USE_SYSTEM_CASES)) + '\n'
        with self.assertRaises(ValueError):
            self.module.validate_result('devices', guest)
        guest += 'DEVICE_POOL_LABEL_RESULT ' + ' '.join(case + '=pass' for case in sorted(checker.POOL_LABEL_CASES)) + '\n'
        with self.assertRaises(ValueError):
            self.module.validate_result('devices', guest)
        guest += 'DEVICE_ACCESS_RESULT ' + ' '.join(case + '=pass' for case in sorted(checker.ACCESS_CASES)) + '\n'
        self.module.validate_result('devices', guest)
        with self.assertRaises(ValueError):
            self.module.validate_result('devices', guest, complete=True)
        self.module.validate_result('devices', guest + 'DEVICE_FIXTURE_CLEANUP pass\n', complete=True)


if __name__ == '__main__':
    unittest.main()
