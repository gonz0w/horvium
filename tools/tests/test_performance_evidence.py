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

    def test_synthetic_pass_without_native_reader_execution_is_rejected(self):
        rows = []
        for count in (100, 10000):
            row = {'datasets': count, 'clients': 10, 'requests': 1000,
                   'replacement_requests': 1000, 'replacements': 3,
                   'warmup_seconds': 60, 'idle_seconds': 300 if count == 100 else 0,
                   'sampled_combined_peak_rss_bytes': 42 << 20,
                   'budget_peak_rss_bytes': 42 << 20,
                   'conservative_combined_high_water_rss_bytes': 40 << 20,
                   'rss_samples': 600, 'p95_ms': 100,
                   'idle_cpu_percent_one_core': 0.1}
            rows.append('PERFORMANCE_RESULT ' + json.dumps(row))
        rows.append('PERFORMANCE_LIMITS accepted_model_records=100000 record_exhaustion=pass byte_exhaustion=pass recovery=pass')
        rows.append('--- PASS: TestNativePerformanceBudgets (430s)')
        with self.assertRaises(ValueError):
            self.checker()('\n'.join(rows))

    def test_complete_measurements_and_recovery_are_accepted(self):
        rows = []
        for count in (100, 10000):
            row = {'datasets': count, 'clients': 10, 'requests': 1000,
                   'replacement_requests': 1000, 'replacements': 3,
                   'warmup_seconds': 60, 'idle_seconds': 300 if count == 100 else 0,
                   'sampled_combined_peak_rss_bytes': 42 << 20,
                   'budget_peak_rss_bytes': 42 << 20,
                   'conservative_combined_high_water_rss_bytes': 40 << 20,
                   'rss_samples': 600, 'p95_ms': 100,
                   'idle_cpu_percent_one_core': 0.1}
            rows.append('PERFORMANCE_RESULT ' + json.dumps(row))
        rows.append('PERFORMANCE_LIMITS accepted_model_records=100000 record_exhaustion=pass byte_exhaustion=pass recovery=pass')
        rows.append('--- PASS: TestNativePerformanceBudgets (430s)')
        for count, mode in ((100, 'native-small'), (10000, 'projection-10000')):
            row = json.loads(rows[0 if count == 100 else 1].split('PERFORMANCE_RESULT ', 1)[1])
            row.update(mode=mode, owned_datasets=100 if count == 100 else 0)
            rows.append('NATIVE_READER_RESULT ' + json.dumps(row))
        rows.append('NATIVE_READER_CRASH captured_native_command=pass subtree_exit=pass stale_retention=pass recovery=pass boot_state=unchanged')
        rows.append('--- PASS: TestNativeReaderResourceBudgets (430s)')
        self.assertEqual(set(self.checker()('\n'.join(rows))), {100, 10000})
        for missing in ('NATIVE_READER_CRASH', '--- PASS: TestNativeReaderResourceBudgets', 'NATIVE_READER_RESULT'):
            with self.assertRaises(ValueError):
                self.checker()('\n'.join(line for line in rows if missing not in line))
        # A below-budget high-water sum cannot hide an above-budget sample.
        oversized = '\n'.join(rows).replace(str(42 << 20), str(129 << 20))
        with self.assertRaises(ValueError):
            self.checker()(oversized)

if __name__ == '__main__':
    unittest.main()
