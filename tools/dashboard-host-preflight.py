#!/usr/bin/env python3
# SPDX-License-Identifier: BSD-3-Clause
"""Inactive candidate/preflight binding; never transfers, executes or accepts native work.

Caller expectations come from the reviewed source and original owned host context,
not guest stdout. Local developer-controlled records are not external attestation.
"""
import hashlib
import json
import io
import math
import re
import subprocess
import tarfile
from pathlib import Path, PurePosixPath

import importlib.util


def load(name):
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


build = load('dashboard-candidate-build')
runner = load('local-qualification')
c = build.contract
PARENT = 'TestNativeDashboardArtifactProvenance'
REMOTE = ('cd /root/work && env HORVIUM_NATIVE_VM=1 .tools/dashboard-native.test '
          '-test.v -test.count=1 -test.timeout=45s -test.run=^' + PARENT + '$')


def owned(root, directory, area):
    directory = Path(directory).absolute()
    c.require(directory.parent == root / '.tools' / area, 'original owned ' + area + ' directory required')
    return build.review.OWNED.private_directory(directory)


def timestamp(value):
    c.require(type(value) in (float, int) and 0 < value <= c.UINT64 and math.isfinite(value),
              'original bounded completion time required')
    return value


def preflight_command(root, lab_directory, port):
    """Describe the fixed existing loopback SSH operation; never execute it."""
    root = Path(root).resolve()
    lab = owned(root, lab_directory, 'lab-runs')
    c.integer(port, 1, 65535)
    return runner.ssh_command(lab / 'fixture-key', lab / 'known_hosts', port) + [REMOTE]


def tracked_tree(root, source_commit):
    c.digest(source_commit, 40)
    try:
        output = subprocess.run(['git', 'ls-tree', '-r', '-z', source_commit], cwd=root,
                                capture_output=True, check=True, timeout=30).stdout
        c.require(0 < len(output) <= 2 << 20 and output.endswith(b'\x00'), 'bounded original Git tree required')
        entries = output[:-1].split(b'\x00')
        c.require(len(entries) <= 20000, 'bounded source entries required')
        result = {}
        for entry in entries:
            metadata, path = entry.split(b'\t', 1)
            mode, kind, blob = metadata.decode('ascii').split(' ')
            name = path.decode('utf-8')
            c.require(mode in ('100644', '100755') and kind == 'blob', 'regular source blobs required')
            c.digest(blob, 40)
            c.require(0 < len(name) <= 512 and name not in result and not PurePosixPath(name).is_absolute()
                      and '..' not in PurePosixPath(name).parts and PurePosixPath(name).as_posix() == name,
                      'canonical original source path required')
            result[name] = (blob, mode == '100755')
        return result
    except (OSError, subprocess.SubprocessError, UnicodeError) as error:
        raise ValueError('original source tree unavailable') from error


def verify_archive(root, path, source_commit):
    expected = tracked_tree(root, source_commit)
    sha, payload = build.files.read_verified_file(path, 128 << 20, retain=True)
    directories = {parent.as_posix() for name in expected for parent in PurePosixPath(name).parents
                   if parent.as_posix() != '.'}
    try:
        with tarfile.open(fileobj=io.BytesIO(payload), mode='r:') as archive:
            c.require(archive.pax_headers.get('comment') == source_commit, 'original Git archive commit header required')
            seen, files, total = set(), set(), 0
            for member in archive:
                name = member.name.rstrip('/') if member.isdir() else member.name
                total += member.size
                c.require(name not in seen and len(seen) < 20000 and total <= 128 << 20,
                          'bounded unique original archive members required')
                seen.add(name)
                if member.isdir():
                    c.require(name in directories, 'unexpected original archive directory')
                    continue
                c.require(member.isfile() and name in expected, 'unexpected/linked original archive file')
                original_blob, executable = expected[name]
                c.require(bool(member.mode & 0o111) == executable, 'original source executable mode mismatch')
                stream = archive.extractfile(member)
                c.require(stream is not None, 'original archive file unavailable')
                with stream:
                    content = stream.read(member.size + 1)
                c.require(len(content) == member.size, 'incomplete original source content')
                blob = hashlib.sha1(b'blob ' + str(len(content)).encode() + b'\x00' + content).hexdigest()
                c.require(blob == original_blob, 'archive bytes differ from signed source blob')
                files.add(name)
            c.require(files == set(expected), 'original source archive incomplete')
    except tarfile.TarError as error:
        raise ValueError('malformed original source archive') from error
    c.require(build.files.read_verified_file(path, 128 << 20)[0] == sha, 'original archive changed during tree verification')
    return sha


