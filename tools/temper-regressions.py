#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Run the fixed affected review regressions; missing/skipped tests cannot pass."""
import json
import os
import platform
import sys
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
MODULES = ('test_review_plan', 'test_review_gate', 'test_temper_review_scope',
           'test_temper_gate_dispatch', 'test_temper_scope_binding',
           'test_temper_bound_dispatch', 'test_temper_source')


def run_modules(loader=None, names=MODULES):
    loader = loader or unittest.TestLoader()
    suites = []
    for name in names:
        suite = loader.loadTestsFromName(name)
        if loader.errors or not suite.countTestCases():
            raise ValueError('missing affected regression: ' + name)
        suites.append(suite)
    result = unittest.TextTestRunner(verbosity=2).run(unittest.TestSuite(suites))
    record = {'modules': list(names), 'tests_run': result.testsRun,
              'failures': len(result.failures), 'errors': len(result.errors),
              'skipped': len(result.skipped), 'successful': result.wasSuccessful()}
    if not result.wasSuccessful() or result.skipped or not result.testsRun:
        raise ValueError('affected regressions failed or skipped: ' + json.dumps(record, sort_keys=True))
    return record


def main():
    if platform.system() not in {'FreeBSD', 'OpenBSD'} or os.getuid() == 0:
        raise RuntimeError('Affected LAB regressions require an authorized unprivileged BSD')
    sys.path.insert(0, str(ROOT / 'tools/tests'))
    print('REGRESSION_RESULT ' + json.dumps(run_modules(), sort_keys=True), flush=True)


if __name__ == '__main__':
    main()
