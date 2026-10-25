#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Run proportional portable review with exact-input deep evidence reuse."""
import argparse
import hashlib
import importlib.util
import json
import os
import platform
import re
import signal
import shutil
import stat
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


# Only these exact validation-tooling/policy files can join the finite route.
# This allowlist does not attest that a supplied scope received separate review.
TOOLING_PATHS = frozenset({
    'AGENTS.md', 'PROJECT.md', 'CONTRIBUTING.md',
    'docs/engineering/github-workflow.md', 'docs/engineering/local-review.md',
    'docs/compatibility/matrix.md',
    'tools/review-plan.py', 'tools/review-gate.py',
    'tools/check-temper-source.py', 'tools/temper-regressions.py',
    'tools/tests/test_review_plan.py', 'tools/tests/test_review_gate.py',
    'tools/tests/test_temper_review_scope.py', 'tools/tests/test_temper_gate_dispatch.py',
    'tools/tests/test_temper_scope_binding.py', 'tools/tests/test_temper_bound_dispatch.py',
    'tools/tests/test_temper_source.py',
})
REGRESSION_MODULES = ('test_review_plan', 'test_review_gate', 'test_temper_review_scope',
                      'test_temper_gate_dispatch', 'test_temper_scope_binding',
                      'test_temper_bound_dispatch', 'test_temper_source')
SCOPE_COMMANDS = ['metadata', 'temper-source', 'review-regressions']


def changed_entries(root, base, source):
    """Complete Git paths, modes and blobs; renames are deletion plus addition."""
    data = subprocess.check_output(['git', 'diff', '--raw', '--no-abbrev', '--no-renames',
                                    '-z', base, source], cwd=root).split(b'\0')
    entries = []
    for index in range(0, len(data) - 1, 2):
        metadata, path = data[index:index + 2]
        old_mode, new_mode, old_blob, new_blob, status = metadata.decode('ascii').removeprefix(':').split()
        entries.append(dict(path=path.decode('utf-8'), old_mode=old_mode, new_mode=new_mode,
                            old_blob=old_blob, new_blob=new_blob, status=status))
    return sorted(entries, key=lambda entry: entry['path'])


def strict_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError('duplicate scope key')
        result[key] = value
    return result


def invalid_constant(value):
    raise ValueError('nonstandard JSON constant: ' + value)


def scope_binding(root, path, record):
    """Admit external, bounded data only; invalid/missing scope stays conservative.

    An independent reviewer supplies the exact binding after examining the
    candidate. This developer-controlled file is not a signed attestation.
    """
    if not path:
        return None
    try:
        path = Path(path)
        # The candidate cannot carry its own exemption. Reject link components
        # before reading, including a link resolving outside the candidate.
        if any(parent.is_symlink() for parent in (path, *path.parents)):
            return None
        path = path.resolve(strict=True)
        if path.is_relative_to(root.resolve()) or not stat.S_ISREG(path.stat().st_mode) or path.stat().st_size > 1 << 20:
            return None
        raw = path.read_bytes()
        if len(raw) > 1 << 20:
            return None
        binding = json.loads(raw, object_pairs_hook=strict_object, parse_constant=invalid_constant)
        required = {'schema', 'route', 'base_commit', 'source_commit', 'source_tree',
                    'changed_entries', 'commands', 'review_reference'}
        if (type(binding) is not dict or set(binding) != required or
                type(binding['schema']) is not int or binding['schema'] != 1 or
                binding['route'] != 'temper-validation-tooling-v1' or
                binding['commands'] != SCOPE_COMMANDS or
                type(binding['review_reference']) is not str or
                not re.fullmatch(r'https://github\.com/gonz0w/horvium/issues/424#issuecomment-[0-9]+', binding['review_reference'])):
            return None
        for key in ('base_commit', 'source_commit', 'source_tree'):
            if binding[key] != record[key] or not re.fullmatch('[0-9a-f]{40}', binding[key]):
                return None
        entries = changed_entries(root, record['base_commit'], record['source_commit'])
        if not entries or binding['changed_entries'] != entries:
            return None
        seen = set()
        for entry in entries:
            if set(entry) != {'path', 'old_mode', 'new_mode', 'old_blob', 'new_blob', 'status'}:
                return None
            name = entry['path']
            if (type(name) is not str or name in seen or
                    name not in TOOLING_PATHS | PLAN.TEMPER_PATHS and not PLAN.documentation_path(name)):
                return None
            seen.add(name)
            # Neither executables, links nor submodules can enter as metadata.
            if entry['old_mode'] not in {'000000', '100644'} or entry['new_mode'] not in {'000000', '100644'}:
                return None
            if entry['status'] not in {'A', 'M', 'D'} or any(
                    not re.fullmatch('[0-9a-f]{40}', entry[key]) for key in ('old_blob', 'new_blob')):
                return None
        return {'route': binding['route'], 'sha256': hashlib.sha256(raw).hexdigest(),
                'review_reference': binding['review_reference']}
    except (OSError, ValueError, KeyError, TypeError):
        return None


