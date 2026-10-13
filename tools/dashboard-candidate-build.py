#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Inactive host candidate builder; no CLI, VM, native execution or acceptance.

The future reviewed runner must retain this record and verify the transferred
artifacts against it. Local review files are developer-controlled evidence, not
independent attestation. No transcript is an input to candidate preparation.
"""
import hashlib
import importlib.util
import io
import json
import platform
import subprocess
import sys
import tarfile
import tempfile
from pathlib import Path


def load(name):
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


review = load('review-gate')
files = load('check-policy-provenance-candidate')
contract = load('dashboard-resource-evidence')
ARTIFACT_FILES = {'management': 'horvium', 'helper': 'reader-fixture.test', 'fixture': 'dashboard-native.test'}
REQUIRED = ['make', 'check', 'lint', 'test', 'secrets']


def read_json(path):
    sha, data = files.read_verified_file(path, 1 << 20, retain=True)
    try:
        result = contract.load_json(data.decode('utf-8'))
        contract.require(type(result) is dict, 'record object required')
        return result, sha
    except (UnicodeError, json.JSONDecodeError) as error:
        raise ValueError('bounded JSON record required') from error


def source_identity(root, source_commit, base_commit, pr_title):
    for value in (source_commit, base_commit):
        contract.digest(value, 40)
    files.verify_source(root, source_commit)
    identity = review.identity(root, base_commit, pr_title)
    contract.require(identity['source_commit'] == source_commit and identity['base_commit'] == base_commit,
                     'exact candidate source/base required')
    # The existing gate verifies every signature; the existing DCO policy checks
    # parsed author trailers on every contributor, not only the final commit.
    try:
        subprocess.run([sys.executable, str(root / 'tools/check-dco.py'), base_commit, source_commit],
                       cwd=root, capture_output=True, check=True, timeout=30)
    except (OSError, subprocess.SubprocessError) as error:
        raise ValueError('candidate contributor DCO verification failed') from error
    return identity


def ancestor(root, older, newer):
    contract.digest(older, 40)
    contract.digest(newer, 40)
    return subprocess.run(['git', 'merge-base', '--is-ancestor', older, newer], cwd=root,
                          stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=30).returncode == 0


def stopped(record):
    return (type(record) is dict and record.get('completed') is True
            and record.get('group_cleaned') is True and type(record.get('worker_returncode')) is int
            and record['worker_returncode'] == 0)


def review_directory(root, directory):
    directory = Path(directory).absolute()
    contract.require(directory.parent == root / '.tools/review-runs', 'owned original review directory required')
    return review.OWNED.private_directory(directory)


def verify_review(root, directory, *, source_commit, base_commit, pr_title):
    root = Path(root).resolve()
    directory = review_directory(root, directory)
    identity = source_identity(root, source_commit, base_commit, pr_title)
    record, record_sha = read_json(directory / 'result.json')
    contract.require(type(record.get('schema')) is int and record['schema'] == 2
                     and record.get('qualified') is True, 'completed schema-2 portable review required')
    contract.require(all(contract.exact(record.get(k), v) for k, v in identity.items())
                     and contract.exact(record.get('command'), REQUIRED), 'candidate review identity/command mismatch')
    supervisors = record.get('supervisors')
    contract.require(type(supervisors) is list and bool(supervisors) and all(stopped(s) for s in supervisors),
                     'review supervisor teardown unverified')
    checks_sha, _ = files.read_verified_file(directory / 'checks.log', 16 << 20)
    contract.require(checks_sha == record.get('checks_log_sha256'), 'original required-check log mismatch')
    fingerprint, inputs = review.PLAN.input_fingerprint(root, source_commit)
    key = record.get('deep_key')
    contract.require(type(key) is dict and key.get('inputs_digest') == fingerprint
                     and contract.exact(record.get('verification_inputs'), inputs)
                     and key.get('environment_digest') == review.PLAN.environment_fingerprint(root),
                     'current review inputs/toolchain mismatch')
    plan = record.get('plan')
    contract.require(type(plan) is dict and plan.get('mode') in ('full', 'focused', 'documentation')
                     and contract.exact(key.get('mode'), plan['mode'])
                     and contract.exact(key.get('packages'), plan.get('packages')), 'review plan mismatch')
    paths = review.PLAN.changed_paths(root, base_commit, source_commit)
    graph = review.PLAN.package_graph(root) if any(p.startswith(('internal/', 'cmd/')) for p in paths) and record.get('full_requested') is False else {}
    contract.require(type(record.get('full_requested')) is bool
                     and contract.exact(record.get('changed_paths'), paths)
                     and contract.exact(plan, review.PLAN.select_plan(paths, graph, record['full_requested'])),
                     'independently selected review plan mismatch')
    deep = record.get('deep_evidence')
    contract.require(type(deep) is dict, 'deep evidence required')
    original_directory, original, original_sha = directory, record, record_sha
    if deep.get('executed') is not True:
        reused = deep.get('reused')
        contract.require(type(reused) is dict, 'original executed deep review required')
        path = Path(reused.get('record', ''))
        contract.require(not path.is_absolute() and path.name == 'result.json'
                         and len(path.parts) == 4 and path.parts[:2] == ('.tools', 'review-runs')
                         and all(p not in ('.', '..') for p in path.parts), 'unsafe original review reference')
        original_directory = review_directory(root, root / path.parent)
        original, original_sha = read_json(original_directory / 'result.json')
        contract.require(original_sha == reused.get('record_sha256')
                         and original.get('source_commit') == reused.get('source_commit')
                         and ancestor(root, original['source_commit'], source_commit), 'original review/source mismatch')
        contract.require(original.get('deep_evidence', {}).get('log_sha256') == reused.get('log_sha256')
                         and contract.exact(original.get('deep_evidence', {}).get('result'), reused.get('result')),
                         'original deep evidence mismatch')
    deep_sha, _ = files.read_verified_file(original_directory / 'deep.log', 4 << 20)
    contract.require(review.PLAN.reusable(original, key, base_commit, original_directory / 'deep.log')
                     and stopped(original.get('deep_evidence', {}).get('supervisor'))
                     and deep_sha == original['deep_evidence']['log_sha256'], 'successful exact-input deep execution required')
    # Reopen the local records after all source/environment work; do not return
    # an earlier digest after a path replacement or edit during verification.
    contract.require(read_json(directory / 'result.json')[1] == record_sha
                     and read_json(original_directory / 'result.json')[1] == original_sha
                     and files.read_verified_file(directory / 'checks.log', 16 << 20)[0] == checks_sha
                     and files.read_verified_file(original_directory / 'deep.log', 4 << 20)[0] == deep_sha,
                     'review changed during verification')
    return {'review_sha256': record_sha, 'checks_log_sha256': checks_sha,
            'deep_record_sha256': original_sha, 'deep_log_sha256': deep_sha,
            'deep_source_commit': original['source_commit']}


def compiler_version(compiler, root):
    try:
        result = subprocess.run([str(compiler), 'version'], cwd=root, env=review.PLAN.go_environment(root),
                                capture_output=True, text=True, check=True, timeout=30)
    except (OSError, UnicodeError, subprocess.SubprocessError) as error:
        raise ValueError('pinned compiler version unavailable') from error
    fields = result.stdout.strip().split()
    contract.require(len(result.stdout) <= 512 and len(fields) == 4 and fields[:2] == ['go', 'version'],
                     'unexpected compiler version projection')
    return fields[2]


def verify_compiler(root):
    pins, pins_sha = read_json(root / 'tools/go-pins.json')
    marker, marker_sha = read_json(root / '.tools/go.sha256.json')
    machine = {'x86_64': 'amd64', 'amd64': 'amd64', 'aarch64': 'arm64', 'arm64': 'arm64'}.get(platform.machine())
    target = platform.system().lower() + '_' + str(machine)
    pin = pins.get('artifacts', {}).get(target)
    contract.require(type(pin) is dict, 'reviewed host compiler pin required')
    contract.digest(pin.get('sha256'))
    compiler = root / '.tools/go/bin/go'
    actual, _ = files.read_verified_file(compiler, 128 << 20, executable=True)
    contract.require(contract.exact(marker, {'artifact': pin['sha256'], 'binary': actual}),
                     'opened compiler differs from existing bootstrap pin record')
    version = compiler_version(compiler, root)
    contract.require(version == pins.get('version') and files.read_verified_file(compiler, 128 << 20, executable=True)[0] == actual,
                     'compiler version/content mismatch')
    return {'go_version': version, 'compiler_sha256': actual, 'pins_sha256': pins_sha, 'marker_sha256': marker_sha}


def run_build(command, log, root, env):
    process = None
    with Path(log).open('xb') as output:
        if command[:2] != ['git', 'archive']:
            # Go compilation is normally silent. This fixed controller header
            # frames even empty compiler output; it is not a success assertion.
            output.write(b'Owned candidate command output follows:\n')
            output.flush()
        try:
            process = review.OWNED.OwnedProcess(command, cwd=root, env=env, stdout=output,
                                               stderr=subprocess.DEVNULL if command[:2] == ['git', 'archive'] else subprocess.STDOUT,
                                               runs=review.OWNED.RUNS, timeout=300)
            status = process.wait()
        finally:
            if process is not None:
                process.stop()
        state = process.record()
        contract.require(type(status) is int and status == 0 and stopped(state), 'owned build failed or cleanup unverified')
        return state


def remove_artifacts(root, directory):
    contract.require(Path(directory).absolute().parent == Path(root).resolve() / '.tools/dashboard-candidates',
                     'owned candidate artifact directory required')
    directory = review.OWNED.private_directory(Path(directory).absolute())
    for name in (*ARTIFACT_FILES.values(), 'source.tar'):
        (directory / name).unlink(missing_ok=True)
    path = directory / 'result.json'
    if path.exists():
        record, _ = read_json(path)
        record['artifacts_removed'] = True
        review.OWNED.write_record(path, record)


def extract_source(archive, directory, expected_digest):
    """Extract only bounded tracked files/directories; never links or devices."""
    sha, payload = files.read_verified_file(archive, 128 << 20, retain=True)
    contract.require(sha == expected_digest, 'original archive changed before extraction')
    with tarfile.open(fileobj=io.BytesIO(payload), mode='r:') as source:
        members = source.getmembers()
        total, seen = 0, set()
        contract.require(0 < len(members) <= 20000, 'bounded original archive required')
        for member in members:
            name = Path(member.name)
            total += member.size
            contract.require(not name.is_absolute() and '..' not in name.parts and name not in seen
                             and (member.isfile() or member.isdir()) and total <= 128 << 20,
                             'bounded regular source archive required')
            seen.add(name)
        source.extractall(directory, members=members, filter='data')


def build_candidate(root, review_directory_path, directory, *, source_commit, base_commit, pr_title):
    """Build one owned set for both releases, never run it or open the VM gate.

    Success retains binaries for a future reviewed consumer. That consumer must
    verify transfer/execution and call remove_artifacts after its owned teardown.
    Failure retains the private record/logs and removes only fixed owned outputs.
    """
    root = Path(root).resolve()
    directory = Path(directory).absolute()
    parent = root / '.tools/dashboard-candidates'
    contract.require(directory.parent == parent, 'owned candidate output directory required')
    parent.mkdir(mode=0o700, exist_ok=True)
    review.OWNED.private_directory(parent)
    try:
        directory.mkdir(mode=0o700)
    except OSError as error:
        raise ValueError('fresh candidate output directory required') from error
    record = {'schema': 1, 'source_commit': source_commit, 'base_commit': base_commit,
              'pr_title': pr_title, 'build_complete': False, 'native_execution': False,
              'artifacts_removed': False, 'builds': [], 'supervisors': []}
    review.OWNED.write_record(directory / 'result.json', record)
    try:
        identity = source_identity(root, source_commit, base_commit, pr_title)
        compiler = verify_compiler(root)
        binding = verify_review(root, review_directory_path, source_commit=source_commit, base_commit=base_commit, pr_title=pr_title)
        env = review.PLAN.go_environment(root, native=True)
        env['GOPROXY'] = 'off'
        archive = directory / 'source.tar'
        record['supervisors'].append(run_build(['git', 'archive', '--format=tar', source_commit], archive, root, env))
        archive_sha, _ = files.read_verified_file(archive, 128 << 20)
        hashes = {}
        go = str(root / '.tools/go/bin/go')
        selections = (('management', ['build', '-trimpath', '-buildvcs=false'], './cmd/horvium'),
                      ('helper', ['test', '-c', '-trimpath', '-buildvcs=false'], './internal/host/freebsd'),
                      ('fixture', ['test', '-c', '-trimpath', '-buildvcs=false'], './cmd/horvium'))
        with tempfile.TemporaryDirectory(prefix='source-', dir=directory) as isolated:
            extract_source(archive, Path(isolated), archive_sha)
            record['original_archive_build'] = True
            module_logs = []
            before_log = directory / 'modules-before.log'
            record['supervisors'].append(run_build([go, 'mod', 'verify'], before_log, Path(isolated), env))
            module_logs.append(files.read_verified_file(before_log, 16 << 20)[0])
            for name, arguments, package in selections:
                filename = ARTIFACT_FILES[name]
                command = [go, *arguments, '-o', str(directory / filename), package]
                state = run_build(command, directory / (name + '.log'), Path(isolated), env)
                record['supervisors'].append(state)
                hashes[name], _ = files.read_verified_file(directory / filename, 128 << 20, executable=True)
                record['builds'].append({'role': name, 'filename': filename, 'sha256': hashes[name],
                                         'log_sha256': files.read_verified_file(directory / (name + '.log'), 16 << 20)[0],
                                         'target': 'freebsd/amd64', 'goamd64': 'v1', 'cgo_enabled': False,
                                         'arguments': ['go', *arguments, '-o', '<owned-artifact>', package]})
                review.OWNED.write_record(directory / 'result.json', record)
            after_log = directory / 'modules-after.log'
            record['supervisors'].append(run_build([go, 'mod', 'verify'], after_log, Path(isolated), env))
            module_logs.append(files.read_verified_file(after_log, 16 << 20)[0])
            record['modules_verified_before_after'] = True
            record['module_log_sha256'] = module_logs
        record['isolated_source_removed'] = True
        contract.require(source_identity(root, source_commit, base_commit, pr_title) == identity
                         and verify_compiler(root) == compiler
                         and verify_review(root, review_directory_path, source_commit=source_commit, base_commit=base_commit, pr_title=pr_title) == binding,
                         'candidate source/compiler/review changed during build')
        contract.require(files.read_verified_file(root / '.tools/go/bin/go', 128 << 20, executable=True)[0] == compiler['compiler_sha256']
                         and files.read_verified_file(archive, 128 << 20)[0] == archive_sha,
                         'opened compiler/archive changed during build')
        for name, filename in ARTIFACT_FILES.items():
            contract.require(files.read_verified_file(directory / filename, 128 << 20, executable=True)[0] == hashes[name],
                             'opened artifact changed during build')
        candidate = {'source_commit': source_commit, 'base_commit': base_commit, 'pr_title': pr_title,
                     'go_version': compiler['go_version'], 'compiler_sha256': compiler['compiler_sha256'],
                     'archive_sha256': archive_sha, 'review_verified': True,
                     'artifacts': {release: dict(hashes) for release in contract.RELEASES}}
        contract.candidate_identity(candidate)
        record.update(candidate=candidate, compiler=compiler, review=binding, build_complete=True)
        review.OWNED.write_record(directory / 'result.json', record)
        return record
    except BaseException as error:
        record.update(build_complete=False, failure=type(error).__name__)
        review.OWNED.write_record(directory / 'result.json', record)
        remove_artifacts(root, directory)
        # Keep command output private; callers get neither source paths nor raw
        # compiler diagnostics through an exception intended for a public report.
        raise ValueError('dashboard candidate preparation failed; inspect private build record') from None
