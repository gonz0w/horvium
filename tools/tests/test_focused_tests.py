# SPDX-License-Identifier: BSD-3-Clause
"""A selected regression must execute tests; empty success cannot pass."""
import importlib.util
import os
from pathlib import Path
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]


class FocusedTests(unittest.TestCase):
    def setUp(self):
        spec = importlib.util.spec_from_file_location('focused', ROOT / 'tools/focused-tests.py')
        self.tool = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.tool)

    def test_no_matching_tests_refuses_success_and_execution(self):
        with patch.dict(os.environ, PKG='./internal/storage', TEST='^TestMissing$'):
            with patch.object(self.tool.subprocess, 'check_output', return_value='ok fixture\n'):
                with patch.object(self.tool.subprocess, 'run') as run:
                    with self.assertRaisesRegex(ValueError, 'no matching'):
                        self.tool.main()
                    run.assert_not_called()

    def test_named_tests_execute_once_without_shell(self):
        with patch.dict(os.environ, PKG='./internal/storage', TEST='^TestRoundTrip$'):
            with patch.object(self.tool.subprocess, 'check_output', return_value='TestRoundTrip\nok fixture\n'):
                with patch.object(self.tool.subprocess, 'run') as run:
                    self.tool.main()
                    command = run.call_args.args[0]
                    self.assertEqual(command[1:], ['test', './internal/storage', '-run', '^TestRoundTrip$', '-count=1'])
                    self.assertNotIn('shell', run.call_args.kwargs)

    def test_flags_cannot_be_injected_as_a_package(self):
        with patch.dict(os.environ, PKG='-exec=sh', TEST='^TestRoundTrip$'):
            with patch.object(self.tool.subprocess, 'check_output') as discover:
                with self.assertRaises(ValueError):
                    self.tool.main()
                discover.assert_not_called()