def metadata_fingerprint(root, binding):
    """Bind actual metadata tools/checkers/tests and relevant environment, no Go.

    Admission/authenticity is external. These identities detect input drift; they
    do not install tools or turn an unknown binary into an admitted one.
    """
    baseline_python = root / '.tools/venv/bin/python3'
    invoked_python = Path(sys.executable)
    # Parent aliases may name the same candidate (for example a BSD home alias).
    # Keep the entry name and venv boundary; resolving the executable itself
    # would also admit unrelated links to the same underlying base Python.
    if (invoked_python.name != baseline_python.name or
            invoked_python.parent.resolve() != baseline_python.parent.resolve() or
            Path(sys.prefix).resolve() != (root / '.tools/venv').resolve() or
            not baseline_python.is_file() or not os.access(baseline_python, os.X_OK)):
        raise ValueError('metadata review requires the pinned candidate venv interpreter')
    files = {'python': str(Path(sys.executable).resolve()), 'recipe_shell': '/bin/sh'}
    for name in ('git', 'make', 'sh'):
        found = shutil.which(name)
        if not found:
            raise ValueError('required metadata tool is unavailable: ' + name)
        files[name] = str(Path(found).resolve())
    files['gitleaks'] = str(root / '.tools/bin/gitleaks')
    inputs = {name: PLAN.digest(Path(path)) for name, path in files.items()}
    inputs['venv_config'] = PLAN.digest(root / '.tools/venv/pyvenv.cfg')
    for name in sorted(TOOLING_PATHS | PLAN.TEMPER_PATHS | {
            'Makefile', '.gitleaks.toml', '.python-version', 'tools/check-repo.py',
            'tools/check-repo.sh', 'tools/check-dco.py', 'tools/owned-process.py',
            'tools/pins.json', 'tools/requirements-ci.txt'}):
        inputs[name] = PLAN.digest(root / name)
    yaml = importlib.util.find_spec('yaml')
    if yaml is None or not yaml.origin:
        raise ValueError('pinned PyYAML is unavailable')
    for path in sorted(Path(yaml.origin).parent.rglob('*')):
        if path.is_file() and '__pycache__' not in path.parts:
            inputs['yaml/' + path.relative_to(Path(yaml.origin).parent).as_posix()] = PLAN.digest(path)
    inputs['scope'] = binding
    inputs['python_version'] = sys.version
    inputs['host'] = [platform.system(), platform.machine(), platform.release()]
    inputs['environment_digest'] = hashlib.sha256(json.dumps({name: value for name, value in os.environ.items()
        if name in {'PATH', 'LANG', 'LC_ALL', 'LC_CTYPE', 'GIT_CONFIG', 'GIT_CONFIG_GLOBAL', 'GIT_CONFIG_SYSTEM',
                    'GIT_CONFIG_NOSYSTEM', 'GIT_EXEC_PATH'} or name.startswith(('PYTHON', 'GITLEAKS_', 'GIT_CONFIG_'))},
        sort_keys=True).encode()).hexdigest()
    # Hashes only: no personal tool paths or arbitrary environment values in receipts.
    return hashlib.sha256(json.dumps(inputs, sort_keys=True).encode()).hexdigest(), inputs


