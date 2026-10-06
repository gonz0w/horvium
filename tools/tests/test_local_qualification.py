# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
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
        self.module.validate_result('devices', guest)
        with self.assertRaises(ValueError):
            self.module.validate_result('devices', guest, complete=True)
        self.module.validate_result('devices', guest + 'DEVICE_FIXTURE_CLEANUP pass\n', complete=True)


if __name__ == '__main__':
    unittest.main()
