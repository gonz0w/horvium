#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Offline runner-input verification; never establishes native acceptance."""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import stat
import subprocess

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('provenance_consumer', Path(__file__).with_name('check-policy-startup-provenance.py'))
consumer = importlib.util.module_from_spec(spec)
spec.loader.exec_module(consumer)


class IncompleteDiagnostic(ValueError):
    """Retain validated scalar evidence without returning runner success."""
    def __init__(self, evidence):
        super().__init__('required diagnostic provenance is incomplete')
        self.evidence = evidence


def git(root, *arguments):
    try:
        return subprocess.run(['git', *arguments], cwd=root, text=True,
                              capture_output=True, check=True, timeout=30).stdout.strip()
    except (OSError, UnicodeError, subprocess.SubprocessError) as error:
        raise ValueError('candidate Git verification failed') from error


def verify_source(root, source):
    consumer.digest(source, 40)
    if Path(git(root, 'rev-parse', '--show-toplevel')).resolve() != root.resolve():
        raise ValueError('exact repository root required')
    if git(root, 'rev-parse', 'HEAD') != source or git(root, 'status', '--porcelain', '--untracked-files=all'):
        raise ValueError('clean expected source HEAD required')
    lines = git(root, 'log', '-1', '--format=%G?%n%an <%ae>%n%B').splitlines()
    if len(lines) < 3 or lines[0] != 'G' or not any(
            line.casefold() == ('Signed-off-by: ' + lines[1]).casefold() for line in lines[2:]):
        raise ValueError('verified source signature and author DCO required')


def file_identity(info):
    return (info.st_dev, info.st_ino, info.st_size, info.st_mtime_ns,
            info.st_ctime_ns, info.st_mode, info.st_nlink)


def read_verified_file(path, cap, *, executable=False, retain=False):
    """Hash one stable opened regular file; never execute it or follow its leaf."""
    fd = None
    try:
        fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
        before = os.fstat(fd)
        if not stat.S_ISREG(before.st_mode) or not 0 < before.st_size <= cap or (
                executable and not before.st_mode & 0o111):
            raise ValueError('bounded regular candidate file required')
        digest, chunks, count = hashlib.sha256(), [], 0
        while True:
            chunk = os.read(fd, min(65536, cap + 1 - count))
            if not chunk:
                break
            count += len(chunk)
            if count > cap:
                raise ValueError('candidate file exceeds input bound')
            digest.update(chunk)
            if retain:
                chunks.append(chunk)
        if count != before.st_size or file_identity(before) != file_identity(os.fstat(fd)) or (
                file_identity(before) != file_identity(os.stat(path, follow_symlinks=False))):
            raise ValueError('candidate file changed during verification')
        return digest.hexdigest(), b''.join(chunks) if retain else None
    except OSError as error:
        raise ValueError('candidate file cannot be verified safely') from error
    finally:
        if fd is not None:
            os.close(fd)


def verify_candidate(root, transcript, compiler, artifact, *, source_commit,
                     compiler_sha256, artifact_sha256):
    for value, length in ((source_commit, 40), (compiler_sha256, 64), (artifact_sha256, 64)):
        consumer.digest(value, length)
    verify_source(root, source_commit)
    # Development input bounds only; these do not change any native time/output
    # allowance. Expected digests come from the independently reviewed build.
    actual_compiler, _ = read_verified_file(compiler, 128 << 20, executable=True)
    actual_artifact, _ = read_verified_file(artifact, 128 << 20, executable=True)
    if actual_compiler != compiler_sha256 or actual_artifact != artifact_sha256:
        raise ValueError('opened compiler/artifact differs from expected candidate')
    transcript_sha256, data = read_verified_file(transcript, 128 << 10, retain=True)
    try:
        text = data.decode('utf-8')
    except UnicodeError as error:
        raise ValueError('valid scalar transcript encoding required') from error
    result = consumer.validate(text, source_commit=source_commit,
                               compiler_sha256=compiler_sha256, artifact_sha256=artifact_sha256)
    verify_source(root, source_commit)
    result['transcript_sha256'] = transcript_sha256
    if not result['diagnostic_complete']:
        raise IncompleteDiagnostic(result)
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('transcript', type=Path)
    parser.add_argument('--repo-root', type=Path, default=ROOT)
    parser.add_argument('--compiler', type=Path, required=True)
    parser.add_argument('--artifact', type=Path, required=True)
    parser.add_argument('--source-commit', required=True)
    parser.add_argument('--compiler-sha256', required=True)
    parser.add_argument('--artifact-sha256', required=True)
    args = parser.parse_args()
    try:
        result = verify_candidate(args.repo_root, args.transcript, args.compiler, args.artifact,
                                  source_commit=args.source_commit,
                                  compiler_sha256=args.compiler_sha256, artifact_sha256=args.artifact_sha256)
    except IncompleteDiagnostic as error:
        print(json.dumps(error.evidence, separators=(',', ':')))
        return 1
    except ValueError as error:
        parser.exit(2, 'Invalid candidate: ' + str(error) + '\n')
    print(json.dumps(result, separators=(',', ':')))
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
