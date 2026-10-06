# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
from pathlib import Path
import unittest
ROOT = Path(__file__).resolve().parents[2]

class DeepCheckTests(unittest.TestCase):

    def module(self):
        s = importlib.util.spec_from_file_location('deep', ROOT / 'tools/deep-check.py')
        m = importlib.util.module_from_spec(s)
        s.loader.exec_module(m)
        return m

    def test_fuzz_inventory_requires_compiler_listed_targets(self):
        m = self.module()
        self.assertEqual(m.fuzz_names('FuzzParser\nFuzzScope\nok \tfixture (0.1s)\n'), ['FuzzParser', 'FuzzScope'])
        self.assertEqual(m.fuzz_names('? \tfixture [no test files]\n'), [])
        for bad in ['FuzzParser\nFuzzParser\n', 'FuzzParser;rm -rf /\n', 'FAIL\n']:
            with self.assertRaises(ValueError):
                m.fuzz_names(bad)

    def test_deep_commands_are_scoped_and_never_shell_strings(self):
        m = self.module()
        cmd = m.fuzz_command('/go', './internal/a', 'FuzzParser')
        self.assertEqual(cmd, ['/go', 'test', './internal/a', '-run', '^$', '-fuzz', '^FuzzParser$', '-fuzztime=3s', '-parallel=2'])
        for package in ['./...', '../outside', './internal/a;echo', '-exec=bad']:
            with self.assertRaises(ValueError):
                m.valid_package(package)
