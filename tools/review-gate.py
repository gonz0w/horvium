#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Run all portable review checks locally against a clean signed contributor range."""
import importlib.util
import json
import os
import platform
import re
import signal
import subprocess
import sys
import tempfile
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SPEC = importlib.util.spec_from_file_location('owned_process', ROOT / 'tools/owned-process.py')
OWNED = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(OWNED)


def validate_title(title):
    if not re.fullmatch(r'(feat|fix|docs|test|refactor|perf|build|ci|chore|revert)(\([a-z0-9._/-]+\))?!?: .{3,100}', title):
        raise ValueError('PR title must use Conventional Commit format')


def git(root, *args):
    return subprocess.check_output(['git', *args], cwd=root, text=True).strip()


def identity(root, base_ref, title):
    validate_title(title)
    if git(root, 'status', '--porcelain'):
        raise ValueError('commit changes and use a clean worktree before review')
    source = git(root, 'rev-parse', '--verify', 'HEAD^{commit}')
    base = git(root, 'rev-parse', '--verify', '--end-of-options', base_ref + '^{commit}')
    if subprocess.run(['git', 'merge-base', '--is-ancestor', base, source], cwd=root).returncode:
        raise ValueError('review base must be an ancestor; reconcile main before review')
    commits = git(root, 'rev-list', f'{base}..{source}').splitlines()
    if not commits:
        raise ValueError('no contributor commits in review range')
    for commit in commits:
        result = subprocess.run(['git', 'verify-commit', commit], cwd=root, capture_output=True)
        if result.returncode:
            raise ValueError(f'commit signature verification failed: {commit}')
    return {'source_commit': source, 'base_commit': base,
            'source_tree': git(root, 'rev-parse', 'HEAD^{tree}'),
            'title': title, 'signed_commits': commits}


def main():
    title = os.environ.get('PR_TITLE', '')
    if not title:
        raise ValueError('set PR_TITLE to the actual proposed PR title')
    record = identity(ROOT, os.environ.get('BASE_REF', 'origin/main'), title)
    subprocess.run([sys.executable, str(ROOT / 'tools/check-dco.py'),
                    record['base_commit'], record['source_commit']], cwd=ROOT, check=True)
    runs = ROOT / '.tools/review-runs'
    runs.mkdir(parents=True, exist_ok=True, mode=0o700)
    directory = Path(tempfile.mkdtemp(prefix='', dir=runs))
    record.update(qualified=False, host_os=platform.system(), host_arch=platform.machine(),
                  started_at=time.time(), command=['make', 'check', 'lint', 'test', 'secrets', 'go-deep-check'])
    OWNED.write_record(directory / 'result.json', record)
    def interrupted(_signum, _frame):
        raise InterruptedError('local review interrupted')
    signal.signal(signal.SIGTERM, interrupted)
    signal.signal(signal.SIGINT, interrupted)
    process = None
    print(f'Local review {record["source_commit"]}; logs: {directory}', flush=True)
    try:
        with (directory / 'checks.log').open('w') as log:
            process = OWNED.OwnedProcess(record['command'], stdout=log, stderr=subprocess.STDOUT)
            record['supervisor_session'] = process.directory.name
            OWNED.write_record(directory / 'result.json', record)
            result = process.wait()
            process.stop()
            record['supervisor'] = process.record()
            if result:
                raise ValueError('local checks failed; inspect checks.log')
        if identity(ROOT, record['base_commit'], title) != {key: record[key] for key in
                ('source_commit', 'base_commit', 'source_tree', 'title', 'signed_commits')}:
            raise ValueError('source changed during review; rerun on the new commit')
        record['qualified'] = True
    except BaseException as error:
        record['failure'] = type(error).__name__ + ': ' + str(error)
        raise
    finally:
        try:
            if process:
                process.stop()
        except Exception as error:
            record.update(qualified=False, cleanup_failure=str(error))
            raise
        finally:
            record['completed_at'] = time.time()
            OWNED.write_record(directory / 'result.json', record)
    print(f'PASS: complete local review and teardown; evidence {directory / "result.json"}', flush=True)


if __name__ == '__main__':
    main()
