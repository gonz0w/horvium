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
    for count, row in rows.items():
        expected = {'clients': 10, 'requests': 1000, 'replacement_requests': 1000,
                    'replacements': 3, 'warmup_seconds': 60,
                    'idle_seconds': 300 if count == 100 else 0}
        if any(row.get(key) != value for key, value in expected.items()):
            raise ValueError('fixture procedure is incomplete')
        if not 0 < row.get('sampled_combined_peak_rss_bytes', 0) <= row.get('conservative_combined_high_water_rss_bytes', 0):
            raise ValueError('RSS evidence missing or inconsistent')
        if row['conservative_combined_high_water_rss_bytes'] > (128 if count == 100 else 256) * 1024 * 1024:
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
    return rows


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('log', type=Path)
    args = parser.parse_args()
    validate(args.log.read_text())
    print('PASS: both measured fixtures and limit recovery executed within budgets')


if __name__ == '__main__':
    main()
