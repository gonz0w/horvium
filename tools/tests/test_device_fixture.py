# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
from pathlib import Path
import tempfile
import hashlib
import unittest

SOURCE = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location('device_fixture', SOURCE / 'tools/device-fixture.py')

class DeviceFixtureTests(unittest.TestCase):
    def module(self):
        module = importlib.util.module_from_spec(SPEC)
        SPEC.loader.exec_module(module)
        return module

    def test_only_declared_disks_and_geometry_are_appended_to_snapshot_vm(self):
        m = self.module()
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp).resolve() / 'devices'
            m.prepare(root, size=4096)
            args = ['-snapshot', '-drive', 'file=/fixture/base.qcow2,format=qcow2']
            command = m.command(args, root, size=4096)
            self.assertEqual(command[:len(args)], args)
            self.assertEqual(command.count('-drive'), 6)
            self.assertEqual(command.count('-device'), 5)
            firmware = args + ['-drive','if=pflash,format=raw,readonly=on,file=/fixture/code', '-drive','if=pflash,format=raw,file=/fixture/vars']
            self.assertEqual(m.command(firmware, root, size=4096).count('-drive'), 8)
            self.assertIn('serial=HVM-UNIQUE-512', ','.join(command))
            self.assertIn('logical_block_size=4096,physical_block_size=4096', ','.join(command))
            self.assertEqual(','.join(command).count('serial=HVM-DUPLICATE'), 2)
            self.assertNotIn('serial=', command[-1])

    def test_queries_pass_through_but_unqualified_launch_is_denied(self):
        m = self.module()
        self.assertEqual(m.command(['--version'], Path('/absent')), ['--version'])
        self.assertEqual(m.command(['-accel', 'help'], Path('/absent')), ['-accel', 'help'])
        for args in [[], ['-drive','file=/base'], ['-snapshot'], ['-snapshot','-drive','file=/base','-drive','file=/other']]:
            with self.subTest(args=args), self.assertRaises(ValueError):
                m.command(args, Path('/absent'))

    def test_symlink_changed_size_or_changed_contents_invalidates_fixture(self):
        m=self.module()
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp).resolve()/'devices'; m.prepare(root, size=4096)
            m.verify(root, size=4096)
            backing=root/'disk-0.raw'
            backing.write_bytes(b'x'+b'\0'*4095)
            with self.assertRaises(ValueError): m.verify(root, size=4096)
            backing.write_bytes(b'changed')
            with self.assertRaises(ValueError): m.verify(root, size=4096)
            backing.unlink(); backing.symlink_to(root/'disk-1.raw')
            with self.assertRaises(ValueError): m.verify(root, size=4096)

    def test_existing_directory_and_unsafe_qemu_option_paths_are_denied(self):
        m=self.module()
        with tempfile.TemporaryDirectory() as tmp:
            with self.assertRaises(FileExistsError): m.prepare(Path(tmp).resolve(), size=4096)
            with self.assertRaises(ValueError): m.prepare(Path(tmp).resolve()/'a,b', size=4096)

    def test_preservation_failure_still_removes_owned_disposable_backings(self):
        m = self.module()
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp).resolve() / 'devices'
            m.prepare(root, size=4096)
            (root / 'disk-0.raw').write_bytes(b'x' * 4096)
            with self.assertRaises(ValueError):
                m.cleanup(root, size=4096)
            self.assertFalse(root.exists())

    def test_partial_preparation_cleanup_preserves_foreign_entries(self):
        m = self.module()
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp).resolve() / 'devices'
            root.mkdir(mode=0o700)
            (root / 'disk-0.raw').write_text('owned')
            (root / 'sentinel').write_text('foreign')
            with self.assertRaises(OSError):
                m.remove(root)
            self.assertEqual((root / 'sentinel').read_text(), 'foreign')
            self.assertFalse((root / 'disk-0.raw').exists())

    def test_fixture_is_private_and_contains_no_linux_launcher(self):
        m = self.module()
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp).resolve() / 'devices'
            m.prepare(root, size=4096)
            self.assertFalse((root / 'bin').exists())
            self.assertEqual(len(list(root.iterdir())), 6)
            root.chmod(0o755)
            with self.assertRaises(ValueError):
                m.verify(root, size=4096)

    def test_detached_qemu_launch_cannot_escape_supervision(self):
        m = self.module()
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp).resolve() / 'devices'
            m.prepare(root, size=4096)
            with self.assertRaises(ValueError):
                m.command(['-snapshot', '-drive', 'file=/fixture/base', '-daemonize'], root, size=4096)

    def test_policy_facts_profile_adds_only_two_unique_disks_and_cleans_all_seven(self):
        m = self.module()
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp).resolve() / 'devices'
            m.prepare(root, size=4096, profile='policy-facts')
            records = m.verify(root, size=4096, profile='policy-facts')
            self.assertEqual(len(records), 7)
            self.assertEqual([(r['serial'], r['logical'], r['physical']) for r in records[:5]], m.DISKS)
            self.assertEqual([(r['serial'], r['logical'], r['physical']) for r in records[5:]],
                             [('HVM-FACTS-512', 512, 4096), ('HVM-FACTS-4K', 4096, 4096)])
            args = ['-snapshot', '-drive', 'file=/fixture/base.qcow2,format=qcow2']
            command = m.command(args, root, size=4096, profile='policy-facts')
            self.assertEqual(command.count('-device'), 7)
            self.assertEqual(command[:len(args)], args)
            with self.assertRaises(ValueError):
                m.verify(root, size=4096)
            m.cleanup(root, size=4096, profile='policy-facts')
            self.assertFalse(root.exists())
            baseline = Path(tmp).resolve() / 'baseline'
            m.prepare(baseline, size=4096)
            self.assertEqual(len(m.verify(baseline, size=4096)), 5)
            with self.assertRaises(ValueError):
                m.verify(baseline, size=4096, profile='policy-facts')
            m.cleanup(baseline, size=4096)

    def test_scoped_profile_failure_cleanup_and_unknown_profiles_remain_safe(self):
        m = self.module()
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp).resolve() / 'devices'
            with self.assertRaises(ValueError):
                m.prepare(root, size=4096, profile='unknown')
            self.assertFalse(root.exists())
            m.prepare(root, size=4096, profile='policy-facts')
            (root / 'disk-6.raw').write_bytes(b'x' * 4096)
            with self.assertRaises(ValueError):
                m.cleanup(root, size=4096, profile='policy-facts')
            self.assertFalse(root.exists())