def verify_build(root, review_directory, candidate_directory, *, source_commit, base_commit, pr_title):
    """Reobserve original retained outputs, source/review/compiler and build logs."""
    root = Path(root).resolve()
    directory = owned(root, candidate_directory, 'dashboard-candidates')
    record, record_sha = build.read_json(directory / 'result.json')
    c.require(type(record.get('schema')) is int and record['schema'] == 1, 'original builder schema required')
    identity = dict(source_commit=source_commit, base_commit=base_commit, pr_title=pr_title)
    c.require(all(c.exact(record.get(k), v) for k, v in identity.items()), 'independent build identity mismatch')
    source_identity = build.source_identity(root, source_commit, base_commit, pr_title)
    compiler = build.verify_compiler(root)
    reviewed = build.verify_review(root, review_directory, **identity)
    c.require(c.exact(record.get('compiler'), compiler) and c.exact(record.get('review'), reviewed),
              'original compiler/review binding mismatch')
    for key in ('build_complete', 'original_archive_build', 'modules_verified_before_after', 'isolated_source_removed'):
        c.require(record.get(key) is True, 'original candidate preparation incomplete')
    c.require(record.get('native_execution') is False and record.get('artifacts_removed') is False,
              'retained unexecuted candidate artifacts required')
    candidate = record.get('candidate')
    c.candidate_identity(candidate)
    archive_sha = verify_archive(root, directory / 'source.tar', source_commit)
    snapshots = {'source.tar': archive_sha, 'result.json': record_sha}
    hashes = {}
    rows = record.get('builds')
    c.require(type(rows) is list and len(rows) == 3, 'original three controlled builds required')
    for row in rows:
        c.object_keys(row, ('role', 'filename', 'sha256', 'log_sha256', 'target', 'goamd64', 'cgo_enabled', 'arguments'), 'build')
        role = row['role']
        c.require(type(role) is str and role in build.ARTIFACT_FILES and role not in hashes, 'duplicate/unknown build role')
        arguments = (['go', 'build', '-trimpath', '-buildvcs=false'] if role == 'management'
                     else ['go', 'test', '-c', '-trimpath', '-buildvcs=false'])
        arguments += ['-o', '<owned-artifact>', './internal/host/freebsd' if role == 'helper' else './cmd/horvium']
        c.require(row['filename'] == build.ARTIFACT_FILES[role] and row['target'] == 'freebsd/amd64'
                  and row['goamd64'] == 'v1' and row['cgo_enabled'] is False and c.exact(row['arguments'], arguments),
                  'original controlled build selection mismatch')
        actual, _ = build.files.read_verified_file(directory / build.ARTIFACT_FILES[role], 128 << 20, executable=True)
        log, _ = build.files.read_verified_file(directory / (role + '.log'), 16 << 20)
        c.require(actual == row['sha256'] and log == row['log_sha256'], 'original executable/build log mismatch')
        hashes[role] = actual
        snapshots[build.ARTIFACT_FILES[role]] = actual
        snapshots[role + '.log'] = log
    expected = dict(identity, go_version=compiler['go_version'], compiler_sha256=compiler['compiler_sha256'],
                    archive_sha256=archive_sha, review_verified=True,
                    artifacts={release: dict(hashes) for release in c.RELEASES})
    c.require(c.exact(candidate, expected), 'candidate differs from independently observed build inputs')
    modules = [build.files.read_verified_file(directory / name, 16 << 20)[0]
               for name in ('modules-before.log', 'modules-after.log')]
    c.require(c.exact(modules, record.get('module_log_sha256')), 'original module verification logs mismatch')
    supervisors = record.get('supervisors')
    c.require(type(supervisors) is list and len(supervisors) == 6 and all(build.stopped(s) for s in supervisors),
              'original build supervision incomplete')
    for state in supervisors:
        c.require(state.get('reason') == 'exit' and state.get('escalated') is False
                  and 'cleanup_failure' not in state, 'original build completion/cleanup uncertain')
        c.integer(state.get('group'), 1)
        c.digest(state.get('command_sha256'))
        c.require(timestamp(state.get('completed_at')) >= timestamp(state.get('started_at')),
                  'original build completion ordering invalid')
    completed = max(timestamp(s.get('completed_at')) for s in supervisors)
    snapshots.update(zip(('modules-before.log', 'modules-after.log'), modules))
    c.require(c.exact(build.source_identity(root, source_commit, base_commit, pr_title), source_identity)
              and c.exact(build.verify_compiler(root), compiler)
              and c.exact(build.verify_review(root, review_directory, **identity), reviewed),
              'source/compiler/review changed during build revalidation')
    for name, digest in snapshots.items():
        cap = 128 << 20 if name in (*build.ARTIFACT_FILES.values(), 'source.tar') else 16 << 20
        c.require(build.files.read_verified_file(directory / name, cap)[0] == digest, 'original build input changed')
    return {'candidate': candidate, 'record_sha256': record_sha, 'completed_at': completed}


