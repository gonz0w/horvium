# SPDX-License-Identifier: BSD-3-Clause
"""Disposable input verification; synthetic bytes are not a native build."""
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

from test_closed_startup_evidence import module
from test_policy_startup_provenance import fixture, SOURCE, COMPILER, ARTIFACT


class PolicyProvenanceCandidate(unittest.TestCase):
    def setUp(self):
        self.m = module('check-policy-provenance-candidate')
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.compiler = self.root / 'compiler'
        self.artifact = self.root / 'artifact'
        self.transcript = self.root / 'transcript'
        self.compiler.write_bytes(b'synthetic compiler, never executed')
        self.artifact.write_bytes(b'synthetic artifact, never executed')
        self.compiler.chmod(0o700)
        self.artifact.chmod(0o700)
        self.expected = dict(source_commit=SOURCE,
                             compiler_sha256=self.digest(self.compiler),
                             artifact_sha256=self.digest(self.artifact))
        self.write_transcript()

    def digest(self, path):
        return hashlib.sha256(path.read_bytes()).hexdigest()

    def write_transcript(self, record=None, operation=None):
        text = fixture(record, operation).replace(COMPILER, self.expected['compiler_sha256']).replace(ARTIFACT, self.expected['artifact_sha256'])
        self.transcript.write_text(text)

    def verify(self, **changes):
        return self.m.verify_candidate(self.root, self.transcript, self.compiler, self.artifact,
                                       **(self.expected | changes))

    def test_opened_files_and_external_identity_bind_complete_diagnostic(self):
        with patch.object(self.m, 'verify_source') as source:
            result = self.verify()
        self.assertEqual(source.call_count, 2)
        source.assert_called_with(self.root, SOURCE)
        self.assertTrue(result['diagnostic_complete'])
        self.assertFalse(result['native_acceptance'])
        self.assertEqual(result['transcript_sha256'], self.digest(self.transcript))
        self.assertEqual(result['identity'], dict(schema=1, profile='deadline-provenance-v1', **self.expected))

    def test_actual_files_cannot_be_replaced_by_matching_transcript_claims(self):
        with patch.object(self.m, 'verify_source'):
            for path in (self.compiler, self.artifact):
                original = path.read_bytes()
                path.write_bytes(b'substituted bytes')
                with self.subTest(path=path.name), self.assertRaises(ValueError):
                    self.verify()
                path.write_bytes(original)
            for key, value in [('source_commit', True), ('compiler_sha256', 'x' * 64),
                               ('artifact_sha256', 'd' * 64), ('source_commit', 'd' * 40)]:
                with self.subTest(key=key), self.assertRaises(ValueError):
                    self.verify(**{key: value})

    def test_unsafe_files_fail_before_reading_or_execution(self):
        with patch.object(self.m, 'verify_source'):
            for path in (self.compiler, self.artifact, self.transcript):
                original, mode = path.read_bytes(), path.stat().st_mode & 0o777
                other = self.root / 'other'
                other.write_bytes(original)
                path.unlink()
                path.symlink_to(other)
                with self.subTest(kind='symlink', path=path.name), self.assertRaises(ValueError):
                    self.verify()
                path.unlink()
                path.mkdir()
                with self.subTest(kind='directory', path=path.name), self.assertRaises(ValueError):
                    self.verify()
                path.rmdir()
                os.mkfifo(path)
                with self.subTest(kind='fifo', path=path.name), self.assertRaises(ValueError):
                    self.verify()
                path.unlink()
                path.write_bytes(original)
                path.chmod(mode)
            self.artifact.chmod(0o600)
            with self.assertRaises(ValueError):
                self.verify()

    def test_transcript_encoding_bounds_and_raw_records_fail_closed(self):
        with patch.object(self.m, 'verify_source'):
            for data in (b'\xff', b'x' * ((128 << 10) + 1),
                         self.transcript.read_bytes() + b'private raw inventory\n'):
                self.transcript.write_bytes(data)
                with self.subTest(size=len(data)), self.assertRaises(ValueError) as error:
                    self.verify()
                self.assertNotIn('private raw inventory', str(error.exception))
                self.assertNotIn(str(self.root), str(error.exception))

    def test_incomplete_trace_is_retained_but_never_runner_success(self):
        with patch.object(self.m, 'verify_source'):
            for status in ('unattributed', 'ambiguous', 'invalid', 'overflow'):
                self.write_transcript(dict(status=status, frames=[]), 'deadline')
                with self.subTest(status=status), self.assertRaises(ValueError) as error:
                    self.verify()
                record = error.exception.evidence
                self.assertFalse(record['diagnostic_complete'])
                self.assertFalse(record['native_acceptance'])
                self.assertEqual(record['trace'], dict(status=status, frames=[]))
                self.assertEqual(record['transcript_sha256'], self.digest(self.transcript))
            self.write_transcript(dict(status='success', frames=[]), 'none')
            self.assertTrue(self.verify()['diagnostic_complete'])

    def test_source_requires_actual_clean_signed_author_dco_head(self):
        responses = [str(self.root), SOURCE, '', 'G\nCam <cam@example.test>\n\nSigned-off-by: Cam <cam@example.test>\n']
        with patch.object(self.m, 'git', side_effect=responses):
            self.m.verify_source(self.root, SOURCE)
        for index, value in [(0, str(self.root / 'nested')), (1, 'd' * 40), (2, ' M source.go'),
                             (3, 'N\nCam <cam@example.test>\nSigned-off-by: Cam <cam@example.test>'),
                             (3, 'G\nCam <cam@example.test>\nSigned-off-by: Other <other@example.test>')]:
            changed = list(responses)
            changed[index] = value
            with self.subTest(index=index, value=value), patch.object(self.m, 'git', side_effect=changed), self.assertRaises(ValueError):
                self.m.verify_source(self.root, SOURCE)

    def test_source_change_after_validation_is_rejected(self):
        with patch.object(self.m, 'verify_source', side_effect=[None, ValueError('source changed')]) as source:
            with self.assertRaises(ValueError):
                self.verify()
        self.assertEqual(source.call_count, 2)

    def test_changed_open_file_and_path_replacement_are_rejected(self):
        original_read = os.read
        for replacement in (False, True):
            changed = False
            def read(fd, count):
                nonlocal changed
                data = original_read(fd, count)
                if not changed:
                    changed = True
                    if replacement:
                        self.compiler.unlink()
                    self.compiler.write_bytes(b'changed while reading')
                    self.compiler.chmod(0o700)
                return data
            self.compiler.write_bytes(b'synthetic compiler, never executed')
            with self.subTest(replacement=replacement), patch.object(self.m, 'verify_source'), patch.object(os, 'read', side_effect=read), self.assertRaises(ValueError):
                self.verify()

    def test_cli_does_not_expose_raw_inputs_or_run_artifacts(self):
        tool = Path(__file__).resolve().parents[1] / 'check-policy-provenance-candidate.py'
        command = [os.sys.executable, str(tool), str(self.transcript), '--repo-root', str(self.root),
                   '--compiler', str(self.compiler), '--artifact', str(self.artifact),
                   '--source-commit', SOURCE, '--compiler-sha256', self.expected['compiler_sha256'],
                   '--artifact-sha256', self.expected['artifact_sha256']]
        result = subprocess.run(command, capture_output=True, text=True, timeout=10)
        self.assertEqual(result.returncode, 2)
        self.assertEqual(result.stdout, '')
        self.assertNotIn(str(self.root), result.stderr)
        self.assertNotIn('synthetic compiler', result.stderr)
