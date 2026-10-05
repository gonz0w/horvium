# SPDX-License-Identifier: BSD-3-Clause
"""Prevent narrow manual requests from accidentally starting unrelated guests."""
import re
import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[2]


class DispatchBudgetTests(unittest.TestCase):
    def test_default_starts_no_jobs_and_each_request_selects_only_its_suite(self):
        workflow = yaml.load((ROOT / '.github/workflows/bsd-smoke.yml').read_text(), Loader=yaml.BaseLoader)
        choices = workflow['on']['workflow_dispatch']['inputs']['suite']
        self.assertEqual(choices['default'], 'none')
        expected = {'none': set(), 'discovery': {'discovery'}, 'properties': {'properties'},
                    'continuity': {'continuity'}, 'performance': {'performance'},
                    'browser': {'browser'}, 'foundation': {'freebsd'}, 'research': {'research'},
                    'full': {'discovery', 'properties', 'performance', 'browser', 'freebsd', 'research'}}
        self.assertEqual(set(choices['options']), set(expected))
        for selected, wanted in expected.items():
            actual = set()
            for name, job in workflow['jobs'].items():
                condition = job.get('if', '')
                terms = condition.split(' || ')
                matches = [re.fullmatch(r"inputs\.suite == '([a-z]+)'", term) for term in terms]
                self.assertTrue(all(matches), f'{name}: unconditional or unsupported selection')
                if selected in [match[1] for match in matches]:
                    actual.add(name)
            self.assertEqual(actual, wanted, selected)


if __name__ == '__main__':
    unittest.main()