def observation(payload, candidate, release):
    c.require(type(payload) is bytes and 0 < len(payload) <= 65536 and payload.endswith(b'\n'),
              'bounded complete standalone preflight required')
    try:
        lines = payload.decode('utf-8').splitlines()
    except UnicodeError as error:
        raise ValueError('invalid preflight encoding') from error
    c.require(len(lines) == 4 and re.fullmatch(r'=== RUN\s+' + PARENT, lines[0], re.ASCII) is not None
              and re.fullmatch(r'--- PASS: ' + PARENT + r' \([0-9]+\.[0-9]+s\)', lines[2]) is not None
              and lines[3] == 'PASS', 'original standalone preflight execution/pass incomplete')
    marker = re.fullmatch(r'\s*dashboard_artifacts_freebsd_test\.go:[1-9][0-9]*: DASHBOARD_GUEST_ARTIFACTS (.+)', lines[1], re.ASCII)
    c.require(marker is not None, 'one original guest artifact observation required')
    row = c.load_json(marker[1])
    c.object_keys(row, ('artifacts', 'go_version', 'kernel', 'userland', 'zfs', 'physical_bytes'), 'guest preflight')
    c.require(c.exact(row['artifacts'], candidate['artifacts'][release]) and row['go_version'] == candidate['go_version'],
              'guest artifacts/compiler differ from independent host candidate')
    for key in ('kernel', 'userland'):
        c.require(re.fullmatch(re.escape(release) + r'-RELEASE(?:-p[0-9]+)?', c.text(row[key])) is not None,
                  'actual guest release differs from expected release')
    c.text(row['zfs'])
    c.require('\x7f' not in row['zfs'], 'invalid observed ZFS version')
    c.integer(row['physical_bytes'], 1)
    return row


def supervisor_record(root, directory, command, completed):
    directory = owned(root, directory, 'process-runs')
    state, sha = build.read_json(directory / 'state.json')
    c.object_keys(state, ('completed', 'group_cleaned', 'command_sha256', 'group', 'started_at', 'escalated',
                          'worker_returncode', 'reason', 'completed_at'), 'original preflight supervisor')
    command_sha = hashlib.sha256(json.dumps(command).encode()).hexdigest()
    c.require(build.stopped(state) and state['reason'] == 'exit' and state['escalated'] is False
              and state['command_sha256'] == command_sha, 'original command/completion/cleanup mismatch')
    c.integer(state['group'], 1)
    c.require(timestamp(state['started_at']) >= completed and timestamp(state['completed_at']) >= state['started_at'],
              'preflight predates candidate preparation or has invalid completion ordering')
    c.require(not (directory / 'control.sock').exists(), 'original preflight control socket remains')
    return sha


def verify_preflight(root, review_directory, candidate_directory, lab_directory, supervisor_directory,
                     release, port, *, source_commit, base_commit, pr_title):
    """Bind independently owned files without changing them or opening execution gates.

    A future trusted runner supplies original candidate/source/lab/port/supervisor
    context. This cannot authenticate a malicious developer/root guest or prove
    later launches, transfer implementation, full native controls or VM teardown.
    """
    root = Path(root).resolve()
    c.require(type(release) is str and release in c.RELEASES, 'declared preflight release required')
    command = preflight_command(root, lab_directory, port)
    identity = dict(source_commit=source_commit, base_commit=base_commit, pr_title=pr_title)
    original = verify_build(root, review_directory, candidate_directory, **identity)
    lab = owned(root, lab_directory, 'lab-runs')
    path = lab / 'dashboard-preflight.log'
    transcript_sha, payload = build.files.read_verified_file(path, 65536, retain=True)
    row = observation(payload, original['candidate'], release)
    supervisor_sha = supervisor_record(root, supervisor_directory, command, original['completed_at'])
    c.require(c.exact(verify_build(root, review_directory, candidate_directory, **identity), original),
              'candidate context changed during preflight binding')
    c.require(build.files.read_verified_file(path, 65536)[0] == transcript_sha
              and supervisor_record(root, supervisor_directory, command, original['completed_at']) == supervisor_sha,
              'original preflight inputs changed during binding')
    return {'schema': 1, 'candidate': original['candidate'], 'release': release, 'observed': row,
            'candidate_build_sha256': original['record_sha256'], 'transcript_sha256': transcript_sha,
            'supervisor_sha256': supervisor_sha, 'command_sha256': hashlib.sha256(json.dumps(command).encode()).hexdigest(),
            'native_acceptance': False, 'resource_execution': False}
