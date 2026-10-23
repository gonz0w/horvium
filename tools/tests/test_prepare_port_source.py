# SPDX-License-Identifier: BSD-3-Clause
"""Source extraction refuses path traversal and archive link/special objects."""
import importlib.util
import io
import os
from pathlib import Path
import tarfile
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location(
    'prepare_port_source', Path(__file__).resolve().parents[1] / 'prepare-port-source.py')
SOURCE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(SOURCE)


class SourceArchiveTests(unittest.TestCase):
    def test_go_environment_ignores_ambient_fetch_and_source_overrides(self):
        hostile = {'GOFLAGS': '-modfile=/other.mod -overlay=/other.json',
                   'GOPRIVATE': '*', 'GONOPROXY': '*', 'GOINSECURE': '*',
                   'GOROOT': '/other-sdk', 'GOVCS': '*:all',
                   'HTTPS_PROXY': 'https://unexpected.invalid',
                   'GIT_CONFIG_COUNT': '1', 'GOAUTH': 'external-helper'}
        compiler = Path('/pinned/sdk/bin/go')
        with patch.dict(os.environ, hostile):
            env = SOURCE.go_environment(compiler, Path('/cache'), Path('/fresh'))
        self.assertEqual(env['GOROOT'], '/pinned/sdk')
        self.assertEqual(env['GOPROXY'], 'off')
        self.assertEqual(env['GOFLAGS'], '')
        self.assertEqual(env['GOPRIVATE'], '')
        self.assertEqual(env['GONOPROXY'], '')
        self.assertEqual(env['GONOSUMDB'], '')
        self.assertEqual(env['GOINSECURE'], '')
        self.assertEqual(env['GOVCS'], '*:off')
        self.assertEqual(env['GOTOOLCHAIN'], 'local')
        for name in ('HTTPS_PROXY', 'GIT_CONFIG_COUNT', 'GOAUTH'):
            self.assertNotIn(name, env)

    def archive(self, name, kind=tarfile.REGTYPE):
        data = io.BytesIO()
        with tarfile.open(fileobj=data, mode='w') as archive:
            entry = tarfile.TarInfo(name)
            entry.type = kind
            entry.linkname = 'outside'
            archive.addfile(entry)
        data.seek(0)
        return tarfile.open(fileobj=data, mode='r:')

    def test_regular_source_is_admitted(self):
        with self.archive('cmd/horvium/main.go') as archive:
            self.assertEqual(len(SOURCE.safe_members(archive)), 1)

    def test_external_paths_and_nonregular_objects_are_refused(self):
        for name, kind in [('../escape', tarfile.REGTYPE), ('/escape', tarfile.REGTYPE),
                           ('link', tarfile.SYMTYPE), ('hardlink', tarfile.LNKTYPE),
                           ('device', tarfile.CHRTYPE), ('pipe', tarfile.FIFOTYPE)]:
            with self.subTest(name=name, kind=kind), self.archive(name, kind) as archive:
                with self.assertRaises(ValueError):
                    SOURCE.safe_members(archive)
