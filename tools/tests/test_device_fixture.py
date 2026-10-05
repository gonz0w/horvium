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

    def test_pid_reuse_and_command_mismatch_cannot_authorize_cleanup_signal(self):
        m=self.module()
        args=[b'/usr/bin/qemu-system-x86_64',b'-snapshot',b'-drive',b'file=/owned']
        record={'pid':1234,'start_ticks':'9876','argv_sha256':hashlib.sha256(b'\0'.join(args[1:])).hexdigest()}
        self.assertEqual(m.verify_process(record,args,'9876'),1234)
        for ticks,argv in [('9877',args),('9876',[b'/usr/bin/other']+args[1:]),('9876',args+[b'-drive',b'file=/foreign'])]:
            with self.subTest(ticks=ticks,argv=argv),self.assertRaises(ValueError):m.verify_process(record,argv,ticks)
        for pid in [True,1,0,-1,'1234']:
            with self.subTest(pid=pid),self.assertRaises(ValueError):m.verify_process(dict(record,pid=pid),args,'9876')