def regression_result(path):
    markers = re.findall(r'(?m)^REGRESSION_RESULT ([^\n]+)$', path.read_text())
    if len(markers) != 1:
        raise ValueError('complete affected regression result required')
    result = json.loads(markers[0], object_pairs_hook=strict_object, parse_constant=invalid_constant)
    if (set(result) != {'modules', 'tests_run', 'failures', 'errors', 'skipped', 'successful'} or
            result['modules'] != list(REGRESSION_MODULES) or type(result['tests_run']) is not int or
            result['tests_run'] <= 0 or result['successful'] is not True or
            any(type(result[key]) is not int or result[key] != 0 for key in ('failures', 'errors', 'skipped'))):
        raise ValueError('complete affected regression result required')
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--full', action='store_true')
    parser.add_argument('--scope', type=Path, default=os.environ.get('REVIEW_SCOPE'),
                        help='external independently reviewed exact-source scope binding')
    args = parser.parse_args()
    title = os.environ.get('PR_TITLE', '')
    if not title:
        raise ValueError('set PR_TITLE to the actual proposed PR title')
    record = identity(ROOT, os.environ.get('BASE_REF', 'origin/main'), title)
    subprocess.run([sys.executable, str(ROOT / 'tools/check-dco.py'), record['base_commit'], record['source_commit']], cwd=ROOT, check=True)
    paths = PLAN.changed_paths(ROOT, record['base_commit'], record['source_commit'])
    binding = None if args.full else scope_binding(ROOT, args.scope, record)
    graph = PLAN.package_graph(ROOT) if any((p.startswith(('internal/', 'cmd/')) for p in paths)) and (not args.full) else {}
    plan = PLAN.select_plan(paths, graph, args.full)
    if binding:
        plan = {'mode': 'temper-tooling', 'packages': [], 'native_review_required': False,
                'reason': 'externally reviewed exact-source validation-tooling scope'}
    metadata = plan['mode'] in {'temper', 'temper-tooling'}
    if not metadata:
        os.environ.update(PLAN.go_environment(ROOT))
    fingerprint, inputs = PLAN.input_fingerprint(ROOT, record['source_commit'])
    environment, metadata_inputs = metadata_fingerprint(ROOT, binding) if metadata else (PLAN.environment_fingerprint(ROOT), None)
    key = {'mode': plan['mode'], 'packages': plan['packages'], 'inputs_digest': fingerprint, 'environment_digest': environment}
    reused = None if args.full or metadata or plan['mode'] == 'documentation' else previous_evidence(ROOT, record, key)
    runs = ROOT / '.tools/review-runs'
    runs.mkdir(parents=True, exist_ok=True, mode=0o700)
    directory = Path(tempfile.mkdtemp(prefix='', dir=runs))
    record.update(schema=2, qualified=False, qualification_scope='affected validation-tooling review only' if binding else 'Temper source metadata review only' if metadata else 'portable review only', host_os=platform.system(), host_arch=platform.machine(), started_at=time.time(), command=['make', 'check', 'secrets'] if metadata else ['make', 'check', 'lint', 'test', 'secrets'], plan=plan, changed_paths=paths, verification_inputs=inputs, deep_key=key, full_requested=args.full)
    record.update(scope_binding=binding, metadata_inputs=metadata_inputs)
    OWNED.write_record(directory / 'result.json', record)

    def interrupted(_signum, _frame):
        raise InterruptedError('local review interrupted')
    signal.signal(signal.SIGTERM, interrupted)
    signal.signal(signal.SIGINT, interrupted)
    print(f"Local {plan['mode']} review {record['source_commit']}; logs: {directory}", flush=True)
    try:
        run_step(record['command'], directory / 'checks.log', record, directory)
        if metadata:
            run_step([sys.executable, str(ROOT / 'tools/check-temper-source.py')], directory / 'temper-source.log', record, directory)
            if binding:
                run_step([sys.executable, str(ROOT / 'tools/temper-regressions.py')], directory / 'regressions.log', record, directory)
                record['regression_result'] = regression_result(directory / 'regressions.log')
                record['regressions_log_sha256'] = PLAN.digest(directory / 'regressions.log')
            record['temper_source_log_sha256'] = PLAN.digest(directory / 'temper-source.log')
            record['deep_evidence'] = {'executed': False, 'reason': 'metadata/affected validation-tooling scope; no product deep qualification'}
        elif plan['mode'] == 'documentation':
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
        if binding and scope_binding(ROOT, args.scope, record) != binding:
            raise ValueError('review scope changed during review')
        final_environment = metadata_fingerprint(ROOT, binding)[0] if metadata else PLAN.environment_fingerprint(ROOT)
        if PLAN.input_fingerprint(ROOT, record['source_commit'])[0] != fingerprint or final_environment != key['environment_digest']:
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
