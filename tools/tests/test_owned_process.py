# SPDX-License-Identifier: BSD-3-Clause
"""Real process regressions: owner death, descendants, escalation and isolation."""
import importlib.util
import json
import os
import signal
import subprocess
import sys
import tempfile
import time
import unittest
from unittest.mock import patch
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
TOOL = ROOT / 'tools/owned-process.py'


def eventually(predicate, timeout=12):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        if predicate():
            return
        time.sleep(0.05)
    raise AssertionError('process fixture deadline expired')


class OwnedProcessTests(unittest.TestCase):
    def setUp(self):
        spec = importlib.util.spec_from_file_location('owned_process', TOOL)
        self.tool = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.tool)
        self.temp = tempfile.TemporaryDirectory(prefix='hv-owned-', dir='/private/tmp' if Path('/private/tmp').is_dir() else None)
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name).resolve()

    def worker(self, ignore=False, exit_early=False):
        # The listening child is an observable descendant; its process group is
        # inherited. The parent's early exit must not orphan it.
        child = "import socket,time; s=socket.socket(); s.bind(('127.0.0.1',0)); s.listen(); open(%r,'w').write(str(s.getsockname()[1])); time.sleep(60)" % str(self.root / 'port')
        return [sys.executable, '-c',
                "import subprocess,sys,time,signal; " +
                ("signal.signal(signal.SIGTERM,signal.SIG_IGN); " if ignore else '') +
                "subprocess.Popen([sys.executable,'-c',%r]); " % child +
                ("time.sleep(0.4); sys.exit(0)" if exit_early else "time.sleep(60)")]

    def closed(self):
        import socket
        if not (self.root / 'port').exists():
            return False
        try:
            port = int((self.root / 'port').read_text())
        except ValueError:
            return False
        with socket.socket() as s:
            s.settimeout(0.2)
            return s.connect_ex(('127.0.0.1', port)) != 0

    def test_normal_exit_cleans_living_descendant(self):
        process = self.tool.OwnedProcess(self.worker(exit_early=True), runs=self.root / 'runs', grace=0.5)
        self.addCleanup(process.stop)
        self.assertEqual(process.wait(timeout=15), 0)
        self.assertTrue(self.closed())
        self.assertTrue(process.record()['group_cleaned'])
        self.assertFalse((process.directory / 'control.sock').exists())

    def test_term_escalates_and_unrelated_process_survives(self):
        sentinel = subprocess.Popen([sys.executable, '-c', 'import time; time.sleep(60)'])
        self.addCleanup(lambda: (sentinel.terminate(), sentinel.wait(timeout=5)))
        process = self.tool.OwnedProcess(self.worker(ignore=True), runs=self.root / 'runs', grace=0.5)
        self.addCleanup(process.stop)
        eventually(lambda: (self.root / 'port').exists())
        process.stop()
        self.assertTrue(process.record()['group_cleaned'])
        self.assertTrue(process.record()['escalated'])
        self.assertTrue(self.closed())
        self.assertIsNone(sentinel.poll())

    def test_killed_controller_is_cleaned_without_pid_recovery(self):
        code = "import importlib.util,time; from pathlib import Path; s=importlib.util.spec_from_file_location('owned',%r); m=importlib.util.module_from_spec(s); s.loader.exec_module(m); p=m.OwnedProcess(%r,runs=Path(%r),grace=0.5); time.sleep(60)" % (str(TOOL), self.worker(), str(self.root / 'runs'))
        owner = subprocess.Popen([sys.executable, '-c', code])
        self.addCleanup(lambda: owner.poll() is None and (owner.kill(), owner.wait(timeout=5)))
        eventually(lambda: (self.root / 'port').exists())
        owner.kill()
        owner.wait(timeout=5)
        session = next((self.root / 'runs').iterdir())
        eventually(lambda: json.loads((session / 'state.json').read_text()).get('completed', False))
        record = json.loads((session / 'state.json').read_text())
        self.assertEqual(record['reason'], 'owner-disconnected')
        self.assertTrue(record['group_cleaned'])
        self.assertTrue(self.closed())

    def test_explicit_stop_uses_owned_socket(self):
        process = self.tool.OwnedProcess(self.worker(), runs=self.root / 'runs', grace=0.5)
        self.addCleanup(process.stop)
        eventually(lambda: (self.root / 'port').exists())
        self.assertTrue(self.tool.request_stop(process.directory))
        process.wait(timeout=15)
        self.assertTrue(self.closed())
        self.assertEqual(process.record()['reason'], 'operator-stop')

    def test_timeout_is_not_a_success(self):
        process = self.tool.OwnedProcess([sys.executable, '-c', 'import time; time.sleep(60)'], runs=self.root / 'runs', timeout=0.2, grace=0.5)
        self.addCleanup(process.stop)
        self.assertNotEqual(process.wait(timeout=15), 0)
        self.assertEqual(process.record()['reason'], 'timeout')
        self.assertTrue(process.record()['group_cleaned'])

    def test_symlink_session_is_refused(self):
        link = self.root / 'link'
        link.symlink_to(self.root, target_is_directory=True)
        with self.assertRaises(ValueError):
            self.tool.request_stop(link)

    def test_interrupted_native_cleanup_is_scoped_and_unqualified(self):
        self.tool.ROOT = self.root
        lab = self.root / '.tools/lab-runs/fixture'
        lab.mkdir(parents=True, mode=0o700)
        (lab / 'fixture-key').write_text('synthetic fixture credential')
        (lab / 'vars.fd').write_text('synthetic variables')
        (lab / 'state-native.test').write_text('synthetic executable')
        (lab / 'pool-properties-native.test').write_text('synthetic executable')
        (lab / 'image').mkdir()
        (lab / 'image/base').write_text('disposable image')
        (lab / 'result.json').write_text(json.dumps({'qualified': True}))
        sentinel = self.root / 'unrelated'
        sentinel.write_text('preserve')
        self.tool.native_cleanup(lab, interrupted=True)
        result = json.loads((lab / 'result.json').read_text())
        self.assertFalse(result['qualified'])
        self.assertTrue(result['supervisor_fixture_cleanup'])
        for name in ('fixture-key', 'vars.fd', 'image', 'state-native.test', 'pool-properties-native.test'):
            self.assertFalse((lab / name).exists())
        self.assertEqual(sentinel.read_text(), 'preserve')
        with self.assertRaises(ValueError):
            self.tool.native_cleanup(self.root, interrupted=True)

    def test_failing_worker_never_has_success_status(self):
        process = self.tool.OwnedProcess([sys.executable, '-c', 'import sys; sys.exit(7)'], runs=self.root / 'runs', grace=0.5)
        self.addCleanup(process.stop)
        self.assertEqual(process.wait(timeout=15), 7)
        self.assertTrue(process.record()['group_cleaned'])

    def test_control_bind_failure_starts_no_worker_and_cleans_prepared_fixture(self):
        for native in [None, str(self.root / 'prepared-native')]:
            with self.subTest(native=native):
                session = self.root / ('plain-startup' if native is None else 'native-startup')
                session.mkdir(mode=0o700)
                self.tool.write_record(session / 'launch.json', {
                    'argv': [sys.executable, '-c', 'raise SystemExit(0)'],
                    'cwd': str(self.root), 'timeout': 1, 'grace': 0.5,
                    'native_fixture': native,
                })
                with patch.object(self.tool.socket, 'socket') as socket, \
                        patch.object(self.tool.subprocess, 'Popen') as launch, \
                        patch.object(self.tool.signal, 'signal'), \
                        patch.object(self.tool, 'native_cleanup') as cleanup:
                    socket.return_value.__enter__.return_value.bind.side_effect = PermissionError('owned bind denial')
                    with self.assertRaises(PermissionError):self.tool.supervise(session)
                    launch.assert_not_called()
                    if native is None:cleanup.assert_not_called()
                    else:cleanup.assert_called_once_with(Path(native), True)
                record = json.loads((session / 'state.json').read_text())
                self.assertTrue(record['completed'])
                self.assertTrue(record['group_cleaned'])
                self.assertEqual(record['reason'], 'supervisor-failure')
                self.assertNotIn('group', record)
                self.assertNotIn('worker_returncode', record)
                self.assertFalse((session / 'control.sock').exists())

    def test_interrupted_device_fixture_is_removed_with_native_resources(self):
        self.tool.ROOT = self.root
        import shutil
        (self.root / 'tools').mkdir()
        shutil.copyfile(ROOT / 'tools/device-fixture.py', self.root / 'tools/device-fixture.py')
        lab = self.root / '.tools/lab-runs/fixture'
        lab.mkdir(parents=True, mode=0o700)
        devices = lab / 'devices'
        devices.mkdir(mode=0o700)
        (devices / 'disk-0.raw').write_text('partially prepared disposable backing')
        (lab / 'fixture-key').write_text('synthetic credential')
        self.tool.native_cleanup(lab, interrupted=True)
        self.assertFalse(devices.exists())
        self.assertFalse((lab / 'fixture-key').exists())
        self.assertFalse(json.loads((lab / 'result.json').read_text())['qualified'])
