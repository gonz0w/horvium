# SPDX-License-Identifier: BSD-3-Clause
"""Pending portable selector contract; native RED has NOT RUN.

Only the pure selector is loaded. No Git, Go, compiler, process supervisor,
repository checker, network or product command is invoked by these cases.
"""
import importlib.util
from pathlib import Path
import platform
import unittest

if platform.system() not in {'FreeBSD', 'OpenBSD'}:
    raise RuntimeError('This pending LAB contract requires an authorized BSD')

ROOT = Path(__file__).resolve().parents[2]
ROLE_FILES = (
    '.codex/agents/temper_workflow.toml',
    '.codex/agents/temper_implementer.toml',
    '.codex/agents/temper_analyst.toml',
    '.codex/agents/temper_reviewer.toml',
)
BUNDLE_FILES = (
    '.agent-temper/LICENSE', '.agent-temper/METHOD.md',
    '.agent-temper/SOURCE.md', '.agent-temper/SHA256SUMS',
    '.agent-temper/docs/reference/documentation.md',
    '.agent-temper/docs/reference/extensions.md',
    '.agent-temper/docs/reference/verification.md',
    '.agent-temper/docs/reference/workflow.md',
    '.agent-temper/profiles/codex.md', '.agent-temper/profiles/github.md',
    '.agent-temper/profiles/strict-test-first.md',
)


class TemperReviewScopeTests(unittest.TestCase):
    def setUp(self):
        spec = importlib.util.spec_from_file_location('review_plan_under_test', ROOT / 'tools/review-plan.py')
        self.plan = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.plan)

    def test_selected_roles_choose_temper_scope(self):
        self.assertEqual(self.plan.select_plan(list(ROLE_FILES), {})['mode'], 'temper')

    def test_pinned_bundle_files_choose_temper_scope(self):
        self.assertEqual(self.plan.select_plan(list(BUNDLE_FILES), {})['mode'], 'temper')

    def test_prose_with_selected_roles_remains_temper_scope(self):
        paths = ['README.md', 'docs/testing/status.md', ROLE_FILES[0]]
        self.assertEqual(self.plan.select_plan(paths, {})['mode'], 'temper')

    def test_temper_scope_has_no_selected_product_packages(self):
        selected = self.plan.select_plan([ROLE_FILES[0]], {})
        self.assertEqual(selected['mode'], 'temper')
        self.assertEqual(selected['packages'], [])
        self.assertFalse(selected['native_review_required'])

    def test_unknown_or_global_codex_paths_keep_full_fallback(self):
        for name in ['.codex/config.toml', '.codex/agents/unknown.toml', '.codex/permissions.toml']:
            with self.subTest(path=name):
                self.assertEqual(self.plan.select_plan([name], {})['mode'], 'full')

    def test_new_executable_bundle_paths_keep_full_fallback(self):
        self.assertEqual(self.plan.select_plan(['.agent-temper/tools/runner.py'], {})['mode'], 'full')

    def test_unbound_mixed_policy_is_not_pure_temper(self):
        for name in ['AGENTS.md', 'PROJECT.md', 'CONTRIBUTING.md', 'docs/engineering/local-review.md']:
            with self.subTest(path=name):
                self.assertEqual(self.plan.select_plan([ROLE_FILES[0], name], {})['mode'], 'full')

    def test_selector_tests_and_shared_execution_do_not_exempt_themselves(self):
        paths = ['tools/review-plan.py', 'tools/review-gate.py', 'tools/tests/test_temper_review_scope.py',
                 '.githooks/pre-commit', 'Makefile', 'go.mod', 'packaging/ports/README.md']
        for name in paths:
            with self.subTest(path=name):
                self.assertEqual(self.plan.select_plan([ROLE_FILES[0], name], {})['mode'], 'full')

    def test_product_input_mixed_with_temper_keeps_full_fallback(self):
        self.assertEqual(self.plan.select_plan([ROLE_FILES[0], 'internal/storage/change.go'], {})['mode'], 'full')

    def test_explicit_full_review_is_never_narrowed(self):
        self.assertEqual(self.plan.select_plan(list(ROLE_FILES), {}, True)['mode'], 'full')
