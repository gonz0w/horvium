#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Reject successful VM boots or skipped suites without actual measurements."""
import argparse
import json
from pathlib import Path


def validate(log):
    rows = {}
    for line in log.splitlines():
        marker = 'PERFORMANCE_RESULT '
        if marker not in line:
            continue
        row = json.loads(line.split(marker, 1)[1])
        count = row.get('datasets')
        if count not in (100, 10000) or count in rows:
            raise ValueError('missing or duplicate fixture measurements')
        rows[count] = row
    if set(rows) != {100, 10000}:
        raise ValueError('both measured fixtures must execute')
    native = {}
    for line in log.splitlines():
        marker = 'NATIVE_READER_RESULT '
        if marker not in line:
            continue
        row = json.loads(line.split(marker, 1)[1])
        mode = row.get('mode')
        if mode not in ('native-small', 'projection-10000') or mode in native:
            raise ValueError('missing or duplicate native measurements')
        native[mode] = row
    if set(native) != {'native-small', 'projection-10000'}:
        raise ValueError('both native reader procedures must execute')
    small = native['native-small']
    large = native['projection-10000']
    if small.get('owned_datasets') != 100 or small.get('datasets', 0) < 100:
        raise ValueError('native owned fixture was not independently counted')
    if large.get('datasets') != 10000 or large.get('owned_datasets') != 0:
        raise ValueError('projection fixture is misrepresented')
    measured = list(rows.items()) + [(100, small), (10000, large)]
    for count, row in measured:
        expected = {'clients': 10, 'requests': 1000, 'replacement_requests': 1000,
                    'replacements': 3, 'warmup_seconds': 60,
                    'idle_seconds': 300 if count == 100 else 0}
        if any(row.get(key) != value for key, value in expected.items()):
            raise ValueError('fixture procedure is incomplete')
        sampled = row.get('sampled_combined_peak_rss_bytes', 0)
        high_water = row.get('conservative_combined_high_water_rss_bytes', 0)
        if min(sampled, high_water) <= 0 or row.get('budget_peak_rss_bytes') != max(sampled, high_water):
            raise ValueError('RSS evidence missing or inconsistent')
        if row['budget_peak_rss_bytes'] > (128 if count == 100 else 256) * 1024 * 1024:
            raise ValueError('RSS budget failed')
        if row.get('rss_samples', 0) < 1 or row.get('p95_ms', 0) <= 0:
            raise ValueError('measurement missing')
        if count == 100 and not 0 <= row.get('idle_cpu_percent_one_core', -1) < 1:
            raise ValueError('idle CPU budget failed')
        if count == 10000 and row['p95_ms'] >= 250:
            raise ValueError('latency budget failed')
    limits = 'PERFORMANCE_LIMITS accepted_model_records=100000 record_exhaustion=pass byte_exhaustion=pass recovery=pass'
    if limits not in log or '--- PASS: TestNativePerformanceBudgets ' not in log:
        raise ValueError('limit recovery or native suite did not pass')
    crash = 'NATIVE_READER_CRASH captured_native_command=pass subtree_exit=pass stale_retention=pass recovery=pass boot_state=unchanged'
    if crash not in log or '--- PASS: TestNativeReaderResourceBudgets ' not in log:
        raise ValueError('built-helper native crash or reader suite did not pass')
    return rows


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('log', type=Path)
    args = parser.parse_args()
    validate(args.log.read_text())
    print('PASS: synthetic and native reader measurements, crash and limit recovery passed')


if __name__ == '__main__':
    main()
