# SPDX-License-Identifier: BSD-3-Clause
import importlib.util
import json
import unittest
from pathlib import Path

SOURCE = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location('performance_evidence', SOURCE / 'tools/check-performance-evidence.py')

class PerformanceEvidenceTests(unittest.TestCase):
    def checker(self):
        module = importlib.util.module_from_spec(SPEC)
        SPEC.loader.exec_module(module)
        return module.validate

    def test_green_boot_without_measured_records_is_not_qualification(self):
        with self.assertRaises(ValueError):
            self.checker()('VM booted\nPASS\n')

    def test_duplicate_small_results_cannot_replace_large_fixture(self):
        line = 'PERFORMANCE_RESULT ' + json.dumps({'datasets': 100})
        with self.assertRaises(ValueError):
            self.checker()(line + '\n' + line)

if __name__ == '__main__':
    unittest.main()
