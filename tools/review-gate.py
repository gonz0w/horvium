#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Run proportional portable review with exact-input deep evidence reuse."""
import argparse
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
PLAN_SPEC = importlib.util.spec_from_file_location('review_plan', ROOT / 'tools/review-plan.py')
PLAN = importlib.util.module_from_spec(PLAN_SPEC)
PLAN_SPEC.loader.exec_module(PLAN)


def validate_title(title):
    if not re.fullmatch('(feat|fix|docs|test|refactor|perf|build|ci|chore|revert)(\\([a-z0-9._/-]+\\))?!?: .{3,100}', title):
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
    return {'source_commit': source, 'base_commit': base, 'source_tree': git(root, 'rev-parse', 'HEAD^{tree}'), 'title': title, 'signed_commits': commits}


def previous_evidence(root, record, key):
    runs = root / '.tools/review-runs'
    if not runs.exists():
        return None
    candidates = sorted(runs.glob('*/result.json'), key=lambda p: p.stat().st_mtime, reverse=True)
    for path in candidates:
        try:
            if path.stat().st_size > 1 << 20:
                continue
            report = json.loads(path.read_text())
            log = path.parent / 'deep.log'
            if not PLAN.reusable(report, key, record['base_commit'], log):
                continue
            if subprocess.run(['git', 'merge-base', '--is-ancestor', report['source_commit'], record['source_commit']], cwd=root, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL).returncode:
                continue
            return {'record': str(path.relative_to(root)), 'record_sha256': PLAN.digest(path), 'source_commit': report['source_commit'], 'log_sha256': PLAN.digest(log), 'result': report['deep_evidence']['result']}
        except (OSError, ValueError, KeyError, TypeError):
            continue
    return None


def run_step(command, log, record, directory):
    process = None
    try:
        with log.open('w') as output:
            process = OWNED.OwnedProcess(command, stdout=output, stderr=subprocess.STDOUT)
            record['active_supervisor_session'] = process.directory.name
            OWNED.write_record(directory / 'result.json', record)
            result = process.wait()
            process.stop()
            supervisor = process.record()
            record.setdefault('supervisors', []).append(supervisor)
            record['supervisor'] = supervisor
            if result or not supervisor.get('group_cleaned'):
                raise ValueError('local checks or teardown failed; inspect ' + log.name)
            return supervisor
    finally:
        if process:
            process.stop()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--full', action='store_true')
    args = parser.parse_args()
    title = os.environ.get('PR_TITLE', '')
    if not title:
        raise ValueError('set PR_TITLE to the actual proposed PR title')
    os.environ.update(PLAN.go_environment(ROOT))
    record = identity(ROOT, os.environ.get('BASE_REF', 'origin/main'), title)
    subprocess.run([sys.executable, str(ROOT / 'tools/check-dco.py'), record['base_commit'], record['source_commit']], cwd=ROOT, check=True)
    paths = PLAN.changed_paths(ROOT, record['base_commit'], record['source_commit'])
    graph = PLAN.package_graph(ROOT) if any((p.startswith(('internal/', 'cmd/')) for p in paths)) and (not args.full) else {}
    plan = PLAN.select_plan(paths, graph, args.full)
    fingerprint, inputs = PLAN.input_fingerprint(ROOT, record['source_commit'])
    key = {'mode': plan['mode'], 'packages': plan['packages'], 'inputs_digest': fingerprint, 'environment_digest': PLAN.environment_fingerprint(ROOT)}
    reused = None if args.full or plan['mode'] == 'documentation' else previous_evidence(ROOT, record, key)
    runs = ROOT / '.tools/review-runs'
    runs.mkdir(parents=True, exist_ok=True, mode=0o700)
    directory = Path(tempfile.mkdtemp(prefix='', dir=runs))
    record.update(schema=2, qualified=False, qualification_scope='portable review only', host_os=platform.system(), host_arch=platform.machine(), started_at=time.time(), command=['make', 'check', 'lint', 'test', 'secrets'], plan=plan, changed_paths=paths, verification_inputs=inputs, deep_key=key, full_requested=args.full)
    OWNED.write_record(directory / 'result.json', record)

    def interrupted(_signum, _frame):
        raise InterruptedError('local review interrupted')
    signal.signal(signal.SIGTERM, interrupted)
    signal.signal(signal.SIGINT, interrupted)
    print(f"Local {plan['mode']} review {record['source_commit']}; logs: {directory}", flush=True)
    try:
        run_step(record['command'], directory / 'checks.log', record, directory)
        if plan['mode'] == 'documentation':
            record['deep_evidence'] = {'executed': False, 'reason': 'no behavioral inputs changed in contributor range'}
        elif reused:
            record['deep_evidence'] = {'executed': False, 'reused': reused}
            print('Reused exact-input deep evidence from ' + reused['source_commit'], flush=True)
        else:
            command = [sys.executable, str(ROOT / 'tools/deep-check.py')]
            command += ['--all'] if plan['mode'] == 'full' else ['--packages', *plan['packages']]
            supervisor = run_step(command, directory / 'deep.log', record, directory)
            text = (directory / 'deep.log').read_text()
            markers = re.findall('(?m)^DEEP_RESULT ([^\\n]+)$', text)
            if len(markers) != 1:
                raise ValueError('complete deep result required')
            result = json.loads(markers[0])
            record['deep_evidence'] = {'executed': True, 'command': command, 'supervisor': supervisor, 'log_sha256': PLAN.digest(directory / 'deep.log'), 'result': result}
        if identity(ROOT, record['base_commit'], title) != {key: record[key] for key in ('source_commit', 'base_commit', 'source_tree', 'title', 'signed_commits')}:
            raise ValueError('source changed during review; rerun on new commit')
        if PLAN.input_fingerprint(ROOT, record['source_commit'])[0] != fingerprint or PLAN.environment_fingerprint(ROOT) != key['environment_digest']:
            raise ValueError('review inputs or toolchain changed during review')
        record['checks_log_sha256'] = PLAN.digest(directory / 'checks.log')
        record['qualified'] = True
    except BaseException as error:
        record['failure'] = type(error).__name__ + ': ' + str(error)
        raise
    finally:
        record['completed_at'] = time.time()
        OWNED.write_record(directory / 'result.json', record)
    print(f"PASS: {plan['mode']} portable review and teardown; evidence {directory / 'result.json'}", flush=True)
if __name__ == '__main__':
    main()
