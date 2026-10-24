# SPDX-License-Identifier: BSD-3-Clause
"""No-process failure injection for the fresh supervisor's finalizers."""
import importlib.util
import json
from pathlib import Path
import signal
import subprocess
import tempfile
import unittest
from unittest.mock import Mock, patch

SPEC = importlib.util.spec_from_file_location('observer_cleanup_owner', Path(__file__).parents[1] / 'owned-process.py')
OWNER = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(OWNER)


class ObserverCleanupTests(unittest.TestCase):
    def session(self, directory):
        root = Path(directory).resolve()
        OWNER.write_record(root / 'launch.json', dict(argv=['never-executed'], cwd=str(root),
                                                    timeout=1, grace=.1, native_fixture=None))
        return root

    def exercise(self, root, observer, *, wait_error=None, waitid_error=None):
        events = []
        process = Mock(pid=731)
        process.wait.side_effect = lambda **kw: events.append(('wait', kw['timeout'])) or 0
        if wait_error:
            process.wait.side_effect = wait_error
        with patch.object(OWNER.socket, 'socket') as socket, patch.object(OWNER.selectors, 'DefaultSelector'), \
                patch.object(OWNER.signal, 'signal'), patch.object(OWNER.subprocess, 'Popen', return_value=process), \
                patch.object(OWNER.os, 'waitid', side_effect=waitid_error, return_value=object()), \
                patch.object(OWNER, 'live_members', side_effect=observer), \
                patch.object(OWNER.os, 'killpg', side_effect=lambda group, sig: events.append(('signal', group, sig))):
            socket.return_value.__enter__.return_value.bind.side_effect = lambda path: Path(path).touch()
            if waitid_error:
                with self.assertRaises(type(waitid_error)):
                    OWNER.supervise(root)
            else:
                self.assertEqual(OWNER.supervise(root), 1)
        return events, process, json.loads((root / 'state.json').read_text())

    def test_observer_failures_cannot_suppress_kill_wait_or_terminal_record(self):
        for failure in (PermissionError('denied'), subprocess.CalledProcessError(1, 'ps'),
                        subprocess.TimeoutExpired('ps', .25), ValueError('bad row')):
            with self.subTest(failure=type(failure).__name__), tempfile.TemporaryDirectory() as directory:
                events, process, record = self.exercise(self.session(directory), failure)
                self.assertEqual([e[0] for e in events], ['signal', 'signal', 'wait'])
                self.assertEqual([e[2] for e in events[:2]], [signal.SIGTERM, signal.SIGKILL])
                process.wait.assert_called_once()
                self.assertTrue(record['completed'])
                self.assertFalse(record['group_cleaned'])
                self.assertTrue(record['cleanup_failures'])

    def test_unexpected_cleanup_failure_cannot_suppress_final_owned_kill(self):
        with tempfile.TemporaryDirectory() as directory, patch.object(OWNER, 'stop_group', side_effect=RuntimeError('observer hook')):
            events, _, record = self.exercise(self.session(directory), False)
        self.assertEqual([e[0] for e in events], ['signal', 'wait'])
        self.assertEqual(events[0][2], signal.SIGKILL)
        self.assertFalse(record['group_cleaned'])

    def test_wait_failure_still_writes_failed_terminal_record(self):
        with tempfile.TemporaryDirectory() as directory:
            _, process, record = self.exercise(self.session(directory), PermissionError(), wait_error=TimeoutError())
        process.wait.assert_called_once()
        self.assertFalse(record['group_cleaned'])
        self.assertTrue(any('sole wait' in s for s in record['cleanup_failures']))

    def test_lost_reservation_never_signals_or_waits_again(self):
        with tempfile.TemporaryDirectory() as directory:
            events, process, record = self.exercise(self.session(directory), True, waitid_error=ChildProcessError())
        self.assertEqual(events, [])
        process.wait.assert_not_called()
        self.assertFalse(record['group_cleaned'])

    def test_ps_timeout_uses_original_remaining_budget(self):
        with patch.object(OWNER.time, 'monotonic', return_value=9.9), \
                patch.object(OWNER.subprocess, 'check_output', return_value='731 Z\n') as observe:
            self.assertFalse(OWNER.live_members(731, 10))
            self.assertAlmostEqual(observe.call_args.kwargs['timeout'], .1)
        with patch.object(OWNER.time, 'monotonic', return_value=10), \
                patch.object(OWNER.subprocess, 'check_output') as observe:
            with self.assertRaises(TimeoutError):
                OWNER.live_members(731, 10)
            observe.assert_not_called()
